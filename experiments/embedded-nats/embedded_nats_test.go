package embedded_nats_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	wmnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// =============================================================================
// Experiment 1: Raw embedded NATS server lifecycle
// How fast does it start? How fast is ReadyForConnections?
// =============================================================================

func TestEmbeddedNATS_StartupTime(t *testing.T) {
	// Ephemeral — temp dir, auto-cleanup
	storeDir := t.TempDir()

	start := time.Now()
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1, // random
		NoLog:     true,
		NoSigs:    true,
		JetStream: true,
		StoreDir:  storeDir,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	newServerTime := time.Since(start)

	startTime := time.Now()
	ns.Start()
	serverStartTime := time.Since(startTime)

	readyStart := time.Now()
	if !ns.ReadyForConnections(10 * time.Second) {
		t.Fatal("server not ready in 10s")
	}
	readyTime := time.Since(readyStart)

	totalTime := time.Since(start)

	t.Logf("NewServer:             %s", newServerTime)
	t.Logf("Start():               %s", serverStartTime)
	t.Logf("ReadyForConnections(): %s", readyTime)
	t.Logf("TOTAL startup:         %s", totalTime)
	t.Logf("ClientURL:             %s", ns.ClientURL())
	t.Logf("JetStream enabled:     %v", ns.JetStreamEnabled())
	t.Logf("Server ID:             %s", ns.ID())

	ns.Shutdown()
	ns.WaitForShutdown()
	t.Logf("Shutdown complete")
}

// =============================================================================
// Experiment 2: Connect nats.go client to embedded server
// =============================================================================

func TestEmbeddedNATS_ClientConnect(t *testing.T) {
	ns := startEmbeddedServer(t)

	start := time.Now()
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer nc.Close()
	connectTime := time.Since(start)

	t.Logf("Client connect time: %s", connectTime)
	t.Logf("Connected to: %s", nc.ConnectedUrl())
	t.Logf("RTT: %s", func() string {
		rtt, err := nc.RTT()
		if err != nil {
			return "error: " + err.Error()
		}
		return rtt.String()
	}())
}

// =============================================================================
// Experiment 3: JetStream stream provisioning speed
// This is the critical question: how fast can we create ~30 streams
// (simulating our ~30 command handlers)?
// =============================================================================

func TestEmbeddedNATS_JetStreamProvisioningSpeed(t *testing.T) {
	ns := startEmbeddedServer(t)

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("JetStream: %v", err)
	}

	// Simulate the ~30 command handler streams brainkit creates
	topics := []string{
		"tools-call", "tools-resolve", "tools-list",
		"agents-list", "agents-discover", "agents-get-status", "agents-set-status",
		"kit-deploy", "kit-teardown", "kit-redeploy", "kit-list", "kit-deploy-file",
		"kit-eval-ts", "kit-eval-module", "kit-set-draining", "kit-health",
		"workflow-start", "workflow-startAsync", "workflow-status", "workflow-resume",
		"workflow-cancel", "workflow-list", "workflow-runs",
		"secrets-set", "secrets-get", "secrets-delete", "secrets-list",
		"schedules-create", "schedules-cancel", "schedules-list",
		"registry-has", "registry-list", "registry-resolve",
		"mcp-listTools", "mcp-callTool",
	}

	start := time.Now()
	for _, topic := range topics {
		_, err := js.AddStream(&nats.StreamConfig{
			Name:     topic,
			Subjects: []string{topic},
		})
		if err != nil {
			t.Fatalf("AddStream(%s): %v", topic, err)
		}
	}
	provisionTime := time.Since(start)

	t.Logf("Created %d JetStream streams in %s", len(topics), provisionTime)
	t.Logf("Average per stream: %s", provisionTime/time.Duration(len(topics)))
}

// =============================================================================
// Experiment 4: Watermill NATS adapter on embedded server
// This is what brainkit actually uses. Can we create a Watermill publisher
// and subscriber connected to the embedded NATS?
// =============================================================================

