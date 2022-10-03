package pubsub

import (
	"fmt"
	"github.com/syntax-framework/chain/lib"
	"sync"
	"time"
)

// Adapter Specification to implement a custom PubSub adapter.
type Adapter interface {
	Name() string                              // Returns the Adapter name
	Broadcast(topic string, message any) error // Broadcasts the given topic and message to all nodes in the cluster (except the current node itself).
	Subscribe(topic string)                    // The Adapter that has an external broker must subscribe to the given topic
	Unsubscribe(topic string)                  // The Adapter that has an external broker must unsubscribe to the given topic
}

type AdapterConfig struct {
	Adapter Adapter
	Topics  []string
}

type Dispatcher interface {
	Dispatch(topic string, message any)
}

// subscription representa as subscrições que este server possui. See pubsub.Subscribe
type subscription struct {
	dispatchers map[Dispatcher]int // incremental dispatcher subscriptions
}

// pubsub Realtime Publisher/Subscriber _ignore.service.
type pubsub struct {
	adapters           *lib.WildcardStore[Adapter]
	subscriptions      map[string]*subscription
	unsubscribeTimers  map[string]*time.Timer
	unsubscribeMutex   sync.Mutex
	subscriptionsMutex sync.RWMutex
}

var p = &pubsub{
	subscriptions:     map[string]*subscription{},
	unsubscribeTimers: map[string]*time.Timer{},
}

func init() {
	SetAdapters([]AdapterConfig{{
		Adapter: &AdapterLocal{},
		Topics:  []string{"*"},
	}})
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

// Broadcast broadcasts ServiceMsg on given topic across the whole cluster.
//
// Para um dispatcher, ver service.Send
func Broadcast(topic string, message any) (err error) {
	if adapter := GetAdapter(topic); adapter != nil {
		if err = adapter.Broadcast(topic, message); err == nil {
			// local dispatch
			dispatch(topic, message)
		}
		return
	}

	// log adapter not found
	return
}

// LocalBroadcast broadcasts ServiceMsg on given topic only for the current node.
//
// `topic` - The topic to broadcast to, ie: `"users:123"`
// `message` - The payload of the broadcast
func LocalBroadcast(topic string, message any) {
	dispatch(topic, message)
}

// DirectBroadcast Broadcasts ServiceMsg on given topic to a given node.
func DirectBroadcast(nodeName string, topic string, message any, dispatcher string) {

}

// SetAdapters configure the adapters topics
func SetAdapters(adapters []AdapterConfig) {
	p.adapters = &lib.WildcardStore[Adapter]{}
	for _, config := range adapters {
		for _, topic := range config.Topics {
			if err := p.adapters.Insert(topic, config.Adapter); err != nil {
				panic(any(fmt.Sprintf("invalid adapter config for topic %s. Cause: %s", topic, err.Error())))
			}
		}
	}
}

func GetAdapter(topic string) Adapter {
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

	if adapter := GetAdapter(topic); adapter != nil {
		adapter.Subscribe(topic)
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

	if adapter := GetAdapter(topic); adapter != nil {
		adapter.Unsubscribe(topic)
	}
}

func dispatch(topic string, message any) {
	p.subscriptionsMutex.RLock()
	var sub *subscription
	var exist bool
	if sub, exist = p.subscriptions[topic]; !exist {
		p.subscriptionsMutex.RUnlock()
		go scheduleUnsubscribe(topic)
		return
	}
	var dispatchers []Dispatcher
	for dispatchFunc, _ := range sub.dispatchers {
		dispatchers = append(dispatchers, dispatchFunc)
	}
	p.subscriptionsMutex.RUnlock()

	for _, dispatcher := range dispatchers {
		dispatcher.Dispatch(topic, message)
	}
}
