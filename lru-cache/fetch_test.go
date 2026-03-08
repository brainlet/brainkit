package lrucache

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fetchCallResult[V any] struct {
	value V
	ok    bool
	err   error
}

type fetchReply[V any] struct {
	value V
	ok    bool
	err   error
}

func startAsyncFetch[K comparable, V any](c *LRUCache[K, V], key K, opts ...FetchOptions[K, V]) <-chan fetchCallResult[V] {
	ch := make(chan fetchCallResult[V], 1)
	go func() {
		v, ok, err := c.Fetch(key, opts...)
		ch <- fetchCallResult[V]{value: v, ok: ok, err: err}
	}()
	return ch
}

func awaitFetchResult[V any](t *testing.T, ch <-chan fetchCallResult[V]) fetchCallResult[V] {
	t.Helper()
	select {
	case res := <-ch:
		return res
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for fetch result")
		var zero fetchCallResult[V]
		return zero
	}
}

func waitUntil(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not satisfied before timeout")
}

func TestFetchAsyncStaleWhileRevalidate(t *testing.T) {
	clock := newTestClock(1)
	stales := make(chan *int, 4)
	replies := make(chan fetchReply[int], 4)
	calls := 0

	c := New[string, int](Options[string, int]{
		Max:   5,
		TTL:   5,
		NowFn: clock.nowFn,
		FetchMethod: func(key string, stale *int, _ FetcherOptions[string, int]) (int, bool, error) {
			calls++
			stales <- stale
			reply := <-replies
			return reply.value, reply.ok, reply.err
		},
	})

	first := startAsyncFetch(c, "key")
	if stale := <-stales; stale != nil {
		t.Fatalf("first fetch should have no stale value, got %v", *stale)
	}
	if _, ok := c.Get("key"); ok {
		t.Fatal("get should miss while first fetch is in flight")
	}
	replies <- fetchReply[int]{value: 0, ok: true}
	firstRes := awaitFetchResult(t, first)
	assertEqual(t, firstRes.err, error(nil), "first fetch error")
	assertTrue(t, firstRes.ok, "first fetch ok")
	assertEqual(t, firstRes.value, 0, "first fetch value")

	v, ok, err := c.Fetch("key")
	assertEqual(t, err, error(nil), "cached fetch error")
	assertTrue(t, ok, "cached fetch ok")
	assertEqual(t, v, 0, "cached fetch value")

	clock.advance(10)
	allowStale := Bool(true)
	v, ok, err = c.Fetch("key", FetchOptions[string, int]{AllowStale: allowStale})
	assertEqual(t, err, error(nil), "stale fetch error")
	assertTrue(t, ok, "stale fetch ok")
	assertEqual(t, v, 0, "allowStale should return stale value immediately")
	stale := <-stales
	if stale == nil || *stale != 0 {
		t.Fatalf("expected stale value 0, got %#v", stale)
	}

	index := exposeKeyMap(c)["key"]
	assertTrue(t, exposeIsBackgroundFetch(c, index), "key should be a background fetch placeholder")

	waiter := startAsyncFetch(c, "key")
	replies <- fetchReply[int]{value: 1, ok: true}
	waiterRes := awaitFetchResult(t, waiter)
	assertEqual(t, waiterRes.err, error(nil), "waiter error")
	assertTrue(t, waiterRes.ok, "waiter ok")
	assertEqual(t, waiterRes.value, 1, "waiter value")

	v, ok, err = c.Fetch("key")
	assertEqual(t, err, error(nil), "refreshed fetch error")
	assertTrue(t, ok, "refreshed fetch ok")
	assertEqual(t, v, 1, "refreshed fetch value")
	assertEqual(t, calls, 2, "fetchMethod should be called twice")
}

func TestFetchWithoutFetchMethod(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 3})
	c.Set(0, 0)
	c.Set(1, 1)

	status := &Status[int]{}
	v, ok, err := c.Fetch(0, FetchOptions[int, int]{Status: status})
	assertEqual(t, err, error(nil), "fetch without fetchMethod error")
	assertTrue(t, ok, "fetch without fetchMethod ok")
	assertEqual(t, v, 0, "fetch without fetchMethod value")
	assertEqual(t, status.Fetch, "get", "status should report get")
}

