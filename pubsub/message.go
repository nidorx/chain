package pubsub

// MessageType is an integer ID of a type of message that can be received on network channels from other members.
type MessageType uint8

// The list of available message types.
const (
	MessageTypeCompress MessageType = iota
	MessageTypeEncrypt
	MessageTypeBroadcast
	MessageTypeDirectBroadcast
	IndirectPingMsg
	AckRespMsg
	SuspectMsg
	AliveMsg
	DeadMsg
	PushPullMsg
	CompoundMsg
	UserMsg
	NackRespMsg
	ErrMsg
)
