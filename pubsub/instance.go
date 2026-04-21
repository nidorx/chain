package pubsub

import (
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/nidorx/chain/pkg"
	"github.com/segmentio/ksuid"
)

// PubSubConfig holds configuration options for a PubSub instance.
type PubSubConfig struct {
	// DispatchWorkers sets the number of worker goroutines for message dispatch.
	// Default: runtime.NumCPU()
	DispatchWorkers int

	// DispatchQueueSize sets the buffer size for the dispatch queue.
	// Default: 1000
	DispatchQueueSize int

	// SelfID allows setting a specific node ID (primarily for testing).
	// If not set, a new KSUID will be generated.
	SelfID *ksuid.KSUID
}

// PubSub is an instance-based Publisher/Subscriber, enabling multi-tenancy
// and isolated testing.
type PubSub struct {
	// adapters holds adapter configurations mapped by topic pattern.
	adapters *pkg.WildcardStore[*AdapterConfig]

	// subscriptions holds topic patterns mapped to their dispatchers.
	subscriptions *pkg.WildcardStore[*subscription]

	// unsubscribeTimers holds pending unsubscribe timers per topic pattern.
	unsubscribeTimers map[string]*time.Timer

	// unsubscribeMutex protects unsubscribeTimers.
	unsubscribeMutex sync.Mutex

	// subscriptionsMutex protects subscriptions.
	subscriptionsMutex sync.RWMutex

	// selfId is this node's unique identifier.
	selfId ksuid.KSUID

	// selfIdBytes is the raw bytes of selfId (20 bytes).
	selfIdBytes []byte

	// selfIdString is the string representation of selfId.
	selfIdString string

	// selfIdMutex protects selfId, selfIdBytes, and selfIdString.
	selfIdMutex sync.RWMutex

	// dispatchQueue channels dispatch jobs to workers.
	dispatchQueue chan dispatchJob

	// workerWg waits for all dispatch workers to finish.
	workerWg sync.WaitGroup

	// metrics collects observability data.
	metrics MetricsCollector

	// config holds the instance configuration.
	config PubSubConfig

	// closed indicates whether this instance has been closed.
	closed bool

	// closeMutex protects shutdown operations.
	closeMutex sync.Mutex
}

// dispatchJob represents a unit of work for dispatch workers.
type dispatchJob struct {
	topic   string
	message []byte
	from    string
}

// New creates a new PubSub instance with the given configuration options.
//
// This is the recommended way to use pubsub for multi-tenancy and testing:
//
//	ps := pubsub.New()
//	ps.Subscribe("user:*", dispatcher)
//	ps.Broadcast("user:123", []byte("hello"))
//
// For backward compatibility, the global functions (Subscribe, Broadcast, etc.)
// continue to work and delegate to Default instance.
func New(opts ...PubSubOption) *PubSub {
	cfg := PubSubConfig{
		DispatchWorkers:   runtime.NumCPU(),
		DispatchQueueSize: 1000,
	}

	// Apply options
	for _, opt := range opts {
		opt(&cfg)
	}

	ps := &PubSub{
		subscriptions:     &pkg.WildcardStore[*subscription]{},
		unsubscribeTimers: map[string]*time.Timer{},
		metrics:           &noopMetrics{},
		config:            cfg,
	}

	// Set node ID
	if cfg.SelfID != nil {
		ps.selfId = *cfg.SelfID
	} else {
		ps.selfId = ksuid.New()
	}
	ps.selfIdBytes = ps.selfId.Bytes()
	ps.selfIdString = ps.selfId.String()

	// Initialize dispatch queue
	ps.dispatchQueue = make(chan dispatchJob, cfg.DispatchQueueSize)

	// Start dispatch workers
	for i := 0; i < cfg.DispatchWorkers; i++ {
		ps.workerWg.Add(1)
		go ps.dispatchWorker()
	}

	return ps
}

// PubSubOption is a functional option for configuring a PubSub instance.
type PubSubOption func(*PubSubConfig)

// WithDispatchWorkers sets the number of dispatch worker goroutines.
func WithDispatchWorkers(count int) PubSubOption {
	return func(cfg *PubSubConfig) {
		cfg.DispatchWorkers = count
	}
}

// WithDispatchQueueSize sets the dispatch queue buffer size.
func WithDispatchQueueSize(size int) PubSubOption {
	return func(cfg *PubSubConfig) {
		cfg.DispatchQueueSize = size
	}
}

