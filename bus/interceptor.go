package bus

// Interceptor processes messages before dispatch.
// Sorted by priority (lowest first). Can modify Payload/Metadata only.
// Return error to reject the message.
type Interceptor interface {
	Name() string
	Priority() int
	Match(topic string) bool
	Process(msg *Message) error
}

type interceptorEntry struct {
	interceptor Interceptor
	priority    int
}

// runInterceptors executes matching interceptors in priority order.
// Enforces immutability on Topic, CallerID, and Address.
func runInterceptors(interceptors []interceptorEntry, msg *Message) error {
	for _, entry := range interceptors {
		if entry.interceptor.Match(msg.Topic) {
			origTopic := msg.Topic
			origCaller := msg.CallerID
			origAddr := msg.Address

			if err := entry.interceptor.Process(msg); err != nil {
				return err
			}

			msg.Topic = origTopic
			msg.CallerID = origCaller
			msg.Address = origAddr
		}
	}
	return nil
}
