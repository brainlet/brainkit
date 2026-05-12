package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
	"github.com/coder/websocket"
)

type gatewayTestRuntime struct {
	mu          sync.Mutex
	unsubbed    int
	failTopic   string
	failPublish bool
}

func (r *gatewayTestRuntime) PublishRaw(context.Context, string, json.RawMessage) (string, error) {
	if r.failPublish {
		return "", fmt.Errorf("publish failed")
	}
	return "corr", nil
}

func (r *gatewayTestRuntime) SubscribeRaw(_ context.Context, topic string, _ func(sdk.Message)) (func(), error) {
	if topic == r.failTopic {
		return nil, fmt.Errorf("subscribe failed")
	}
	return func() {
		r.mu.Lock()
		r.unsubbed++
		r.mu.Unlock()
	}, nil
}

func (r *gatewayTestRuntime) Close() error { return nil }

func (r *gatewayTestRuntime) unsubscribeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.unsubbed
}

type gatewayHandleRuntime struct {
	gatewayTestRuntime
	closeErr      error
	closeAttempts int
}

func (r *gatewayHandleRuntime) SubscribeRawHandle(_ context.Context, topic string, _ func(sdk.Message)) (bkmodule.Handle, error) {
	if topic == r.failTopic {
		return nil, fmt.Errorf("subscribe failed")
	}
	return bkmodule.HandleFunc(func(context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.closeAttempts++
		if r.closeErr != nil {
			return r.closeErr
		}
		r.unsubbed++
		return nil
	}), nil
}

func (r *gatewayHandleRuntime) closeAttemptCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeAttempts
}

type gatewayBlockingStreamHandleRuntime struct {
	gatewayTestRuntime
	closeStarted     chan struct{}
	closeStartedOnce sync.Once
	releaseClose     chan struct{}
	closeDeadline    chan bool
}

func (r *gatewayBlockingStreamHandleRuntime) SubscribeRawHandle(_ context.Context, topic string, _ func(sdk.Message)) (bkmodule.Handle, error) {
	if topic == r.failTopic {
		return nil, fmt.Errorf("subscribe failed")
	}
	return bkmodule.HandleFunc(func(ctx context.Context) error {
		if r.closeStarted != nil {
			r.closeStartedOnce.Do(func() { close(r.closeStarted) })
		}
		if r.closeDeadline != nil {
			_, hasDeadline := ctx.Deadline()
			select {
			case r.closeDeadline <- hasDeadline:
			default:
			}
		}
		if r.releaseClose != nil {
			select {
			case <-r.releaseClose:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		r.mu.Lock()
		r.unsubbed++
		r.mu.Unlock()
		return nil
	}), nil
}

type gatewayAudioHandleRuntime struct {
	gatewayTestRuntime
	audioSubscribed    chan struct{}
	audioCloseDeadline chan bool
}

func (r *gatewayAudioHandleRuntime) SubscribeRawHandle(_ context.Context, topic string, _ func(sdk.Message)) (bkmodule.Handle, error) {
	if strings.HasPrefix(topic, "audio.out.") {
		select {
		case r.audioSubscribed <- struct{}{}:
		default:
		}
		return bkmodule.HandleFunc(func(ctx context.Context) error {
			_, hasDeadline := ctx.Deadline()
			select {
			case r.audioCloseDeadline <- hasDeadline:
			default:
			}
			return nil
		}), nil
	}
	return bkmodule.HandleFunc(func(context.Context) error { return nil }), nil
}

type gatewayTestCaller struct{}

func (gatewayTestCaller) Call(context.Context, string, json.RawMessage, sdk.CallerConfig) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}

func TestCallerAccessIsSafeDuringConcurrentClear(t *testing.T) {
	gw := New(Config{Listen: "127.0.0.1:0"})
	caller := gatewayTestCaller{}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				if (i+j)%2 == 0 {
					gw.setCaller(caller)
				} else {
					gw.setCaller(nil)
				}
				_ = gw.requestCaller()
			}
		}(i)
	}
	wg.Wait()
}