// WithSelfID sets the node ID for this PubSub instance.
func WithSelfID(id ksuid.KSUID) PubSubOption {
	return func(cfg *PubSubConfig) {
		cfg.SelfID = &id
	}
}

// Close gracefully shuts down the PubSub instance, stopping all dispatch workers.
func (ps *PubSub) Close() {
	ps.closeMutex.Lock()
	defer ps.closeMutex.Unlock()

	if ps.closed {
		return
	}
	ps.closed = true

	// Close dispatch queue to signal workers to stop
	close(ps.dispatchQueue)

	// Wait for all workers to finish
	ps.workerWg.Wait()

	// Clean up unsubscribe timers
	ps.unsubscribeMutex.Lock()
	for _, timer := range ps.unsubscribeTimers {
		timer.Stop()
	}
	ps.unsubscribeTimers = nil
	ps.unsubscribeMutex.Unlock()
}

// dispatchWorker processes dispatch jobs from the queue.
func (ps *PubSub) dispatchWorker() {
	defer ps.workerWg.Done()

	for job := range ps.dispatchQueue {
		ps.dispatchMessageSync(job.topic, job.message, job.from)
	}
}

// dispatchMessageSync delivers a message to local subscribers synchronously.
func (ps *PubSub) dispatchMessageSync(topic string, message []byte, from string) {
	if from == "" {
		from = ps.getSelfIDString()
	}

	// Get subscriptions & dispatchers
	ps.subscriptionsMutex.RLock()
	subs := ps.subscriptions.MatchAll(topic)
	if len(subs) == 0 {
		ps.subscriptionsMutex.RUnlock()
		// No subscribers - schedule potential topic cleanup
		go ps.scheduleUnsubscribe(topic)
		return
	}

	var dispatchers []Dispatcher
	for _, sub := range subs {
		for dispatchFunc := range sub.dispatchers {
			dispatchers = append(dispatchers, dispatchFunc)
		}
	}
	ps.subscriptionsMutex.RUnlock()

	// Deliver to all dispatchers
	start := time.Now()
	for _, dispatcher := range dispatchers {
		dispatcher.Dispatch(topic, message, from)
	}
	elapsed := time.Since(start)

	ps.metrics.Dispatched(topic, len(dispatchers), elapsed)
}

// getSelfIDBytes returns a copy of selfIdBytes safely.
func (ps *PubSub) getSelfIDBytes() []byte {
	ps.selfIdMutex.RLock()
	defer ps.selfIdMutex.RUnlock()
	b := make([]byte, len(ps.selfIdBytes))
	copy(b, ps.selfIdBytes)
	return b
}

// getSelfIDString returns selfIdString safely.
func (ps *PubSub) getSelfIDString() string {
	ps.selfIdMutex.RLock()
	defer ps.selfIdMutex.RUnlock()
	return ps.selfIdString
}

// getDirectTopic returns the direct broadcast topic for this instance.
func (ps *PubSub) getDirectTopic() string {
	ps.selfIdMutex.RLock()
	defer ps.selfIdMutex.RUnlock()
	return "direct:" + ps.selfIdString
}

// Subscribe registers a dispatcher to receive messages matching a topic pattern.
func (ps *PubSub) Subscribe(topicPattern string, dispatcher Dispatcher) {
	ps.subscriptionsMutex.Lock()
	var sub *subscription

	if sub = ps.subscriptions.Get(topicPattern); sub == nil {
		sub = &subscription{dispatchers: map[Dispatcher]int{}}
		if err := ps.subscriptions.Insert(topicPattern, sub); err != nil {
			slog.Warn(
				"[chain.pubsub] failed to subscribe",
				slog.String("topic", topicPattern),
				slog.Any("error", err),
			)
			ps.subscriptionsMutex.Unlock()
			return
		}
	}

	if _, exist := sub.dispatchers[dispatcher]; !exist {
		sub.dispatchers[dispatcher] = 0
	}
	sub.dispatchers[dispatcher] = sub.dispatchers[dispatcher] + 1
	count := sub.dispatchers[dispatcher]

	// Cancel any pending unsubscribe synchronously while holding lock
	ps.unsubscribeMutex.Lock()
	if timer, exist := ps.unsubscribeTimers[topicPattern]; exist {
		delete(ps.unsubscribeTimers, topicPattern)
		timer.Stop()
	}
	ps.unsubscribeMutex.Unlock()

	ps.subscriptionsMutex.Unlock()

	slog.Debug(
		"[chain.pubsub] subscribe",
		slog.String("topic", topicPattern),
		slog.Int("dispatchers", count),
	)

	// Subscribe adapter (outside lock to avoid deadlock)
	if config := ps.GetAdapter(topicPattern); config != nil {
		config.Adapter.Subscribe(topicPattern)
	}

	ps.metrics.SubscribeCount(topicPattern, count)
}

