package pubsub

var (
	globalOptions = map[string]any{}
)

type Option struct {
	key   string
	value any
}

func (o *Option) Key() string {
	return o.key
}

func (o *Option) Value() any {
	return o.value
}

func O(key string, value any) *Option {
	return &Option{key, value}
}

// SetGlobalOptions set global options for sending messages
func SetGlobalOptions(options ...*Option) {
	for _, option := range options {
		key := option.key
		value := option.value
		globalOptions[key] = value
	}
}