func TestFetchInflightUnique(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	calls := 0

	c := New[int, int](Options[int, int]{
		Max: 5,
		FetchMethod: func(key int, stale *int, _ FetcherOptions[int, int]) (int, bool, error) {
			calls++
			started <- struct{}{}
			<-release
			return key * 10, true, nil
		},
	})

	results := []<-chan fetchCallResult[int]{
		startAsyncFetch(c, 1),
		startAsyncFetch(c, 1),
		startAsyncFetch(c, 1),
		startAsyncFetch(c, 1),
	}

	<-started
	waitUntil(t, func() bool { return calls == 1 })
	assertFalse(t, c.Has(1), "has should be false while fetching without stale value")
	if _, ok := c.Get(1); ok {
		t.Fatal("get should miss while fetching without stale value")
	}

	close(release)
	for i, ch := range results {
		res := awaitFetchResult(t, ch)
		assertEqual(t, res.err, error(nil), "inflight unique error")
		assertTrue(t, res.ok, "inflight unique ok")
		assertEqual(t, res.value, 10, "inflight unique value")
		if i == 0 {
			continue
		}
	}
	assertEqual(t, calls, 1, "fetchMethod should only run once")
}

func TestFetchAbortOnDeleteReplaceEvict(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		started := make(chan struct{}, 1)
		c := New[int, int](Options[int, int]{
			Max: 5,
			FetchMethod: func(key int, stale *int, opts FetcherOptions[int, int]) (int, bool, error) {
				started <- struct{}{}
				<-opts.Signal.Done()
				return 0, false, nil
			},
		})
		res := startAsyncFetch(c, 1)
		<-started
		assertTrue(t, c.Delete(1), "delete should remove inflight fetch")
		out := awaitFetchResult(t, res)
		if out.err == nil || !strings.Contains(out.err.Error(), "deleted") {
			t.Fatalf("expected deleted error, got %v", out.err)
		}
	})

	t.Run("replace", func(t *testing.T) {
		started := make(chan struct{}, 1)
		c := New[int, int](Options[int, int]{
			Max: 5,
			FetchMethod: func(key int, stale *int, opts FetcherOptions[int, int]) (int, bool, error) {
				started <- struct{}{}
				<-opts.Signal.Done()
				return 0, false, nil
			},
		})
		res := startAsyncFetch(c, 1)
		<-started
		c.Set(1, 99)
		out := awaitFetchResult(t, res)
		if out.err == nil || !strings.Contains(out.err.Error(), "replaced") {
			t.Fatalf("expected replaced error, got %v", out.err)
		}
		v, ok := c.Get(1)
		assertTrue(t, ok, "replacement value should be cached")
		assertEqual(t, v, 99, "replacement value")
	})

	t.Run("evict", func(t *testing.T) {
		started := make(chan struct{}, 1)
		c := New[int, int](Options[int, int]{
			Max: 1,
			FetchMethod: func(key int, stale *int, opts FetcherOptions[int, int]) (int, bool, error) {
				started <- struct{}{}
				<-opts.Signal.Done()
				return 0, false, nil
			},
		})
		res := startAsyncFetch(c, 1)
		<-started
		c.Set(2, 2)
		out := awaitFetchResult(t, res)
		if out.err == nil || !strings.Contains(out.err.Error(), "evicted") {
			t.Fatalf("expected evicted error, got %v", out.err)
		}
		_, ok := c.Get(1)
		assertFalse(t, ok, "evicted key should not remain")
		v, ok := c.Get(2)
		assertTrue(t, ok, "replacement key should remain")
		assertEqual(t, v, 2, "replacement key value")
	})
}