func TestEmbeddedNATS_WatermillAdapter(t *testing.T) {
	ns := startEmbeddedServer(t)
	wmLogger := watermill.NopLogger{}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	start := time.Now()
	pub, err := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               ns.ClientURL(),
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, wmLogger)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()
	pubTime := time.Since(start)

	start = time.Now()
	sub, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               ns.ClientURL(),
		QueueGroupPrefix:  "test",
		SubscribersCount:  1,
		CloseTimeout:      5 * time.Second,
		AckWaitTimeout:    10 * time.Second,
		SubscribeTimeout:  10 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision: true,
			DurablePrefix: "test",
			TrackMsgId:    true,
		},
	}, wmLogger)
	if err != nil {
		t.Fatalf("NewSubscriber: %v", err)
	}
	defer sub.Close()
	subTime := time.Since(start)

	t.Logf("Publisher create:  %s", pubTime)
	t.Logf("Subscriber create: %s", subTime)

	// Pub/sub round trip
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := sub.Subscribe(ctx, "test-topic")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	roundTripStart := time.Now()
	msg := message.NewMessage(watermill.NewUUID(), []byte(`{"hello":"embedded-nats"}`))
	if err := pub.Publish("test-topic", msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case received := <-ch:
		roundTripTime := time.Since(roundTripStart)
		received.Ack()
		t.Logf("Round-trip time: %s", roundTripTime)
		t.Logf("Payload: %s", string(received.Payload))
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}
}

// =============================================================================
// Experiment 5: Full Watermill router startup (simulates brainkit's router)
// This is the closest simulation of what brainkit.New() would do with
// embedded NATS. Creates a router, adds ~30 handlers, starts it, measures
// time to router.Running().
// =============================================================================

func TestEmbeddedNATS_RouterStartupTime(t *testing.T) {
	ns := startEmbeddedServer(t)
	wmLogger := watermill.NopLogger{}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	pub, err := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               ns.ClientURL(),
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, wmLogger)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	sub, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               ns.ClientURL(),
		QueueGroupPrefix:  "brainkit",
		SubscribersCount:  1,
		CloseTimeout:      5 * time.Second,
		AckWaitTimeout:    10 * time.Second,
		SubscribeTimeout:  10 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision: true,
			DurablePrefix: "brainkit",
			TrackMsgId:    true,
		},
	}, wmLogger)
	if err != nil {
		t.Fatalf("NewSubscriber: %v", err)
	}
	defer sub.Close()

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	// Simulate brainkit's ~30 command handlers
	topics := []string{
		"test-tools-call", "test-tools-resolve", "test-tools-list",
		"test-agents-list", "test-agents-discover", "test-agents-get-status",
		"test-kit-deploy", "test-kit-teardown", "test-kit-redeploy", "test-kit-list",
		"test-kit-eval-ts", "test-kit-eval-module", "test-kit-set-draining", "test-kit-health",
		"test-workflow-start", "test-workflow-status", "test-workflow-resume",
		"test-workflow-cancel", "test-workflow-list", "test-workflow-runs",
		"test-secrets-set", "test-secrets-get", "test-secrets-delete", "test-secrets-list",
		"test-schedules-create", "test-schedules-cancel", "test-schedules-list",
		"test-registry-has", "test-registry-list", "test-registry-resolve",
		"test-mcp-listTools", "test-mcp-callTool",
	}

	noopHandler := func(msg *message.Message) ([]*message.Message, error) {
		msg.Ack()
		return nil, nil
	}

	for _, topic := range topics {
		router.AddHandler(
			"handler-"+topic,
			topic,
			sub,
			topic+"-out",
			pub,
			noopHandler,
		)
	}

	start := time.Now()
	go func() {
		_ = router.Run(context.Background())
	}()

	select {
	case <-router.Running():
		routerTime := time.Since(start)
		t.Logf("Router with %d handlers ready in: %s", len(topics), routerTime)
		t.Logf("Average per handler: %s", routerTime/time.Duration(len(topics)))
	case <-time.After(2 * time.Minute):
		t.Fatal("router didn't start in 2 minutes")
	}

	router.Close()
}