// Unsubscribe removes a dispatcher from a topic subscription.
func (ps *PubSub) Unsubscribe(topicPattern string, dispatcher Dispatcher) {
	ps.subscriptionsMutex.Lock()
	defer ps.subscriptionsMutex.Unlock()
	var sub *subscription
	var exist bool

	if sub = ps.subscriptions.Get(topicPattern); sub == nil {
		return
	}

	if _, exist = sub.dispatchers[dispatcher]; !exist {
		return
	}
	sub.dispatchers[dispatcher] = sub.dispatchers[dispatcher] - 1
	count := sub.dispatchers[dispatcher]

	if count < 1 {
		delete(sub.dispatchers, dispatcher)

		// Cancel any pending unsubscribe timer first (re-subscribe case)
		ps.unsubscribeMutex.Lock()
		if timer, exists := ps.unsubscribeTimers[topicPattern]; exists {
			delete(ps.unsubscribeTimers, topicPattern)
			timer.Stop()
		}
		ps.unsubscribeMutex.Unlock()

		slog.Debug(
			"[chain.pubsub] unsubscribe (last dispatcher removed, scheduling adapter unsubscribe)",
			slog.String("topic", topicPattern),
		)

		// Schedule adapter unsubscribe after grace period
		go ps.scheduleUnsubscribe(topicPattern)
	} else {
		slog.Debug(
			"[chain.pubsub] unsubscribe",
			slog.String("topic", topicPattern),
			slog.Int("remaining_dispatchers", count),
		)
	}

	ps.metrics.SubscribeCount(topicPattern, count)
}