func TestFetchAllowStaleOnFetchRejection(t *testing.T) {
	clock := newTestClock(1)
	fail := true
	c := New[int, int](Options[int, int]{
		Max:                        10,
		TTL:                        10,
		NowFn:                      clock.nowFn,
		AllowStaleOnFetchRejection: true,
		FetchMethod: func(key int, stale *int, _ FetcherOptions[int, int]) (int, bool, error) {
			if fail {
				return 0, false, errors.New("fetch rejection")
			}
			return key, true, nil
		},
	})

	c.Set(1, 1)
	clock.advance(20)

	status := &Status[int]{}
	v, ok, err := c.Fetch(1, FetchOptions[int, int]{Status: status})
	assertEqual(t, err, error(nil), "allow stale on rejection error")
	assertTrue(t, ok, "allow stale on rejection ok")
	assertEqual(t, v, 1, "allow stale on rejection value")
	assertTrue(t, status.ReturnedStale, "status should report stale return")

	allowStale := Bool(false)
	_, ok, err = c.Fetch(1, FetchOptions[int, int]{
		AllowStaleOnFetchRejection: allowStale,
	})
	assertFalse(t, ok, "override should not return a value")
	if err == nil || !strings.Contains(err.Error(), "fetch rejection") {
		t.Fatalf("expected fetch rejection error, got %v", err)
	}
	_, ok = c.Get(1)
	assertFalse(t, ok, "failed fetch without allowStaleOnFetchRejection should delete the entry")
}

func TestFetchForceRefreshAndForceFetch(t *testing.T) {
	replies := make(chan fetchReply[int], 4)
	started := make(chan struct{}, 4)

	c := New[int, int](Options[int, int]{
		Max: 10,
		FetchMethod: func(key int, stale *int, _ FetcherOptions[int, int]) (int, bool, error) {
			started <- struct{}{}
			reply := <-replies
			return reply.value, reply.ok, reply.err
		},
	})

	c.Set(1, 100)
	c.Set(2, 200)

	v, ok, err := c.Fetch(1)
	assertEqual(t, err, error(nil), "hit fetch error")
	assertTrue(t, ok, "hit fetch ok")
	assertEqual(t, v, 100, "hit fetch value")

	allowStale := Bool(true)
	v, ok, err = c.Fetch(1, FetchOptions[int, int]{
		ForceRefresh: true,
		AllowStale:   allowStale,
	})
	assertEqual(t, err, error(nil), "force refresh stale error")
	assertTrue(t, ok, "force refresh stale ok")
	assertEqual(t, v, 100, "force refresh with allowStale should return stale value")
	<-started
	replies <- fetchReply[int]{value: 1, ok: true}
	waitUntil(t, func() bool {
		got, ok := c.Get(1)
		return ok && got == 1
	})

	waiter := startAsyncFetch(c, 2, FetchOptions[int, int]{ForceRefresh: true})
	<-started
	replies <- fetchReply[int]{value: 2, ok: true}
	waitRes := awaitFetchResult(t, waiter)
	assertEqual(t, waitRes.err, error(nil), "force refresh wait error")
	assertTrue(t, waitRes.ok, "force refresh wait ok")
	assertEqual(t, waitRes.value, 2, "force refresh wait value")

	undefinedCache := New[int, int](Options[int, int]{
		Max: 1,
		FetchMethod: func(key int, stale *int, _ FetcherOptions[int, int]) (int, bool, error) {
			return 0, false, nil
		},
	})
	if _, err := undefinedCache.ForceFetch(1); err == nil || !strings.Contains(err.Error(), "fetch() returned undefined") {
		t.Fatalf("expected undefined forceFetch error, got %v", err)
	}
}

func TestFetchSendSignalAndIgnoreAbort(t *testing.T) {
	started := make(chan struct{}, 2)
	sawAbort := make(chan struct{}, 1)
	release := make(chan struct{})
	cache := New[int, int](Options[int, int]{
		Max:              10,
		IgnoreFetchAbort: true,
		FetchMethod: func(key int, stale *int, opts FetcherOptions[int, int]) (int, bool, error) {
			started <- struct{}{}
			<-opts.Signal.Done()
			sawAbort <- struct{}{}
			<-release
			return key, true, nil
		},
	})

	ctx, cancel := context.WithCancelCause(context.Background())
	status := &Status[int]{}
	result := startAsyncFetch(cache, 1, FetchOptions[int, int]{
		Signal: ctx,
		Status: status,
	})
	<-started
	cancel(errors.New("ignored abort signal"))
	<-sawAbort
	close(release)

	out := awaitFetchResult(t, result)
	assertEqual(t, out.err, error(nil), "ignored abort error")
	assertTrue(t, out.ok, "ignored abort ok")
	assertEqual(t, out.value, 1, "ignored abort value")
	assertTrue(t, status.FetchAbortIgnored, "status should report ignored abort")
	waitUntil(t, func() bool {
		v, ok := cache.Get(1)
		return ok && v == 1
	})
}
