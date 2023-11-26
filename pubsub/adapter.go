package pubsub

import "github.com/nidorx/chain/crypto"

// Adapter Specification to implement a custom PubSub adapter.
type Adapter interface {
	// Name the Adapter name
	Name() string

	// Subscribe the Adapter that has an external broker must subscribe to the given topic
	Subscribe(topic string)

	// Unsubscribe the Adapter that has an external broker must unsubscribe to the given topic
	Unsubscribe(topic string)

	// Broadcast the given topic and message to all nodes in the cluster (except the current node itself).
	Broadcast(topic string, message []byte, opts map[string]any) error
}

type AdapterConfig struct {

	// Adapter The adapter instance being configured
	Adapter Adapter

	// Keyring allow to define a custom Keyring use for message encryption
	Keyring *crypto.Keyring

	// Options options that will be passed to the adapter during the broadcast
	Options []Option

	// Topics The topic name pattern this adapter must match
	Topics []string

	// RawMessage when true, do not encode messages when transmitting to adapter
	RawMessage bool

	// EnableEncryption enable/disable message encryption
	DisableEncryption bool

	// DisableCompression is used to control message compression. This can be used to reduce bandwidth usage at
	// the cost of slightly more CPU utilization.
	DisableCompression bool
}

// DummyAdapter default adapter for local message distribution (only for the current node)
type DummyAdapter struct {
}

func (a *DummyAdapter) Name() string {
	return "dummy"
}

func (a *DummyAdapter) Broadcast(topic string, message []byte, opts map[string]any) error {
	// do nothing
	return nil
}

func (a *DummyAdapter) Subscribe(topic string) {
	// do nothing
}

func (a *DummyAdapter) Unsubscribe(topic string) {
	// do nothing
}

func init() {
	SetAdapters([]AdapterConfig{
		{
			Adapter:            &DummyAdapter{},
			Topics:             []string{"*"},
			RawMessage:         false,
			DisableCompression: false,
			DisableEncryption:  false,
		},
	})
}