// =============================================================================
// Experiment 6: End-to-end simulation — server start + router + message
// This is the total cost of brainkit.New(Config{}) with embedded NATS.
// =============================================================================

func TestEmbeddedNATS_EndToEndStartup(t *testing.T) {
	storeDir := t.TempDir()
	wmLogger := watermill.NopLogger{}

	totalStart := time.Now()

	// Phase 1: Start embedded NATS
	phase1Start := time.Now()
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
		JetStream: true,
		StoreDir:  storeDir,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		t.Fatal("not ready")
	}
	phase1Time := time.Since(phase1Start)

	// Phase 2: Create Watermill transport
	phase2Start := time.Now()
	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	pub, _ := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               ns.ClientURL(),
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, wmLogger)
	defer pub.Close()

	sub, _ := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               ns.ClientURL(),
		QueueGroupPrefix:  "brainkit",
		SubscribersCount:  1,
		CloseTimeout:      5 * time.Second,
		AckWaitTimeout:    10 * time.Second,
		SubscribeTimeout:  10 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision: true,
			DurablePrefix: "brainkit",
			TrackMsgId:    true,
		},
	}, wmLogger)
	defer sub.Close()
	phase2Time := time.Since(phase2Start)

	// Phase 3: Start router with 32 handlers
	phase3Start := time.Now()
	router, _ := message.NewRouter(message.RouterConfig{}, wmLogger)
	noopHandler := func(msg *message.Message) ([]*message.Message, error) {
		msg.Ack()
		return nil, nil
	}
	for i := 0; i < 32; i++ {
		topic := fmt.Sprintf("test-cmd-%d", i)
		router.AddHandler("h-"+topic, topic, sub, topic+"-out", pub, noopHandler)
	}

	go func() { _ = router.Run(context.Background()) }()
	select {
	case <-router.Running():
	case <-time.After(2 * time.Minute):
		t.Fatal("router timeout")
	}
	phase3Time := time.Since(phase3Start)
	defer router.Close()

	totalTime := time.Since(totalStart)

	t.Logf("=== End-to-end embedded NATS startup ===")
	t.Logf("Phase 1 (NATS server):     %s", phase1Time)
	t.Logf("Phase 2 (Watermill setup): %s", phase2Time)
	t.Logf("Phase 3 (Router + 32 cmd): %s", phase3Time)
	t.Logf("TOTAL:                     %s", totalTime)

	// Phase 4: Verify message delivery works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, _ := sub.Subscribe(ctx, "verify-topic")
	verifyStart := time.Now()
	pub.Publish("verify-topic", message.NewMessage(watermill.NewUUID(), []byte(`{"verify":true}`)))
	select {
	case m := <-ch:
		m.Ack()
		t.Logf("Verification msg:          %s", time.Since(verifyStart))
	case <-ctx.Done():
		t.Fatal("verification timeout")
	}
}

// =============================================================================
// Experiment 7: Consumer groups on embedded NATS
// Verify that queue groups work correctly for competing consumers.
// =============================================================================

