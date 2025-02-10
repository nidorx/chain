package pubsub

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nidorx/chain/pkg"
	"github.com/segmentio/ksuid"
)

var (
	selfId       = ksuid.New()
	selfIdBytes  = selfId.Bytes() // 20 bytes
	selfIdString = selfId.String()
	directTopic  = "direct:" + selfIdString
	ErrNoAdapter = errors.New("no adapter matches topic to broadcast the message")
)

type Dispatcher interface {
	Dispatch(topic string, message []byte, from string)
}

type DispatcherFuncImpl struct {
	Dispatcher func(topic string, message []byte, from string)
}

func (d *DispatcherFuncImpl) Dispatch(topic string, message []byte, from string) {
	if d.Dispatcher == nil {
		return
	}
	d.Dispatcher(topic, message, from)
}

func DispatcherFunc(d func(topic string, message []byte, from string)) Dispatcher {
	return &DispatcherFuncImpl{Dispatcher: d}
}

// subscription represents the subscriptions that this server has. See pubsub.Subscribe
type subscription struct {
	dispatchers map[Dispatcher]int // incremental dispatcher subscriptions
}

// pubsub Realtime Publisher/Subscriber service.
type pubsub struct {
	adapters           *pkg.WildcardStore[*AdapterConfig]
	subscriptions      map[string]*subscription
	unsubscribeTimers  map[string]*time.Timer
	unsubscribeMutex   sync.Mutex
	subscriptionsMutex sync.RWMutex
}

var p = &pubsub{
	subscriptions:     map[string]*subscription{},
	unsubscribeTimers: map[string]*time.Timer{},
}

// Self get node id
func Self() string {
	return selfIdString
}

func Subscribe(topic string, dispatcher Dispatcher) {
	p.subscriptionsMutex.Lock()
	defer p.subscriptionsMutex.Unlock()
	var sub *subscription
	var exist bool
	if sub, exist = p.subscriptions[topic]; !exist {
		sub = &subscription{dispatchers: map[Dispatcher]int{}}
		p.subscriptions[topic] = sub
		go trySubscribe(topic)
	}
	if _, exist = sub.dispatchers[dispatcher]; !exist {
		sub.dispatchers[dispatcher] = 0
	}
	sub.dispatchers[dispatcher] = sub.dispatchers[dispatcher] + 1
}

// Unsubscribe the dispatchFunc from the pubsub adapter's topic.
func Unsubscribe(topic string, dispatcher Dispatcher) {
	p.subscriptionsMutex.Lock()
	defer p.subscriptionsMutex.Unlock()
	var sub *subscription
	var exist bool
	if sub, exist = p.subscriptions[topic]; !exist {
		return
	}
	if _, exist = sub.dispatchers[dispatcher]; !exist {
		return
	}
	sub.dispatchers[dispatcher] = sub.dispatchers[dispatcher] - 1
	if sub.dispatchers[dispatcher] < 1 {
		delete(sub.dispatchers, dispatcher)
		go scheduleUnsubscribe(topic)
	}
}

// Broadcast broadcasts message on given topic across the whole cluster.
func Broadcast(topic string, message []byte, options ...*Option) (err error) {
	var config *AdapterConfig
	if config = GetAdapter(topic); config == nil {
		return ErrNoAdapter
	}

	if config.Adapter.Name() == "dummy" {
		dispatchMessage(topic, message, selfIdString)
		return
	}

	opts := map[string]any{}
	for k, v := range globalOptions {
		opts[k] = v
	}
	for _, opt := range options {
		opts[opt.key] = opt.value
	}

	msgToSend := message

	// [messageType: byte] [from: 20 bytes] [msgToSend: ...]
	msgToSend = append(append([]byte{byte(MessageTypeBroadcast)}, selfIdBytes...), msgToSend...)

	// Check if we have compression enabled
	if config.DisableCompression == false {
		var compressed []byte
		if compressed, err = compressPayload(msgToSend); err != nil {
			slog.Warn(
				"[chain.pubsub] failed to compress payload",
				slog.Any("error", err),
			)
		} else if len(compressed) < len(msgToSend) {
			// Only use compression if it reduced the size
			msgToSend = compressed
		}
	}

	// Check if we have encryption enabled
	if config.DisableEncryption == false {
		keyring := config.Keyring
		if keyring == nil {
			keyring = globalKeyring
		}
		var encrypted []byte
		if encrypted, err = encryptPayload(keyring, msgToSend); err != nil {
			return errors.Join(errors.New("encryption of message failed"), err)
		}
		msgToSend = encrypted
	}

	if err = config.Adapter.Broadcast(topic, msgToSend, opts); err == nil {
		// local dispatch
		dispatchMessage(topic, message, selfIdString)
	}
	return
}

