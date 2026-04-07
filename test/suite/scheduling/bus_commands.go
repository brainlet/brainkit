package scheduling

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
)

// ── Bus plumbing tests ───────────────────────────────────────────────────────

func testScheduleCreateViaBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test.sched.create." + uuid.NewString()[:8]
	pr, err := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "every 10m",
		Topic:      topic,
		Payload:    json.RawMessage(`{"test":true}`),
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	respCh := make(chan messages.ScheduleCreateResp, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { respCh <- resp })
	defer unsub()

	select {
	case resp := <-respCh:
		if resp.Error != "" {
			t.Fatalf("error: %s", resp.Error)
		}
		if resp.ID == "" {
			t.Fatal("expected non-empty schedule ID")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testScheduleCreateInvalidExpression(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "bad expression",
		Topic:      "test.sched.invalid",
	})

	respCh := make(chan messages.ScheduleCreateResp, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { respCh <- resp })
	defer unsub()

	select {
	case resp := <-respCh:
		if resp.Error == "" {
			t.Fatal("expected error for bad expression")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testScheduleListViaBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create 2 schedules
	for i := 0; i < 2; i++ {
		pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
			Expression: "every 1h",
			Topic:      "test.sched.list." + uuid.NewString()[:8],
		})
		ch := make(chan messages.ScheduleCreateResp, 1)
		unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
			func(resp messages.ScheduleCreateResp, msg messages.Message) { ch <- resp })
		<-ch
		unsub()
	}

	// List
	pr, _ := sdk.PublishScheduleList(env.Kernel, ctx, messages.ScheduleListMsg{})
	listCh := make(chan messages.ScheduleListResp, 1)
	unsub, _ := sdk.SubscribeScheduleListResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleListResp, msg messages.Message) { listCh <- resp })
	defer unsub()

	select {
	case resp := <-listCh:
		if len(resp.Schedules) < 2 {
			t.Fatalf("expected ≥2 schedules, got %d", len(resp.Schedules))
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testScheduleCancelViaBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create
	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "every 1h",
		Topic:      "test.sched.cancel." + uuid.NewString()[:8],
	})
	createCh := make(chan messages.ScheduleCreateResp, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { createCh <- resp })
	createResp := <-createCh
	unsub()

	// Cancel
	pr2, _ := sdk.PublishScheduleCancel(env.Kernel, ctx, messages.ScheduleCancelMsg{ID: createResp.ID})
	cancelCh := make(chan messages.ScheduleCancelResp, 1)
	unsub2, _ := sdk.SubscribeScheduleCancelResp(env.Kernel, ctx, pr2.ReplyTo,
		func(resp messages.ScheduleCancelResp, msg messages.Message) { cancelCh <- resp })
	defer unsub2()

	select {
	case resp := <-cancelCh:
		if !resp.Cancelled {
			t.Fatal("expected Cancelled=true")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testScheduleCreateBlocksCommandTopic(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "every 1m",
		Topic:      "tools.call", // command topic — should be blocked
	})
	respCh := make(chan messages.ScheduleCreateResp, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { respCh <- resp })
	defer unsub()

	select {
	case resp := <-respCh:
		if resp.Error == "" {
			t.Fatal("expected error for command topic")
		}
		if !strings.Contains(resp.Error, "command topic") {
			t.Fatalf("expected 'command topic' in error, got: %s", resp.Error)
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// ── Real effect tests ────────────────────────────────────────────────────────

func testScheduleCreateFiresOnTopic(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test.sched.fires." + uuid.NewString()[:8]
	received := make(chan struct{}, 10)

	// Subscribe to the target topic FIRST
	unsub, err := env.Kernel.SubscribeRaw(ctx, topic, func(msg messages.Message) {
		received <- struct{}{}
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	// Create fast schedule
	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "every 300ms",
		Topic:      topic,
		Payload:    json.RawMessage(`{"tick":true}`),
	})
	createCh := make(chan messages.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { createCh <- resp })
	createResp := <-createCh
	unsubCreate()

	if createResp.Error != "" {
		t.Fatalf("create error: %s", createResp.Error)
	}

	// Wait for at least 2 fires
	count := 0
	deadline := time.After(3 * time.Second)
	for count < 2 {
		select {
		case <-received:
			count++
		case <-deadline:
			t.Fatalf("expected ≥2 fires in 3s, got %d", count)
		}
	}

	// Cancel it
	sdk.PublishScheduleCancel(env.Kernel, ctx, messages.ScheduleCancelMsg{ID: createResp.ID})
}

func testScheduleCreateOneTimeFires(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test.sched.once." + uuid.NewString()[:8]
	received := make(chan json.RawMessage, 5)

	unsub, _ := env.Kernel.SubscribeRaw(ctx, topic, func(msg messages.Message) {
		received <- msg.Payload
	})
	defer unsub()

	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "in 300ms",
		Topic:      topic,
		Payload:    json.RawMessage(`{"once":true}`),
	})
	createCh := make(chan messages.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { createCh <- resp })
	<-createCh
	unsubCreate()

	// Wait for exactly 1 fire
	select {
	case payload := <-received:
		if !strings.Contains(string(payload), "once") {
			t.Fatalf("unexpected payload: %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for one-time fire")
	}

	// Verify no second fire
	select {
	case <-received:
		t.Fatal("one-time schedule fired twice")
	case <-time.After(1 * time.Second):
		// good — no second fire
	}
}

func testScheduleCreateWithPayload(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test.sched.payload." + uuid.NewString()[:8]
	received := make(chan json.RawMessage, 1)

	unsub, _ := env.Kernel.SubscribeRaw(ctx, topic, func(msg messages.Message) {
		received <- msg.Payload
	})
	defer unsub()

	expectedPayload := `{"key":"value","num":42}`
	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "in 200ms",
		Topic:      topic,
		Payload:    json.RawMessage(expectedPayload),
	})
	createCh := make(chan messages.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { createCh <- resp })
	<-createCh
	unsubCreate()

	select {
	case payload := <-received:
		var got map[string]any
		json.Unmarshal(payload, &got)
		if got["key"] != "value" || got["num"] != float64(42) {
			t.Fatalf("payload mismatch: %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for scheduled payload")
	}
}

func testScheduleCancelStopsFiring(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	topic := "test.sched.stop." + uuid.NewString()[:8]
	received := make(chan struct{}, 20)

	unsub, _ := env.Kernel.SubscribeRaw(ctx, topic, func(msg messages.Message) {
		received <- struct{}{}
	})
	defer unsub()

	// Create fast schedule
	pr, _ := sdk.PublishScheduleCreate(env.Kernel, ctx, messages.ScheduleCreateMsg{
		Expression: "every 200ms",
		Topic:      topic,
	})
	createCh := make(chan messages.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kernel, ctx, pr.ReplyTo,
		func(resp messages.ScheduleCreateResp, msg messages.Message) { createCh <- resp })
	createResp := <-createCh
	unsubCreate()

	// Wait for 2 fires
	for i := 0; i < 2; i++ {
		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for fire")
		}
	}

	// Cancel
	pr2, _ := sdk.PublishScheduleCancel(env.Kernel, ctx, messages.ScheduleCancelMsg{ID: createResp.ID})
	cancelCh := make(chan messages.ScheduleCancelResp, 1)
	unsubCancel, _ := sdk.SubscribeScheduleCancelResp(env.Kernel, ctx, pr2.ReplyTo,
		func(resp messages.ScheduleCancelResp, msg messages.Message) { cancelCh <- resp })
	<-cancelCh
	unsubCancel()

	// Drain any in-flight fires
	time.Sleep(300 * time.Millisecond)
	for len(received) > 0 {
		<-received
	}

	// Verify no more fires arrive in 1s
	select {
	case <-received:
		t.Fatal("schedule still firing after cancel")
	case <-time.After(1 * time.Second):
		// good
	}
}
