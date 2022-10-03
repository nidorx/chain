package pubsub

// AdapterLocal default adapter for local message distribution (only for the current node)
type AdapterLocal struct {
}

func (a *AdapterLocal) Name() string {
	return "dummy"
}

func (a *AdapterLocal) NodeName() string {
	return "local"
}

func (a *AdapterLocal) Broadcast(topic string, message any) error {
	// do nothing
	return nil
}

func (a *AdapterLocal) Subscribe(topic string) {
	// do nothing
}

func (a *AdapterLocal) Unsubscribe(topic string) {
	// do nothing
}
