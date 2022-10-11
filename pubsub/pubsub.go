package pubsub

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"github.com/syntax-framework/chain/pkg"
	"sync"
	"time"
)

var (
	selfKSUID    = ksuid.New()
	selfBytes    = selfKSUID.Bytes() // 20 bytes
	selfString   = selfKSUID.String()
	directTopic  = "stx:direct:" + selfString
	ErrNoAdapter = errors.New("no adapter matches topic to broadcast the message")
)

type Dispatcher interface {
	Dispatch(topic string, message any, from string)
}

type DispatcherFuncImpl struct {
	Dispatcher func(topic string, message any, from string)
}

func (d *DispatcherFuncImpl) Dispatch(topic string, message any, from string) {
	if d.Dispatcher == nil {
		return
	}
	d.Dispatcher(topic, message, from)
}

func DispatcherFunc(d func(topic string, message any, from string)) Dispatcher {
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
	return selfString
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
		dispatchMessage(topic, message, selfString)
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

	// [from: 20 bytes] [msgToSend: ...]
	msgToSend = append(selfBytes, msgToSend...)

	// Check if we have compression enabled
	if config.DisableCompression == false {
		var compressed []byte
		if compressed, err = compressPayload(message); err != nil {
			log.Warn().Err(err).Msg(_l("Failed to compress payload"))
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
			log.Error().Err(err).Msg(_l("Encryption of message failed"))
			return err
		}
		msgToSend = encrypted
	}

	if err = config.Adapter.Broadcast(topic, msgToSend, opts); err == nil {
		// local dispatch
		dispatchMessage(topic, message, selfString)
	}
	return
}

// DirectBroadcast Broadcasts ServiceMsg on given topic to a given node.
func DirectBroadcast(to string, topic string, message []byte, options ...Option) {
}

// Dispatch used by adapters, process and delivery messages coming from backend (redis, kafka, *MQ), decrypting and
// decompressing if necessary.
func Dispatch(topic string, message []byte) {
	if config := GetAdapter(topic); config != nil {
		// Read the message type
		msgType := messageType(message[0])

		// Check if the message is encrypted
		if msgType == messageTypeEncrypt {
			if config.DisableEncryption {
				log.Error().
					Str("topic", topic).
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Remote message is encrypted and encryption is not configured"))
				return
			}

			keyring := config.Keyring
			if keyring == nil {
				keyring = globalKeyring
			}
			plain, err := decryptPayload(keyring, message)
			if err != nil {
				log.Error().
					Err(err).
					Str("topic", topic).
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Could not decrypt remote message"))
				return
			}

			// Reset message type and buf
			msgType = messageType(plain[0])
			message = plain[1:]
		} else if config.DisableEncryption == false {
			log.Error().
				Str("topic", topic).
				Str("adapter", config.Adapter.Name()).
				Msg(_l("Encryption is configured but remote message is not encrypted"))
			return
		}

		// Check if we have a compressed message
		if msgType == messageTypeCompress {
			decompressed, err := decompressPayload(message)
			if err != nil {
				log.Error().
					Err(err).
					Str("topic", topic).
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Could not decompress remote message"))
				return
			}

			// Reset message type and buf
			msgType = messageType(decompressed[0])
			message = decompressed[1:]
		}

		// Check if is a direct broadcast
		if msgType == messageTypeDirectBroadcast {
			if topic != directTopic {
				log.Error().
					Str("topic", topic).
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Invalid topic for remote direct broadcast message"))
			}

			// [messageType: byte] [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
			if len(message) < 25 {
				log.Error().
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Invalid remote direct broadcast length"))
				return
			}

			toBytes := message[1:21]
			message = message[21:]

			if !bytes.Equal(selfBytes, toBytes) {
				log.Error().
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Invalid remote direct broadcast destination"))
				return
			}

			// [topicNameLen: uint] [topic: topicNameLen] [message: ...]
			topicNameLen := int(binary.BigEndian.Uint16(message[1:4]))
			message = message[4:]

			if len(message) < topicNameLen {
				log.Error().
					Str("adapter", config.Adapter.Name()).
					Msg(_l("Invalid remote direct broadcast length"))
				return
			}
			topic = string(message[:topicNameLen])
			message = message[topicNameLen:]
		}

		// [from: 20 bytes] [message: ...]
		if len(message) < 20 {
			log.Error().
				Str("topic", topic).
				Str("adapter", config.Adapter.Name()).
				Msg(_l("Invalid remote message length"))
			return
		}
		fromBytes := message[:20]
		message = message[20:]

		fromID, err := ksuid.FromBytes(fromBytes)
		if err != nil {
			log.Error().
				Err(err).
				Str("topic", topic).
				Str("adapter", config.Adapter.Name()).
				Msg(_l(`Invalid remote message from`))
			return
		}
		from := fromID.String()

		dispatchMessage(topic, message, from)
	}
}

// LocalBroadcast broadcasts message on given topic only for the current node.
//
// `topic` - The topic to broadcast to, ie: `"users:123"`
// `message` - The payload of the broadcast
func LocalBroadcast(topic string, message any) {
	dispatchMessage(topic, message, selfString)
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
	p.adapters = &pkg.WildcardStore[*AdapterConfig]{}
	for _, config := range adapters {
		for _, topic := range config.Topics {
			if err := p.adapters.Insert(topic, &config); err != nil {
				log.Panic().Err(err).
					Str("topic", topic).
					Msg(_l("invalid adapter config"))
			}
		}
	}

	// direct broadcast
	Unsubscribe(directTopic, directDispatcher)
	Subscribe(directTopic, directDispatcher)
}

// GetAdapter Gets the adapter associated with a topic.
func GetAdapter(topic string) *AdapterConfig {
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

	if config := GetAdapter(topic); config != nil {
		config.Adapter.Unsubscribe(topic)
	}
}

// dispatchMessage deliver the message locally
func dispatchMessage(topic string, message any, from string) {
	go func() {
		if from == "" {
			from = selfString
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

// dispatchMessage messages sent directly to this node
var directDispatcher = DispatcherFunc(func(topic string, message any, from string) {

})

func _l(msg string) string {
	return "[chain.pubsub] " + msg
}