func TestEmbeddedNATS_ConsumerGroups(t *testing.T) {
	ns := startEmbeddedServer(t)
	wmLogger := watermill.NopLogger{}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	pub, _ := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               ns.ClientURL(),
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, wmLogger)
	defer pub.Close()

	// Two subscribers in the same consumer group — should compete
	makeSub := func(name string) message.Subscriber {
		sub, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
			URL:               ns.ClientURL(),
			QueueGroupPrefix:  "same-group",
			SubscribersCount:  1,
			CloseTimeout:      5 * time.Second,
			AckWaitTimeout:    10 * time.Second,
			SubscribeTimeout:  10 * time.Second,
			Unmarshaler:       wmnats.JSONMarshaler{},
			SubjectCalculator: natsSubjectCalc,
			JetStream: wmnats.JetStreamConfig{
				AutoProvision: true,
				DurablePrefix: "same-group",
				TrackMsgId:    true,
			},
		}, wmLogger)
		if err != nil {
			t.Fatalf("NewSubscriber(%s): %v", name, err)
		}
		return sub
	}

	sub1 := makeSub("sub1")
	defer sub1.Close()
	sub2 := makeSub("sub2")
	defer sub2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ch1, _ := sub1.Subscribe(ctx, "compete-topic")
	ch2, _ := sub2.Subscribe(ctx, "compete-topic")

	// Give subscriptions a moment to register
	time.Sleep(500 * time.Millisecond)

	// Send 20 messages
	for i := 0; i < 20; i++ {
		msg := message.NewMessage(watermill.NewUUID(), []byte(fmt.Sprintf(`{"seq":%d}`, i)))
		if err := pub.Publish("compete-topic", msg); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}

	var count1, count2 atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)

	collect := func(ch <-chan *message.Message, counter *atomic.Int32) {
		defer wg.Done()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				msg.Ack()
				counter.Add(1)
			case <-time.After(3 * time.Second):
				return
			}
		}
	}

	go collect(ch1, &count1)
	go collect(ch2, &count2)
	wg.Wait()

	total := count1.Load() + count2.Load()
	t.Logf("Sub1 received: %d", count1.Load())
	t.Logf("Sub2 received: %d", count2.Load())
	t.Logf("Total:         %d (expected 20)", total)

	if total != 20 {
		t.Errorf("expected 20 total messages, got %d", total)
	}
	// Both should have received SOME messages (competing, not fan-out)
	if count1.Load() == 0 || count2.Load() == 0 {
		t.Logf("WARNING: one subscriber got all messages — queue group may not be working as expected")
	}
}

// =============================================================================
// Experiment 8: Fan-out on embedded NATS
// Verify that subscribers WITHOUT queue groups all receive every message.
// =============================================================================

func TestEmbeddedNATS_FanOut(t *testing.T) {
	ns := startEmbeddedServer(t)
	wmLogger := watermill.NopLogger{}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	pub, _ := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               ns.ClientURL(),
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, wmLogger)
	defer pub.Close()

	// Two subscribers with DIFFERENT durable prefixes (unique consumer names) and NO queue group
	makeFanOutSub := func(id string) message.Subscriber {
		sub, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
			URL:               ns.ClientURL(),
			QueueGroupPrefix:  "", // NO queue group = fan-out
			SubscribersCount:  1,
			CloseTimeout:      5 * time.Second,
			AckWaitTimeout:    10 * time.Second,
			SubscribeTimeout:  10 * time.Second,
			Unmarshaler:       wmnats.JSONMarshaler{},
			SubjectCalculator: natsSubjectCalc,
			JetStream: wmnats.JetStreamConfig{
				AutoProvision: true,
				DurablePrefix: "fanout-" + id,
				TrackMsgId:    true,
			},
		}, wmLogger)
		if err != nil {
			t.Fatalf("NewSubscriber(fanout-%s): %v", id, err)
		}
		return sub
	}

	sub1 := makeFanOutSub("a")
	defer sub1.Close()
	sub2 := makeFanOutSub("b")
	defer sub2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ch1, _ := sub1.Subscribe(ctx, "fanout-topic")
	ch2, _ := sub2.Subscribe(ctx, "fanout-topic")

	time.Sleep(500 * time.Millisecond)

	// Send 10 messages
	for i := 0; i < 10; i++ {
		msg := message.NewMessage(watermill.NewUUID(), []byte(fmt.Sprintf(`{"seq":%d}`, i)))
		pub.Publish("fanout-topic", msg)
	}

	var count1, count2 atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)

	collect := func(ch <-chan *message.Message, counter *atomic.Int32) {
		defer wg.Done()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				msg.Ack()
				counter.Add(1)
			case <-time.After(3 * time.Second):
				return
			}
		}
	}

	go collect(ch1, &count1)
	go collect(ch2, &count2)
	wg.Wait()

	t.Logf("Sub1 received: %d (expect 10)", count1.Load())
	t.Logf("Sub2 received: %d (expect 10)", count2.Load())

	if count1.Load() != 10 {
		t.Errorf("fan-out sub1: expected 10, got %d", count1.Load())
	}
	if count2.Load() != 10 {
		t.Errorf("fan-out sub2: expected 10, got %d", count2.Load())
	}
}

