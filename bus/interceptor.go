package bus

import "context"

// Interceptor processes messages in the bus pipeline.
// Interceptors are sorted by priority and executed in order.
// They can modify Payload and Metadata, but NOT Topic or CallerID.
// Return an error to reject the message.
type Interceptor interface {
	Name() string
	Priority() int
	Match(topic string) bool
	Process(ctx context.Context, msg *Message) error
}

// interceptorEntry pairs an interceptor with its priority for sorting.
type interceptorEntry struct {
	interceptor Interceptor
	priority    int
}

// runInterceptors executes all matching interceptors in priority order.
func runInterceptors(ctx context.Context, interceptors []interceptorEntry, msg *Message) error {
	for _, entry := range interceptors {
		if entry.interceptor.Match(msg.Topic) {
			// Snapshot immutable fields
			origTopic := msg.Topic
			origCaller := msg.CallerID

			if err := entry.interceptor.Process(ctx, msg); err != nil {
				return err
			}

			// Enforce immutability — revert any changes to frozen fields
			msg.Topic = origTopic
			msg.CallerID = origCaller
		}
	}
	return nil
}