func TestStopUnsubscribesStreamSessions(t *testing.T) {
	rt := &gatewayTestRuntime{}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	session, err := newStreamSession(gw, "reply.topic", "corr")
	if err != nil {
		t.Fatalf("new stream session: %v", err)
	}
	if rt.unsubscribeCount() != 0 {
		t.Fatalf("unsubscribe count before stop = %d", rt.unsubscribeCount())
	}
	if got := gw.findSession(session.id); got == nil {
		t.Fatal("stream session was not registered")
	}

	if err := gw.Stop(); err != nil {
		t.Fatalf("stop gateway: %v", err)
	}
	if rt.unsubscribeCount() != 1 {
		t.Fatalf("unsubscribe count after stop = %d, want 1", rt.unsubscribeCount())
	}
	if got := gw.findSession(session.id); got != nil {
		t.Fatal("stream session still registered after stop")
	}
}

func TestStopContextClearsListeningState(t *testing.T) {
	rt := &gatewayTestRuntime{}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	if !gw.DebugSnapshot().Listening {
		t.Fatal("gateway did not report listening after start")
	}
	if !gw.DebugSnapshot().ServerAttached {
		t.Fatal("gateway did not report attached server after start")
	}
	if !gw.DebugSnapshot().SessionSweepRunning {
		t.Fatal("gateway did not report session sweep after start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := gw.StopContext(ctx); err != nil {
		t.Fatalf("stop gateway: %v", err)
	}
	if gw.DebugSnapshot().Listening {
		t.Fatal("gateway still reports listening after stop")
	}
	if gw.DebugSnapshot().ServerAttached {
		t.Fatal("gateway still reports attached server after stop")
	}
	if gw.DebugSnapshot().SessionSweepRunning {
		t.Fatal("gateway still reports session sweep after stop")
	}
	if gw.srv != nil || gw.ln != nil {
		t.Fatalf("gateway server/listener not cleared after stop: srv=%v ln=%v", gw.srv, gw.ln)
	}
}

func TestStartRefusesDuplicateGatewayResources(t *testing.T) {
	rt := &gatewayTestRuntime{}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	err := gw.Start()
	if err == nil || !strings.Contains(err.Error(), "already started") {
		t.Fatalf("second Start error = %v, want already started", err)
	}
	if got := gw.DebugSnapshot().RouteSubscriptions; got != 4 {
		t.Fatalf("route subscriptions after duplicate start = %d, want 4", got)
	}
	if err := gw.Stop(); err != nil {
		t.Fatalf("stop gateway: %v", err)
	}
}

func TestStopContextRetainsRouteSubscriptionsWhenCloseFails(t *testing.T) {
	want := errors.New("unsubscribe failed")
	rt := &gatewayHandleRuntime{closeErr: want}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := gw.StopContext(ctx)
	if !errors.Is(err, want) {
		t.Fatalf("StopContext error = %v, want %v", err, want)
	}
	if got := gw.DebugSnapshot().RouteSubscriptions; got != 4 {
		t.Fatalf("route subscriptions after failed close = %d, want retained 4", got)
	}
	if got := rt.closeAttemptCount(); got != 4 {
		t.Fatalf("close attempts after failed close = %d, want 4", got)
	}

	rt.closeErr = nil
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := gw.StopContext(retryCtx); err != nil {
		t.Fatalf("retry StopContext: %v", err)
	}
	if got := gw.DebugSnapshot().RouteSubscriptions; got != 0 {
		t.Fatalf("route subscriptions after retry = %d, want 0", got)
	}
	if got := rt.unsubscribeCount(); got != 4 {
		t.Fatalf("unsubscribe count after retry = %d, want 4", got)
	}
}

func TestStopContextRetainsStreamSessionWhenSubscriptionCloseFails(t *testing.T) {
	want := errors.New("stream unsubscribe failed")
	rt := &gatewayHandleRuntime{closeErr: want}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	session, err := newStreamSession(gw, "reply.topic", "corr")
	if err != nil {
		t.Fatalf("new stream session: %v", err)
	}
	if got := gw.findSession(session.id); got == nil {
		t.Fatal("stream session was not registered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = gw.StopContext(ctx)
	if !errors.Is(err, want) {
		t.Fatalf("StopContext error = %v, want %v", err, want)
	}
	snapshot := gw.DebugSnapshot()
	if snapshot.StreamSessions != 1 || snapshot.StreamSubscriptions != 1 {
		t.Fatalf("stream state after failed close = sessions %d subscriptions %d, want retained 1/1",
			snapshot.StreamSessions, snapshot.StreamSubscriptions)
	}

	rt.closeErr = nil
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := gw.StopContext(retryCtx); err != nil {
		t.Fatalf("retry StopContext: %v", err)
	}
	snapshot = gw.DebugSnapshot()
	if snapshot.StreamSessions != 0 || snapshot.StreamSubscriptions != 0 {
		t.Fatalf("stream state after retry = sessions %d subscriptions %d, want 0/0",
			snapshot.StreamSessions, snapshot.StreamSubscriptions)
	}
	if got := rt.unsubscribeCount(); got != 1 {
		t.Fatalf("stream unsubscribe count after retry = %d, want 1", got)
	}
}

func TestCloseStreamSessionDoesNotHoldSessionMapLockWhileSubscriptionCloseBlocks(t *testing.T) {
	rt := &gatewayBlockingStreamHandleRuntime{
		closeStarted: make(chan struct{}),
		releaseClose: make(chan struct{}),
	}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	session, err := newStreamSession(gw, "reply.topic", "corr")
	if err != nil {
		t.Fatalf("new stream session: %v", err)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- gw.closeStreamSession(closeCtx, session.id, "test")
	}()

	select {
	case <-rt.closeStarted:
	case <-time.After(time.Second):
		t.Fatal("stream subscription close did not start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- gw.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if snapshot.StreamSessions != 1 || snapshot.StreamSubscriptions != 1 {
			t.Fatalf("stream state while close is blocked = sessions %d subscriptions %d, want retained 1/1",
				snapshot.StreamSessions, snapshot.StreamSubscriptions)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked behind stream subscription close")
	}

	close(rt.releaseClose)
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("close stream session: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream session close did not finish after release")
	}
	snapshot := gw.DebugSnapshot()
	if snapshot.StreamSessions != 0 || snapshot.StreamSubscriptions != 0 {
		t.Fatalf("stream state after close = sessions %d subscriptions %d, want 0/0",
			snapshot.StreamSessions, snapshot.StreamSubscriptions)
	}
	if got := rt.unsubscribeCount(); got != 1 {
		t.Fatalf("stream unsubscribe count after close = %d, want 1", got)
	}
}

func TestStopContextDrainsActiveStreamConnection(t *testing.T) {
	rt := &gatewayTestRuntime{}
	gw := New(Config{
		Listen: "127.0.0.1:0",
		Stream: &StreamConfig{
			HeartbeatTimeout: time.Hour,
			MaxDuration:      time.Hour,
			GracePeriod:      time.Hour,
		},
	})
	gw.SetRuntime(rt)
	gw.HandleStream(http.MethodPost, "/stream", "stream.topic")

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}

	resp, err := http.Post("http://"+gw.Addr()+"/stream", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	waitGatewayCondition(t, func() bool {
		return gw.DebugSnapshot().ActiveConnections == 1
	}, "active stream connection")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := gw.StopContext(ctx); err != nil {
		t.Fatalf("stop gateway: %v", err)
	}
	snapshot := gw.DebugSnapshot()
	if snapshot.ActiveConnections != 0 || snapshot.StreamSessions != 0 || snapshot.StreamSubscriptions != 0 {
		t.Fatalf("gateway did not drain stream state: active=%d sessions=%d subscriptions=%d",
			snapshot.ActiveConnections, snapshot.StreamSessions, snapshot.StreamSubscriptions)
	}
	if snapshot.Listening || snapshot.SessionSweepRunning {
		t.Fatalf("gateway lifecycle state after stream drain = listening %t sweep %t, want false/false",
			snapshot.Listening, snapshot.SessionSweepRunning)
	}
}

func TestStopContextDrainsActiveWebSocketConnection(t *testing.T) {
	rt := &gatewayTestRuntime{}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)
	gw.HandleWebSocket("/ws", "ws.topic")

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), time.Second)
	defer dialCancel()
	conn, _, err := websocket.Dial(dialCtx, "ws://"+gw.Addr()+"/ws", nil)
	if err != nil {
		t.Fatalf("open websocket: %v", err)
	}
	defer conn.CloseNow()
	waitGatewayCondition(t, func() bool {
		return gw.DebugSnapshot().ActiveConnections == 1
	}, "active websocket connection")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := gw.StopContext(stopCtx); err != nil {
		t.Fatalf("stop gateway: %v", err)
	}
	snapshot := gw.DebugSnapshot()
	if snapshot.ActiveConnections != 0 {
		t.Fatalf("active connections after websocket stop = %d, want 0", snapshot.ActiveConnections)
	}
	if snapshot.Listening || snapshot.SessionSweepRunning {
		t.Fatalf("gateway lifecycle state after websocket drain = listening %t sweep %t, want false/false",
			snapshot.Listening, snapshot.SessionSweepRunning)
	}
}

