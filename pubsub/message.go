package pubsub

// messageType is an integer ID of a type of message that can be received on network channels from other members.
type messageType uint8

// The list of available message types.
const (
	messageTypeCompress messageType = iota
	messageTypeEncrypt
	messageTypeBroadcast
	messageTypeDirectBroadcast
	indirectPingMsg
	ackRespMsg
	suspectMsg
	aliveMsg
	deadMsg
	pushPullMsg
	compoundMsg
	userMsg
	nackRespMsg
	errMsg
)