// DirectBroadcast Broadcasts ServiceMsg on given topic to a given node.
func DirectBroadcast(nodeId string, topic string, message []byte, options ...*Option) error {
	// [messageType: byte] [from: 20 bytes] [message: ...]

	nodeIdK, err := ksuid.Parse(nodeId)
	if err != nil {
		return err
	}

	// [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
	buf := &bytes.Buffer{}
	buf.Write(nodeIdK.Bytes())

	topicNameLen := make([]byte, 4)
	binary.BigEndian.PutUint32(topicNameLen, uint32(len(topic)))
	buf.Write(topicNameLen)

	buf.WriteString(topic)
	buf.Write(message)

	return broadcastMessage(MessageTypeDirectBroadcast, "direct:"+nodeId, buf.Bytes(), options...)
}

// Broadcast broadcasts message on given topic across the whole cluster.
func broadcastMessage(msgType MessageType, topic string, message []byte, options ...*Option) (err error) {
	var config *AdapterConfig
	if config = GetAdapter(topic); config == nil {
		return ErrNoAdapter
	}

	if config.Adapter.Name() == "dummy" {
		dispatchMessage(topic, message, selfIdString)
		return
	}

	opts := map[string]any{}
	for k, v := range globalOptions {
		opts[k] = v
	}
	for _, opt := range options {
		opts[opt.key] = opt.value
	}

	// [messageType: byte] [from: 20 bytes] [message: ...]
	// [messageType: byte] [from: 20 bytes] [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
	buf := &bytes.Buffer{}
	buf.WriteByte(byte(msgType))
	buf.Write(selfIdBytes)
	buf.Write(message)
	msgToSend := buf.Bytes()

	// Check if we have compression enabled
	if !config.DisableCompression {
		var compressed []byte
		if compressed, err = compressPayload(msgToSend); err != nil {
			slog.Warn(
				"[chain.pubsub] failed to compress payload",
				slog.Any("error", err),
			)
		} else if len(compressed) < len(msgToSend) {
			// Only use compression if it reduced the size
			msgToSend = compressed
		}
	}

	// Check if we have encryption enabled
	if !config.DisableEncryption {
		keyring := config.Keyring
		if keyring == nil {
			keyring = globalKeyring
		}
		var encrypted []byte
		if encrypted, err = encryptPayload(keyring, msgToSend); err != nil {
			return errors.Join(errors.New("encryption of message failed"), err)
		}
		msgToSend = encrypted
	}

	err = config.Adapter.Broadcast(topic, msgToSend, opts)
	return
}

// Dispatch used by adapters, process and delivery messages coming from backend (redis, kafka, *MQ), decrypting and
// decompressing if necessary.
func Dispatch(topic string, message []byte) {
	if config := GetAdapter(topic); config != nil {
		// Read the message type
		msgType := MessageType(message[0])

		// Check if the message is encrypted
		if msgType == MessageTypeEncrypt {
			if config.DisableEncryption {
				slog.Error(
					"[chain.pubsub] remote message is encrypted and encryption is not configured",
					slog.String("Topic", topic),
					slog.String("Adapter", config.Adapter.Name()),
				)
				return
			}

			keyring := config.Keyring
			if keyring == nil {
				keyring = globalKeyring
			}
			plain, err := decryptPayload(keyring, message)
			if err != nil {
				slog.Error(
					"[chain.pubsub] could not decrypt remote message",
					slog.Any("Error", err),
					slog.String("Topic", topic),
					slog.String("Adapter", config.Adapter.Name()),
				)
				return
			}

			// Reset message type and buf
			msgType = MessageType(plain[0])
			message = plain
		} else if !config.DisableEncryption {
			slog.Error(
				"[chain.pubsub] encryption is configured but remote message is not encrypted",
				slog.String("Topic", topic),
				slog.String("Adapter", config.Adapter.Name()),
			)
			return
		}

		// Check if we have a compressed message
		if msgType == MessageTypeCompress {
			decompressed, err := decompressPayload(message)
			if err != nil {
				slog.Error(
					"[chain.pubsub] could not decompress remote message",
					slog.Any("Error", err),
					slog.String("Topic", topic),
					slog.String("Adapter", config.Adapter.Name()),
				)
				return
			}

			// Reset message type and buf
			msgType = MessageType(decompressed[0])
			message = decompressed
		}

		// [messageType: byte] [from: 20 bytes] [message: ...]
		// [messageType: byte] [from: 20 bytes] [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
		message = message[1:]

		if len(message) < 20 {
			slog.Error(
				"[chain.pubsub] invalid remote message length",
				slog.String("Topic", topic),
				slog.String("Adapter", config.Adapter.Name()),
			)
			return
		}
		fromBytes := message[:20]

		fromID, err := ksuid.FromBytes(fromBytes)
		if err != nil {
			slog.Error(
				"[chain.pubsub] invalid remote message from",
				slog.Any("Error", err),
				slog.String("Topic", topic),
				slog.String("Adapter", config.Adapter.Name()),
			)
			return
		}
		from := fromID.String()

		// [message: ...]
		// [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
		message = message[20:]

		// Check if is a direct broadcast
		if msgType == MessageTypeDirectBroadcast {
			if topic != directTopic {
				slog.Error(
					"[chain.pubsub] invalid topic for remote direct broadcast message",
					slog.String("Topic", topic),
					slog.String("Adapter", config.Adapter.Name()),
					slog.String("Expected", directTopic),
				)
				return
			}

			// [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
			if len(message) < 25 {
				slog.Error(
					"[chain.pubsub] invalid remote direct broadcast length",
					slog.String("Topic", topic),
					slog.String("Adapter", config.Adapter.Name()),
				)
				return
			}

			toBytes := message[0:20]
			message = message[20:]

			if !bytes.Equal(selfIdBytes, toBytes) {
				slog.Error(
					"[chain.pubsub] invalid remote direct broadcast destination",
					slog.String("Adapter", config.Adapter.Name()),
				)
				return
			}

			// [topicNameLen: uint] [topic: topicNameLen] [message: ...]
			topicNameLen := int(binary.BigEndian.Uint32(message[0:4]))
			message = message[4:]

			if len(message) < topicNameLen {
				slog.Error(
					"[chain.pubsub] invalid remote direct broadcast length",
					slog.String("Adapter", config.Adapter.Name()),
				)
				return
			}
			topic = string(message[:topicNameLen])
			message = message[topicNameLen:]
		} else if msgType != MessageTypeBroadcast {
			slog.Error(
				"[chain.pubsub] invalid remote message type",
				slog.String("Topic", topic),
				slog.String("Adapter", config.Adapter.Name()),
			)
			return
		}

		dispatchMessage(topic, message, from)
	}
}