func TestWebSocketAudioClosesSubscriptionWithBoundedContext(t *testing.T) {
	rt := &gatewayAudioHandleRuntime{
		audioSubscribed:    make(chan struct{}, 1),
		audioCloseDeadline: make(chan bool, 1),
	}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)
	gw.HandleWebSocketAudio("/audio", "audio.in", "audio.out")

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = gw.StopContext(ctx)
	})

	dialCtx, dialCancel := context.WithTimeout(context.Background(), time.Second)
	defer dialCancel()
	conn, _, err := websocket.Dial(dialCtx, "ws://"+gw.Addr()+"/audio", nil)
	if err != nil {
		t.Fatalf("open websocket audio: %v", err)
	}

	select {
	case <-rt.audioSubscribed:
	case <-time.After(time.Second):
		t.Fatal("websocket audio subscription was not opened")
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")

	select {
	case hasDeadline := <-rt.audioCloseDeadline:
		if !hasDeadline {
			t.Fatal("websocket audio subscription close used an unbounded context")
		}
	case <-time.After(time.Second):
		t.Fatal("websocket audio subscription was not closed")
	}
}

func TestStopContextReturnsDeadlineForStuckActiveRequest(t *testing.T) {
	rt := &gatewayTestRuntime{}
	entered := make(chan struct{})
	release := make(chan struct{})
	var enterOnce sync.Once
	gw := New(Config{
		Listen: "127.0.0.1:0",
		Middleware: []Middleware{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/hang" {
						enterOnce.Do(func() { close(entered) })
						<-release
						w.WriteHeader(http.StatusNoContent)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
	})
	gw.SetRuntime(rt)

	if err := gw.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	reqDone := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://" + gw.Addr() + "/hang")
		if err != nil {
			reqDone <- err
			return
		}
		resp.Body.Close()
		reqDone <- nil
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stuck request to enter handler")
	}
	waitGatewayCondition(t, func() bool {
		return gw.DebugSnapshot().ActiveConnections == 1
	}, "active stuck request")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	start := time.Now()
	stopErr := make(chan error, 1)
	go func() {
		stopErr <- gw.StopContext(stopCtx)
	}()
	waitGatewayCondition(t, func() bool {
		return gw.DebugSnapshot().Closing
	}, "gateway closing state")
	err := <-stopErr
	stopCancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("StopContext error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("StopContext waited %s after shutdown deadline", elapsed)
	}
	if gw.DebugSnapshot().Listening {
		t.Fatal("gateway still reports listening after shutdown deadline")
	}
	if !gw.DebugSnapshot().ServerAttached {
		t.Fatal("gateway dropped server handle from debug snapshot while active request was still running")
	}

	retryDeadlineCtx, retryDeadlineCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	retryDeadlineErr := gw.StopContext(retryDeadlineCtx)
	retryDeadlineCancel()
	if !errors.Is(retryDeadlineErr, context.DeadlineExceeded) {
		t.Fatalf("second StopContext error = %v, want context deadline exceeded", retryDeadlineErr)
	}
	if gw.DebugSnapshot().Listening {
		t.Fatal("gateway still reports listening after second shutdown deadline")
	}
	if !gw.DebugSnapshot().ServerAttached {
		t.Fatal("gateway dropped server handle after second shutdown deadline")
	}
	if got := gw.DebugSnapshot().ActiveConnections; got != 1 {
		t.Fatalf("active connections after second shutdown deadline = %d, want 1", got)
	}

	close(release)
	select {
	case <-reqDone:
	case <-time.After(time.Second):
		t.Fatal("stuck request did not finish after release")
	}
	waitGatewayCondition(t, func() bool {
		return gw.DebugSnapshot().ActiveConnections == 0
	}, "active stuck request cleanup")

	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := gw.StopContext(retryCtx); err != nil {
		t.Fatalf("retry stop after active request cleanup: %v", err)
	}
	if gw.DebugSnapshot().ServerAttached {
		t.Fatal("gateway still reports attached server after retry stop")
	}
	gw.lifecycleMu.Lock()
	defer gw.lifecycleMu.Unlock()
	if gw.srv != nil || gw.ln != nil {
		t.Fatalf("gateway server/listener not cleared after retry stop: srv=%v ln=%v", gw.srv, gw.ln)
	}
}

