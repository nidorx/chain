package pubsub

// GossipAdapter default adapter for local message distribution (only for the current node)
type GossipAdapter struct {
}

func (a *GossipAdapter) Name() string {
	return "gossip"
}

func (a *GossipAdapter) Broadcast(topic string, message any) error {
	// do nothing
	return nil
}

func (a *GossipAdapter) Subscribe(topic string) {
	// do nothing
}

func (a *GossipAdapter) Unsubscribe(topic string) {
	// do nothing
}
