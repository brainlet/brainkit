package scheduling

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
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
	resp, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "every 10m",
		Topic:      topic,
		Payload:    json.RawMessage(`{"test":true}`),
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if resp.ID == "" {
		t.Fatal("expected non-empty schedule ID")
	}
}

func testScheduleCreateInvalidExpression(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "bad expression",
		Topic:      "test.sched.invalid",
	})
	if err == nil {
		t.Fatal("expected error for bad expression")
	}
}

func testScheduleListViaBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create 2 schedules
	for i := 0; i < 2; i++ {
		_, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
			Expression: "every 1h",
			Topic:      "test.sched.list." + uuid.NewString()[:8],
		})
		if err != nil {
			t.Fatalf("create schedule %d: %v", i, err)
		}
	}

	// List
	resp, err := sdk.Call[schedulemsg.ScheduleListMsg, schedulemsg.ScheduleListResp](env.Kit, ctx, schedulemsg.ScheduleListMsg{})
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}
	if len(resp.Schedules) < 2 {
		t.Fatalf("expected ≥2 schedules, got %d", len(resp.Schedules))
	}
}

func testScheduleCancelViaBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create
	createResp, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "every 1h",
		Topic:      "test.sched.cancel." + uuid.NewString()[:8],
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	// Cancel
	cancelResp, err := sdk.Call[schedulemsg.ScheduleCancelMsg, schedulemsg.ScheduleCancelResp](env.Kit, ctx, schedulemsg.ScheduleCancelMsg{ID: createResp.ID})
	if err != nil {
		t.Fatalf("cancel schedule: %v", err)
	}
	if !cancelResp.Cancelled {
		t.Fatal("expected Cancelled=true")
	}
}

func testScheduleCreateBlocksCommandTopic(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "every 1m",
		Topic:      "tools.call", // command topic — should be blocked
	})
	if err == nil {
		t.Fatal("expected error for command topic")
	}
	if !strings.Contains(err.Error(), "command topic") {
		t.Fatalf("expected 'command topic' in error, got: %s", err)
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
	createResp, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "every 300ms",
		Topic:      topic,
		Payload:    json.RawMessage(`{"tick":true}`),
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
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
	_, _ = sdk.Call[schedulemsg.ScheduleCancelMsg, schedulemsg.ScheduleCancelResp](env.Kit, ctx, schedulemsg.ScheduleCancelMsg{ID: createResp.ID})
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

	_, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "in 300ms",
		Topic:      topic,
		Payload:    json.RawMessage(`{"once":true}`),
	})
	if err != nil {
		t.Fatalf("create one-time schedule: %v", err)
	}

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
	_, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "in 200ms",
		Topic:      topic,
		Payload:    json.RawMessage(expectedPayload),
	})
	if err != nil {
		t.Fatalf("create payload schedule: %v", err)
	}

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
	createResp, err := sdk.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](env.Kit, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "every 200ms",
		Topic:      topic,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	// Wait for 2 fires
	for i := 0; i < 2; i++ {
		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for fire")
		}
	}

	// Cancel
	if _, err := sdk.Call[schedulemsg.ScheduleCancelMsg, schedulemsg.ScheduleCancelResp](env.Kit, ctx, schedulemsg.ScheduleCancelMsg{ID: createResp.ID}); err != nil {
		t.Fatalf("cancel schedule: %v", err)
	}

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