func TestStopContextResetsWaitersAcrossRestart(t *testing.T) {
	rt := &gatewayTestRuntime{}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	if err := gw.Start(); err != nil {
		t.Fatalf("first start gateway: %v", err)
	}
	if err := gw.StopContext(context.Background()); err != nil {
		t.Fatalf("first stop gateway: %v", err)
	}
	if err := gw.Start(); err != nil {
		t.Fatalf("second start gateway: %v", err)
	}
	if err := gw.StopContext(context.Background()); err != nil {
		t.Fatalf("second stop gateway: %v", err)
	}
	snapshot := gw.DebugSnapshot()
	if snapshot.Listening || snapshot.ServerAttached || snapshot.SessionSweepRunning {
		t.Fatalf("gateway lifecycle after restart stop = listening %t attached %t sweep %t, want all false",
			snapshot.Listening, snapshot.ServerAttached, snapshot.SessionSweepRunning)
	}
}

func waitGatewayCondition(t *testing.T, ok func() bool, label string) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", label)
		case <-ticker.C:
			if ok() {
				return
			}
		}
	}
}

func TestStartCleansUpRouteSubscriptionsWhenSubscribeFails(t *testing.T) {
	rt := &gatewayTestRuntime{failTopic: "gateway.http.route.remove"}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	err := gw.Start()
	if err == nil {
		t.Fatal("Start: expected route subscription error")
	}
	if rt.unsubscribeCount() != 1 {
		t.Fatalf("unsubscribe count after failed start = %d, want 1", rt.unsubscribeCount())
	}
	if got := gw.routeSubscriptionCount(); got != 0 {
		t.Fatalf("bus subscriptions after failed start = %d, want 0", got)
	}
}

