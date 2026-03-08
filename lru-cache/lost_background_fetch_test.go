package lrucache

import (
	"context"
	"errors"
	"testing"
)

func TestLostBackgroundFetch(t *testing.T) {
	release := map[int]chan struct{}{
		1: make(chan struct{}),
		2: make(chan struct{}),
	}
	started := make(chan int, 2)

	c := New[int, int](Options[int, int]{
		TTL:                    1000,
		Max:                    10,
		IgnoreFetchAbort:       true,
		AllowStaleOnFetchAbort: true,
		FetchMethod: func(key int, stale *int, opts FetcherOptions[int, int]) (int, bool, error) {
			started <- key
			select {
			case <-release[key]:
				return key, true, nil
			case <-opts.Signal.Done():
				<-release[key]
				return key, true, nil
			}
		},
	})

	ctx, cancel := context.WithCancelCause(context.Background())
	p2 := startAsyncFetch(c, 2, FetchOptions[int, int]{Signal: ctx})
	p1 := startAsyncFetch(c, 1, FetchOptions[int, int]{Signal: ctx})
	<-started
	<-started

	cancel(errors.New("gimme the stale value"))

	r1 := awaitFetchResult(t, p1)
	r2 := awaitFetchResult(t, p2)
	assertFalse(t, r1.ok, "aborted fetch should return undefined for key 1")
	assertEqual(t, r1.err, error(nil), "aborted fetch error for key 1")
	assertFalse(t, r2.ok, "aborted fetch should return undefined for key 2")
	assertEqual(t, r2.err, error(nil), "aborted fetch error for key 2")

	_, ok := c.Get(1, GetOptions[int]{AllowStale: Bool(true)})
	assertFalse(t, ok, "key 1 should not have stale data after abort")
	_, ok = c.Get(2, GetOptions[int]{AllowStale: Bool(true)})
	assertFalse(t, ok, "key 2 should not have stale data after abort")

	close(release[1])
	waitUntil(t, func() bool {
		v, ok := c.Get(1)
		return ok && v == 1
	})
	_, ok = c.Get(2)
	assertFalse(t, ok, "key 2 should still be pending")

	close(release[2])
	waitUntil(t, func() bool {
		v, ok := c.Get(2)
		return ok && v == 2
	})
}
