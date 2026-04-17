package scheduling

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
)

// ── Bus plumbing tests ───────────────────────────────────────────────────────

func testScheduleCreateViaBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test.sched.create." + uuid.NewString()[:8]
	pr, err := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "every 10m",
		Topic:      topic,
		Payload:    json.RawMessage(`{"test":true}`),
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	type createResult struct {
		resp sdk.ScheduleCreateResp
		msg  sdk.Message
	}
	respCh := make(chan createResult, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { respCh <- createResult{resp, msg} })
	defer unsub()

	select {
	case r := <-respCh:
		if errMsg := suite.ResponseErrorMessage(r.msg.Payload); errMsg != "" {
			t.Fatalf("error: %s", errMsg)
		}
		if r.resp.ID == "" {
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

	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "bad expression",
		Topic:      "test.sched.invalid",
	})

	type createResult struct {
		resp sdk.ScheduleCreateResp
		msg  sdk.Message
	}
	respCh := make(chan createResult, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { respCh <- createResult{resp, msg} })
	defer unsub()

	select {
	case r := <-respCh:
		if suite.ResponseErrorMessage(r.msg.Payload) == "" {
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
		pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
			Expression: "every 1h",
			Topic:      "test.sched.list." + uuid.NewString()[:8],
		})
		ch := make(chan sdk.ScheduleCreateResp, 1)
		unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
			func(resp sdk.ScheduleCreateResp, msg sdk.Message) { ch <- resp })
		<-ch
		unsub()
	}

	// List
	pr, _ := sdk.PublishScheduleList(env.Kit, ctx, sdk.ScheduleListMsg{})
	listCh := make(chan sdk.ScheduleListResp, 1)
	unsub, _ := sdk.SubscribeScheduleListResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleListResp, msg sdk.Message) { listCh <- resp })
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
	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "every 1h",
		Topic:      "test.sched.cancel." + uuid.NewString()[:8],
	})
	createCh := make(chan sdk.ScheduleCreateResp, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { createCh <- resp })
	createResp := <-createCh
	unsub()

	// Cancel
	pr2, _ := sdk.PublishScheduleCancel(env.Kit, ctx, sdk.ScheduleCancelMsg{ID: createResp.ID})
	cancelCh := make(chan sdk.ScheduleCancelResp, 1)
	unsub2, _ := sdk.SubscribeScheduleCancelResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp sdk.ScheduleCancelResp, msg sdk.Message) { cancelCh <- resp })
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

	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "every 1m",
		Topic:      "tools.call", // command topic — should be blocked
	})
	type createResult struct {
		resp sdk.ScheduleCreateResp
		msg  sdk.Message
	}
	respCh := make(chan createResult, 1)
	unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { respCh <- createResult{resp, msg} })
	defer unsub()

	select {
	case r := <-respCh:
		errMsg := suite.ResponseErrorMessage(r.msg.Payload)
		if errMsg == "" {
			t.Fatal("expected error for command topic")
		}
		if !strings.Contains(errMsg, "command topic") {
			t.Fatalf("expected 'command topic' in error, got: %s", errMsg)
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
	unsub, err := env.Kit.SubscribeRaw(ctx, topic, func(msg sdk.Message) {
		received <- struct{}{}
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	// Create fast schedule
	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "every 300ms",
		Topic:      topic,
		Payload:    json.RawMessage(`{"tick":true}`),
	})
	type createResult struct {
		resp sdk.ScheduleCreateResp
		msg  sdk.Message
	}
	createCh := make(chan createResult, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { createCh <- createResult{resp, msg} })
	cr := <-createCh
	createResp := cr.resp
	unsubCreate()

	if errMsg := suite.ResponseErrorMessage(cr.msg.Payload); errMsg != "" {
		t.Fatalf("create error: %s", errMsg)
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
	sdk.PublishScheduleCancel(env.Kit, ctx, sdk.ScheduleCancelMsg{ID: createResp.ID})
}

func testScheduleCreateOneTimeFires(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test.sched.once." + uuid.NewString()[:8]
	received := make(chan json.RawMessage, 5)

	unsub, _ := env.Kit.SubscribeRaw(ctx, topic, func(msg sdk.Message) {
		received <- msg.Payload
	})
	defer unsub()

	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "in 300ms",
		Topic:      topic,
		Payload:    json.RawMessage(`{"once":true}`),
	})
	createCh := make(chan sdk.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { createCh <- resp })
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

	unsub, _ := env.Kit.SubscribeRaw(ctx, topic, func(msg sdk.Message) {
		received <- msg.Payload
	})
	defer unsub()

	expectedPayload := `{"key":"value","num":42}`
	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "in 200ms",
		Topic:      topic,
		Payload:    json.RawMessage(expectedPayload),
	})
	createCh := make(chan sdk.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { createCh <- resp })
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

	unsub, _ := env.Kit.SubscribeRaw(ctx, topic, func(msg sdk.Message) {
		received <- struct{}{}
	})
	defer unsub()

	// Create fast schedule
	pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "every 200ms",
		Topic:      topic,
	})
	createCh := make(chan sdk.ScheduleCreateResp, 1)
	unsubCreate, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, msg sdk.Message) { createCh <- resp })
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
	pr2, _ := sdk.PublishScheduleCancel(env.Kit, ctx, sdk.ScheduleCancelMsg{ID: createResp.ID})
	cancelCh := make(chan sdk.ScheduleCancelResp, 1)
	unsubCancel, _ := sdk.SubscribeScheduleCancelResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp sdk.ScheduleCancelResp, msg sdk.Message) { cancelCh <- resp })
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