func TestHandleStreamCleansUpSessionWhenPublishFails(t *testing.T) {
	rt := &gatewayTestRuntime{failPublish: true}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	rec := httptest.NewRecorder()
	gw.handleStream(rec, req, &route{
		Method: http.MethodPost,
		Path:   "/stream",
		Topic:  "stream.topic",
		Type:   routeStream,
	}, nil)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if rt.unsubscribeCount() != 1 {
		t.Fatalf("unsubscribe count after publish failure = %d, want 1", rt.unsubscribeCount())
	}
	snapshot := gw.DebugSnapshot()
	if snapshot.StreamSessions != 0 || snapshot.StreamSubscriptions != 0 {
		t.Fatalf("stream lifecycle after publish failure = sessions %d subscriptions %d, want 0/0", snapshot.StreamSessions, snapshot.StreamSubscriptions)
	}
}

func TestHandleStreamPublishFailureUsesBoundedSessionCloseContext(t *testing.T) {
	rt := &gatewayBlockingStreamHandleRuntime{
		gatewayTestRuntime: gatewayTestRuntime{failPublish: true},
		closeDeadline:      make(chan bool, 1),
	}
	gw := New(Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(rt)

	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	rec := httptest.NewRecorder()
	gw.handleStream(rec, req, &route{
		Method: http.MethodPost,
		Path:   "/stream",
		Topic:  "stream.topic",
		Type:   routeStream,
	}, nil)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	select {
	case hasDeadline := <-rt.closeDeadline:
		if !hasDeadline {
			t.Fatal("publish-failure stream close context had no deadline")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream close context deadline observation")
	}
}
