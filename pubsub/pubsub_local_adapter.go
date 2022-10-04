package pubsub

// LocalAdapter default adapter for local message distribution (only for the current node)
type LocalAdapter struct {
}

func (a *LocalAdapter) Name() string {
	return "dummy"
}

func (a *LocalAdapter) Broadcast(topic string, message any) error {
	// do nothing
	return nil
}

func (a *LocalAdapter) Subscribe(topic string) {
	// do nothing
}

func (a *LocalAdapter) Unsubscribe(topic string) {
	// do nothing
}