// Broadcast sends a message to all nodes in the cluster on a given topic.
func (ps *PubSub) Broadcast(topic string, message []byte, options ...*Option) error {
	config := ps.GetAdapter(topic)
	if config == nil {
		return ErrNoAdapter
	}

	if config.Adapter.Name() == "dummy" {
		ps.dispatchMessage(topic, message, ps.getSelfIDString())
		return nil
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
	msgToSend = append(append([]byte{byte(MessageTypeBroadcast)}, ps.getSelfIDBytes()...), msgToSend...)

	// Check if we have compression enabled
	var err error
	skipCompression := config.DisableCompression || len(msgToSend) <= minCompressionSize
	if !skipCompression {
		var compressed []byte
		if compressed, err = compressPayload(msgToSend); err != nil {
			slog.Warn(
				"[chain.pubsub] compression failed",
				slog.String("topic", topic),
				slog.Int("original_size", len(msgToSend)),
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
			slog.Error(
				"[chain.pubsub] encryption failed",
				slog.String("topic", topic),
				slog.Int("original_size", len(msgToSend)),
				slog.Any("error", err),
			)
			return errJoin(errNew("encryption of message failed"), err)
		}
		msgToSend = encrypted
	}

	// Always dispatch locally first (local-first design)
	ps.dispatchMessage(topic, message, ps.getSelfIDString())

	slog.Debug(
		"[chain.pubsub] broadcast",
		slog.String("topic", topic),
		slog.Int("message_size", len(msgToSend)),
	)

	if broadcastErr := config.Adapter.Broadcast(topic, msgToSend, opts); broadcastErr != nil {
		slog.Error(
			"[chain.pubsub] adapter broadcast failed",
			slog.String("topic", topic),
			slog.Int("message_size", len(msgToSend)),
			slog.Any("error", broadcastErr),
		)
		ps.metrics.Error("broadcast", topic, broadcastErr)
		return broadcastErr
	}

	ps.metrics.MessageSent(topic, len(msgToSend), 0)
	return nil
}

// DirectBroadcast sends a message directly to a specific node in the cluster.
func (ps *PubSub) DirectBroadcast(nodeId string, topic string, message []byte, options ...*Option) error {
	// [messageType: byte] [from: 20 bytes] [message: ...]

	nodeIdK, err := ksuid.Parse(nodeId)
	if err != nil {
		return err
	}

	// [to: 20 bytes] [topicNameLen: uint] [topic: topicNameLen] [message: ...]
	buf := &bytesBuffer{}
	buf.Write(nodeIdK.Bytes())

	topicNameLen := make([]byte, 4)
	binaryBigEndianPutUint32(topicNameLen, uint32(len(topic)))
	buf.Write(topicNameLen)

	buf.WriteString(topic)
	buf.Write(message)

	return ps.broadcastMessage(MessageTypeDirectBroadcast, "direct:"+nodeId, buf.Bytes(), options...)
}

// broadcastMessage is the internal broadcast implementation.
func (ps *PubSub) broadcastMessage(msgType MessageType, topic string, message []byte, options ...*Option) error {
	config := ps.GetAdapter(topic)
	if config == nil {
		return ErrNoAdapter
	}

	if config.Adapter.Name() == "dummy" {
		ps.dispatchMessage(topic, message, ps.getSelfIDString())
		return nil
	}

	opts := map[string]any{}
	for k, v := range globalOptions {
		opts[k] = v
	}
	for _, opt := range options {
		opts[opt.key] = opt.value
	}

	// [messageType: byte] [from: 20 bytes] [message: ...]
	buf := &bytesBuffer{}
	buf.WriteByte(byte(msgType))
	buf.Write(ps.getSelfIDBytes())
	buf.Write(message)
	msgToSend := buf.Bytes()

	// Check if we have compression enabled
	var err error
	skipCompression := config.DisableCompression || len(msgToSend) <= minCompressionSize
	if !skipCompression {
		var compressed []byte
		if compressed, err = compressPayload(msgToSend); err != nil {
			slog.Warn(
				"[chain.pubsub] compression failed",
				slog.String("topic", topic),
				slog.Int("original_size", len(msgToSend)),
				slog.Any("error", err),
			)
		} else if len(compressed) < len(msgToSend) {
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
			return errJoin(errNew("encryption of message failed"), err)
		}
		msgToSend = encrypted
	}

	err = config.Adapter.Broadcast(topic, msgToSend, opts)
	return err
}

// Dispatch processes and delivers messages received from external adapters.
func (ps *PubSub) Dispatch(topic string, message []byte) {
	if len(message) == 0 {
		slog.Warn(
			"[chain.pubsub] received empty message",
			slog.String("topic", topic),
		)
		return
	}

	if len(message) > MaxMessageSize {
		slog.Warn(
			"[chain.pubsub] message exceeds maximum size",
			slog.String("topic", topic),
			slog.Int("size", len(message)),
			slog.Int("max", MaxMessageSize),
		)
		return
	}

	if config := ps.GetAdapter(topic); config != nil {
		// Read the message type
		msgType := MessageType(message[0])

		// Check if the message is encrypted
		if msgType == MessageTypeEncrypt {
			if config.DisableEncryption {
				slog.Error(
					"[chain.pubsub] remote message is encrypted and encryption is not configured",
					slog.String("topic", topic),
					slog.String("adapter", config.Adapter.Name()),
				)
				ps.metrics.Error("dispatch", topic, errNew("encryption not configured"))
				return
			}

			keyring := config.Keyring
			if keyring == nil {
				keyring = globalKeyring
			}
			plain, decErr := decryptPayload(keyring, message)
			if decErr != nil {
				slog.Error(
					"[chain.pubsub] could not decrypt remote message",
					slog.String("topic", topic),
					slog.String("adapter", config.Adapter.Name()),
					slog.Any("error", decErr),
				)
				ps.metrics.Error("dispatch", topic, decErr)
				return
			}

			// Reset message type and buf
			msgType = MessageType(plain[0])
			message = plain
		} else if !config.DisableEncryption {
			slog.Error(
				"[chain.pubsub] encryption is configured but remote message is not encrypted",
				slog.String("topic", topic),
				slog.String("adapter", config.Adapter.Name()),
			)
			ps.metrics.Error("dispatch", topic, errNew("message not encrypted"))
			return
		}

		// Check if we have a compressed message
		if msgType == MessageTypeCompress {
			decompressed, decompErr := decompressPayload(message)
			if decompErr != nil {
				slog.Error(
					"[chain.pubsub] could not decompress remote message",
					slog.String("topic", topic),
					slog.String("adapter", config.Adapter.Name()),
					slog.Any("error", decompErr),
				)
				ps.metrics.Error("dispatch", topic, decompErr)
				return
			}

			// Reset message type and buf
			msgType = MessageType(decompressed[0])
			message = decompressed
		}

		// [messageType: byte] [from: 20 bytes] [message: ...]
		message = message[1:]

		if len(message) < 20 {
			slog.Error(
				"[chain.pubsub] invalid remote message length",
				slog.String("topic", topic),
				slog.String("adapter", config.Adapter.Name()),
				slog.Int("message_len", len(message)),
			)
			return
		}
		fromBytes := message[:20]

		fromID, fromErr := ksuid.FromBytes(fromBytes)
		if fromErr != nil {
			slog.Error(
				"[chain.pubsub] invalid remote message from",
				slog.String("topic", topic),
				slog.String("adapter", config.Adapter.Name()),
				slog.Any("error", fromErr),
			)
			return
		}
		from := fromID.String()

		// [message: ...]
		message = message[20:]

		// Check if is a direct broadcast
		if msgType == MessageTypeDirectBroadcast {
			if topic != ps.getDirectTopic() {
				slog.Error(
					"[chain.pubsub] invalid topic for remote direct broadcast message",
					slog.String("topic", topic),
					slog.String("adapter", config.Adapter.Name()),
					slog.String("expected", ps.getDirectTopic()),
				)
				return
			}

			if len(message) < 25 {
				slog.Error(
					"[chain.pubsub] invalid remote direct broadcast length",
					slog.String("topic", topic),
					slog.String("adapter", config.Adapter.Name()),
					slog.Int("message_len", len(message)),
				)
				return
			}

			toBytes := message[0:20]
			message = message[20:]

			if !bytesEqual(ps.getSelfIDBytes(), toBytes) {
				slog.Error(
					"[chain.pubsub] invalid remote direct broadcast destination",
					slog.String("adapter", config.Adapter.Name()),
				)
				return
			}

			// [topicNameLen: uint] [topic: topicNameLen] [message: ...]
			topicNameLen := int(binaryBigEndianUint32(message[0:4]))
			message = message[4:]

			if len(message) < topicNameLen {
				slog.Error(
					"[chain.pubsub] invalid remote direct broadcast length",
					slog.String("adapter", config.Adapter.Name()),
					slog.Int("message_len", len(message)),
					slog.Int("expected_topic_len", topicNameLen),
				)
				return
			}
			topic = string(message[:topicNameLen])
			message = message[topicNameLen:]
		} else if msgType != MessageTypeBroadcast {
			slog.Error(
				"[chain.pubsub] invalid remote message type",
				slog.String("topic", topic),
				slog.String("adapter", config.Adapter.Name()),
				slog.Uint64("message_type", uint64(msgType)),
			)
			return
		}

		ps.dispatchMessage(topic, message, from)
		ps.metrics.MessageReceived(topic, len(message), from)
	}
}

// LocalBroadcast broadcasts a message only to subscribers on the current node.
func (ps *PubSub) LocalBroadcast(topic string, message []byte) {
	ps.dispatchMessage(topic, message, ps.getSelfIDString())
}

// SetAdapters configures adapter instances for specific topic patterns.
func (ps *PubSub) SetAdapters(adapters []AdapterConfig) {
	if config := ps.GetAdapter(ps.getDirectTopic()); config != nil {
		config.Adapter.Unsubscribe(ps.getDirectTopic())
	}
	defer ps.trySubscribe(ps.getDirectTopic())

	ps.adapters = &pkg.WildcardStore[*AdapterConfig]{}
	for _, config := range adapters {
		for _, topic := range config.Topics {
			if insertErr := ps.adapters.Insert(topic, &config); insertErr != nil {
				panic(errFmt("[chain.pubsub] invalid adapter config. Topic: %s, Error: %s", topic, insertErr.Error()))
			}
		}
	}
}

// GetAdapter retrieves the adapter configuration for a given topic.
func (ps *PubSub) GetAdapter(topic string) *AdapterConfig {
	if ps.adapters == nil {
		return nil
	}
	return ps.adapters.Match(topic)
}

// Self returns the current node's unique identifier.
func (ps *PubSub) Self() string {
	return ps.getSelfIDString()
}

// dispatchMessage enqueues a message for async dispatch via worker pool.
func (ps *PubSub) dispatchMessage(topic string, message []byte, from string) {
	ps.closeMutex.Lock()
	if ps.closed {
		ps.closeMutex.Unlock()
		// Instance is closed, dispatch synchronously instead
		ps.dispatchMessageSync(topic, message, from)
		return
	}
	ps.closeMutex.Unlock()

	select {
	case ps.dispatchQueue <- dispatchJob{topic: topic, message: message, from: from}:
		// Successfully enqueued
	default:
		// Queue full - fall back to synchronous dispatch
		ps.dispatchMessageSync(topic, message, from)
	}
}

// trySubscribe subscribes the adapter on the given topic.
func (ps *PubSub) trySubscribe(topic string) {
	if config := ps.GetAdapter(topic); config != nil {
		config.Adapter.Subscribe(topic)
	}
}

// scheduleUnsubscribe unsubscribes the adapter after 5 seconds.
func (ps *PubSub) scheduleUnsubscribe(topic string) {
	ps.unsubscribeMutex.Lock()
	if _, exist := ps.unsubscribeTimers[topic]; exist {
		ps.unsubscribeMutex.Unlock()
		return
	}

	timer := time.NewTimer(time.Second * 5)
	ps.unsubscribeTimers[topic] = timer
	ps.unsubscribeMutex.Unlock()

	// wait
	<-timer.C

	ps.unsubscribeMutex.Lock()
	defer ps.unsubscribeMutex.Unlock()

	if _, exist := ps.unsubscribeTimers[topic]; !exist {
		// was removed by pubsub.trySubscribe
		return
	}
	delete(ps.unsubscribeTimers, topic)

	if config := ps.GetAdapter(topic); config != nil {
		config.Adapter.Unsubscribe(topic)
	}
}

// SetMetricsCollector configures the metrics collector for this instance.
func (ps *PubSub) SetMetricsCollector(m MetricsCollector) {
	if m == nil {
		ps.metrics = &noopMetrics{}
	} else {
		ps.metrics = m
	}
}

// ============================================================================
// Compatibility layer: bridge between instance methods and global functions
// ============================================================================

// Default is the default PubSub instance used by global functions.
var Default = New(
	WithDispatchWorkers(runtime.NumCPU()),
	WithDispatchQueueSize(1000),
)

// Reset resets the default pubsub state for testing purposes.
func Reset() {
	Default.Reset()
}

// Reset resets the pubsub state for testing purposes.
func (ps *PubSub) Reset() {
	ps.subscriptionsMutex.Lock()
	ps.subscriptions = &pkg.WildcardStore[*subscription]{}
	ps.subscriptionsMutex.Unlock()

	ps.unsubscribeMutex.Lock()
	for _, timer := range ps.unsubscribeTimers {
		timer.Stop()
	}
	ps.unsubscribeTimers = map[string]*time.Timer{}
	ps.unsubscribeMutex.Unlock()
}

// Subscribe (global function) delegates to Default instance.
func Subscribe(topicPattern string, dispatcher Dispatcher) {
	Default.Subscribe(topicPattern, dispatcher)
}

// Unsubscribe (global function) delegates to Default instance.
func Unsubscribe(topicPattern string, dispatcher Dispatcher) {
	Default.Unsubscribe(topicPattern, dispatcher)
}

// Broadcast (global function) delegates to Default instance.
func Broadcast(topic string, message []byte, options ...*Option) error {
	return Default.Broadcast(topic, message, options...)
}

// DirectBroadcast (global function) delegates to Default instance.
func DirectBroadcast(nodeId string, topic string, message []byte, options ...*Option) error {
	return Default.DirectBroadcast(nodeId, topic, message, options...)
}

// LocalBroadcast (global function) delegates to Default instance.
func LocalBroadcast(topic string, message []byte) {
	Default.LocalBroadcast(topic, message)
}

// Dispatch (global function) delegates to Default instance.
func Dispatch(topic string, message []byte) {
	Default.Dispatch(topic, message)
}

// SetAdapters (global function) delegates to Default instance.
func SetAdapters(adapters []AdapterConfig) {
	Default.SetAdapters(adapters)
}

// GetAdapter (global function) delegates to Default instance.
func GetAdapter(topic string) *AdapterConfig {
	return Default.GetAdapter(topic)
}

// Self (global function) delegates to Default instance.
func Self() string {
	return Default.Self()
}

// ResetPubsub is an alias to Reset for backward compatibility.
func ResetPubsub() {
	Default.Reset()
}
