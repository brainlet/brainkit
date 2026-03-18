package bus

import "context"

// AskSync wraps Ask into a blocking call with context cancellation.
// This is NOT a bus primitive — it's a caller convenience for Go code
// that needs synchronous request/response (bridges, Go API methods).
func AskSync(b *Bus, ctx context.Context, msg Message) (*Message, error) {
	ch := make(chan Message, 1)
	cancel := b.Ask(msg, func(reply Message) {
		ch <- reply
	})
	select {
	case reply := <-ch:
		return &reply, nil
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}