// LocalBroadcast broadcasts message on given topic only for the current node.
//
// `topic` - The topic to broadcast to, ie: `"users:123"`
// `message` - The payload of the broadcast
func LocalBroadcast(topic string, message []byte) {
	dispatchMessage(topic, message, selfIdString)
}

// SetAdapters configure the adapters topics.
//
// Allows the application to have instances specialized by topics.
//
// ## Example
//
//	SetAdapters([]AdapterConfig{
//		{&RedisAdapter{Addr: "admin.redis-host:6379"}, []string{"admin:*"}},
//		{&RedisAdapter{Addr: "global.redis-host:6379"}, []string{"*"}},
//	})
func SetAdapters(adapters []AdapterConfig) {

	if config := GetAdapter(directTopic); config != nil {
		// direct broadcast
		config.Adapter.Unsubscribe(directTopic)
	}
	defer trySubscribe(directTopic)

	p.adapters = &pkg.WildcardStore[*AdapterConfig]{}
	for _, config := range adapters {
		for _, topic := range config.Topics {
			if err := p.adapters.Insert(topic, &config); err != nil {
				panic(fmt.Sprintf("[chain.pubsub] invalid adapter config. Topic: %s, Error: %s", topic, err.Error()))
			}
		}
	}
}

// GetAdapter Gets the adapter associated with a topic.
func GetAdapter(topic string) *AdapterConfig {
	if p.adapters == nil {
		return nil
	}
	return p.adapters.Match(topic)
}

// trySubscribe subscribe the adapter on the given topic
func trySubscribe(topic string) {
	p.unsubscribeMutex.Lock()
	defer p.unsubscribeMutex.Unlock()
	if timer, exist := p.unsubscribeTimers[topic]; exist {
		delete(p.unsubscribeTimers, topic)
		defer timer.Stop()
	}

	if config := GetAdapter(topic); config != nil {
		config.Adapter.Subscribe(topic)
	}
}

// scheduleUnsubscribe unsubscribe the adapter after 15 seconds
func scheduleUnsubscribe(topic string) {
	p.unsubscribeMutex.Lock()
	if _, exist := p.unsubscribeTimers[topic]; exist {
		p.unsubscribeMutex.Unlock()
		return
	}

	timer := time.NewTimer(time.Second * 15)
	p.unsubscribeTimers[topic] = timer
	p.unsubscribeMutex.Unlock()

	// wait
	<-timer.C

	p.unsubscribeMutex.Lock()
	defer p.unsubscribeMutex.Unlock()

	if _, exist := p.unsubscribeTimers[topic]; !exist {
		// was removed by pubsub.trySubscribe
		return
	}
	delete(p.unsubscribeTimers, topic)

	if config := GetAdapter(topic); config != nil {
		config.Adapter.Unsubscribe(topic)
	}
}

// dispatchMessage deliver the message locally
func dispatchMessage(topic string, message []byte, from string) {
	go func() {
		if from == "" {
			from = selfIdString
		}

		// get subscriptions & dispatchers
		p.subscriptionsMutex.RLock()
		var sub *subscription
		var exist bool
		if sub, exist = p.subscriptions[topic]; !exist {
			p.subscriptionsMutex.RUnlock()
			// if we are still receiving this message, schedule removal
			go scheduleUnsubscribe(topic)
			return
		}

		var dispatchers []Dispatcher
		for dispatchFunc, _ := range sub.dispatchers {
			dispatchers = append(dispatchers, dispatchFunc)
		}
		p.subscriptionsMutex.RUnlock()

		for _, dispatcher := range dispatchers {
			dispatcher.Dispatch(topic, message, from)
		}
	}()
}