// =============================================================================
// Experiment 9: Persistence across server restart
// =============================================================================

func TestEmbeddedNATS_Persistence(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "nats-data")
	os.MkdirAll(storeDir, 0755)
	wmLogger := watermill.NopLogger{}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: queueGroupPrefix}
	}

	// Start server, publish messages, shut down
	ns1 := startEmbeddedServerWithDir(t, storeDir)
	pub1, _ := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               ns1.ClientURL(),
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, wmLogger)

	for i := 0; i < 5; i++ {
		msg := message.NewMessage(watermill.NewUUID(), []byte(fmt.Sprintf(`{"seq":%d}`, i)))
		if err := pub1.Publish("persist-topic", msg); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}
	pub1.Close()
	ns1.Shutdown()
	ns1.WaitForShutdown()
	t.Log("Server 1 shut down after publishing 5 messages")

	// Restart server on same StoreDir, subscribe, should get the messages
	ns2 := startEmbeddedServerWithDir(t, storeDir)
	defer func() { ns2.Shutdown(); ns2.WaitForShutdown() }()

	sub2, _ := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               ns2.ClientURL(),
		QueueGroupPrefix:  "persist-group",
		SubscribersCount:  1,
		CloseTimeout:      5 * time.Second,
		AckWaitTimeout:    10 * time.Second,
		SubscribeTimeout:  10 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision: true,
			DurablePrefix: "persist-group",
			TrackMsgId:    true,
		},
	}, wmLogger)
	defer sub2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, _ := sub2.Subscribe(ctx, "persist-topic")

	var count int
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				goto done
			}
			msg.Ack()
			count++
			if count >= 5 {
				goto done
			}
		case <-time.After(5 * time.Second):
			goto done
		}
	}
done:
	t.Logf("Received %d messages after restart (expected 5)", count)
	if count < 5 {
		t.Errorf("persistence: expected 5 messages after restart, got %d", count)
	}
}

// =============================================================================
// Experiment 10: Multiple embedded servers in same process
// Tests need parallel embedded NATS instances. Verify they don't conflict.
// =============================================================================

func TestEmbeddedNATS_MultipleInstances(t *testing.T) {
	servers := make([]*server.Server, 5)
	for i := range servers {
		storeDir := t.TempDir()
		opts := &server.Options{
			Host:      "127.0.0.1",
			Port:      -1,
			NoLog:     true,
			NoSigs:    true,
			JetStream: true,
			StoreDir:  storeDir,
		}
		ns, err := server.NewServer(opts)
		if err != nil {
			t.Fatalf("NewServer[%d]: %v", i, err)
		}
		ns.Start()
		if !ns.ReadyForConnections(10 * time.Second) {
			t.Fatalf("Server[%d] not ready", i)
		}
		servers[i] = ns
		t.Logf("Server[%d] at %s", i, ns.ClientURL())
	}

	// Verify each server is independent
	for i, ns := range servers {
		nc, err := nats.Connect(ns.ClientURL())
		if err != nil {
			t.Fatalf("Connect[%d]: %v", i, err)
		}
		nc.Close()
	}

	t.Logf("All %d embedded servers running independently on different ports", len(servers))

	for _, ns := range servers {
		ns.Shutdown()
		ns.WaitForShutdown()
	}
}

// =============================================================================
// Helpers
// =============================================================================

func startEmbeddedServer(t *testing.T) *server.Server {
	t.Helper()
	return startEmbeddedServerWithDir(t, t.TempDir())
}

func startEmbeddedServerWithDir(t *testing.T, storeDir string) *server.Server {
	t.Helper()
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
		JetStream: true,
		StoreDir:  storeDir,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		t.Fatal("server not ready")
	}
	t.Cleanup(func() {
		ns.Shutdown()
		ns.WaitForShutdown()
	})
	return ns
}
