package transport

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

func TestMemoryBrokerDoesNotSpawnCancelWaiterPerBackgroundSubscription(t *testing.T) {
	before := runtime.NumGoroutine()
	broker := newMemoryBroker()
	t.Cleanup(func() { _ = broker.Close() })

	const subscriptions = 128
	channels := make([]<-chan *Message, 0, subscriptions)
	for i := range subscriptions {
		ch, err := broker.Subscribe(context.Background(), fmt.Sprintf("topic.%d", i))
		if err != nil {
			t.Fatalf("subscribe %d: %v", i, err)
		}
		channels = append(channels, ch)
	}
	runtime.Gosched()

	if delta := runtime.NumGoroutine() - before; delta > subscriptions/4 {
		t.Fatalf("background subscriptions spawned %d goroutines, want at most %d", delta, subscriptions/4)
	}
	if err := broker.Close(); err != nil {
		t.Fatalf("close broker: %v", err)
	}
	for i, ch := range channels {
		select {
		case _, ok := <-ch:
			if ok {
				t.Fatalf("subscription %d channel still open after broker close", i)
			}
		default:
			t.Fatalf("subscription %d channel was not closed by broker close", i)
		}
	}
}
