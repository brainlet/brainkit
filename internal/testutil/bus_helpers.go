package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// ── Core pattern: subscribe FIRST, then publish ─────────────────────────────
// GoChannel transport delivers synchronously during Publish. If we publish
// before subscribing, the response arrives before the subscription is set up.
// Fix: generate replyTo, subscribe to it, THEN publish with that replyTo.

// roundTrip subscribes to a replyTo topic, publishes a command, and waits for
// the raw response payload. This is the safe pattern for GoChannel transport.
func roundTrip(rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Generate replyTo and subscribe BEFORE publishing
	correlationID := uuid.NewString()
	replyTo := msg.BusTopic() + ".reply." + correlationID

	ch := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, replyTo, func(m sdk.Message) {
		select {
		case ch <- json.RawMessage(m.Payload):
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", replyTo, err)
	}
	defer unsub()

	// Now publish with the pre-subscribed replyTo
	if _, err := sdk.Publish(rt, ctx, msg, sdk.WithReplyTo(replyTo)); err != nil {
		return nil, fmt.Errorf("publish %s: %w", msg.BusTopic(), err)
	}

	select {
	case payload := <-ch:
		return payload, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", msg.BusTopic(), ctx.Err())
	}
}

// decodeResp unmarshals a response and checks for error field.
func decodeResp[T any](payload json.RawMessage) (T, error) {
	var resp T
	if err := json.Unmarshal(payload, &resp); err != nil {
		return resp, fmt.Errorf("decode response: %w", err)
	}
	// Check for error in ResultMeta
	var meta struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	json.Unmarshal(payload, &meta)
	if meta.Error != "" {
		return resp, fmt.Errorf("%s: %s", meta.Code, meta.Error)
	}
	return resp, nil
}

// ── Deploy ──────────────────────────────────────────────────────────────────

func Deploy(t *testing.T, rt sdk.Runtime, source, code string) {
	t.Helper()
	if err := DeployErr(rt, source, code); err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
}

func DeployErr(rt sdk.Runtime, source, code string) error {
	return DeployWithOpts(rt, source, code, "", "")
}

func DeployWithOpts(rt sdk.Runtime, source, code, role, packageName string) error {
	payload, err := roundTrip(rt, sdk.KitDeployMsg{
		Source: source, Code: code, Role: role, PackageName: packageName,
	}, 15*time.Second)
	if err != nil {
		return err
	}
	_, err = decodeResp[sdk.KitDeployResp](payload)
	return err
}

func DeployWithResources(t *testing.T, rt sdk.Runtime, source, code string) []sdk.ResourceInfo {
	t.Helper()
	payload, err := roundTrip(rt, sdk.KitDeployMsg{Source: source, Code: code}, 15*time.Second)
	if err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
	resp, err := decodeResp[sdk.KitDeployResp](payload)
	if err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
	return resp.Resources
}

// ── EvalTS ──────────────────────────────────────────────────────────────────

func EvalTS(t *testing.T, rt sdk.Runtime, source, code string) string {
	t.Helper()
	result, err := EvalTSErr(rt, source, code)
	if err != nil {
		t.Fatalf("EvalTS(%s): %v", source, err)
	}
	return result
}

func EvalTSErr(rt sdk.Runtime, source, code string) (string, error) {
	payload, err := roundTrip(rt, sdk.KitEvalTSMsg{Source: source, Code: code}, 15*time.Second)
	if err != nil {
		return "", err
	}
	resp, err := decodeResp[sdk.KitEvalTSResp](payload)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

// ── SetDraining ─────────────────────────────────────────────────────────────

func SetDraining(t *testing.T, rt sdk.Runtime, draining bool) {
	t.Helper()
	_, err := roundTrip(rt, sdk.KitSetDrainingMsg{Draining: draining}, 5*time.Second)
	if err != nil {
		t.Fatalf("SetDraining: %v", err)
	}
}

// ── Teardown ────────────────────────────────────────────────────────────────

func Teardown(t *testing.T, rt sdk.Runtime, source string) {
	t.Helper()
	payload, err := roundTrip(rt, sdk.KitTeardownMsg{Source: source}, 10*time.Second)
	if err != nil {
		t.Fatalf("Teardown(%s): %v", source, err)
	}
	if _, err := decodeResp[sdk.KitTeardownResp](payload); err != nil {
		t.Fatalf("Teardown(%s): %v", source, err)
	}
}

// ── ListDeployments ─────────────────────────────────────────────────────────

func ListDeployments(t *testing.T, rt sdk.Runtime) []sdk.DeploymentInfo {
	t.Helper()
	payload, err := roundTrip(rt, sdk.KitListMsg{}, 10*time.Second)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	resp, err := decodeResp[sdk.KitListResp](payload)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	return resp.Deployments
}

// ── Schedule ────────────────────────────────────────────────────────────────

func Schedule(t *testing.T, rt sdk.Runtime, expression, topic string, payload json.RawMessage) string {
	t.Helper()
	id, err := ScheduleErr(rt, expression, topic, payload)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	return id
}

func ScheduleErr(rt sdk.Runtime, expression, topic string, schedPayload json.RawMessage) (string, error) {
	payload, err := roundTrip(rt, sdk.ScheduleCreateMsg{
		Expression: expression, Topic: topic, Payload: schedPayload,
	}, 10*time.Second)
	if err != nil {
		return "", err
	}
	resp, err := decodeResp[sdk.ScheduleCreateResp](payload)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ── Unschedule ──────────────────────────────────────────────────────────────

func Unschedule(t *testing.T, rt sdk.Runtime, id string) {
	t.Helper()
	payload, err := roundTrip(rt, sdk.ScheduleCancelMsg{ID: id}, 5*time.Second)
	if err != nil {
		t.Fatalf("Unschedule(%s): %v", id, err)
	}
	if _, err := decodeResp[sdk.ScheduleCancelResp](payload); err != nil {
		t.Fatalf("Unschedule(%s): %v", id, err)
	}
}

// ── ListSchedules ───────────────────────────────────────────────────────────

func ListSchedules(t *testing.T, rt sdk.Runtime) []sdk.ScheduleInfo {
	t.Helper()
	payload, err := roundTrip(rt, sdk.ScheduleListMsg{}, 5*time.Second)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	resp, err := decodeResp[sdk.ScheduleListResp](payload)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	return resp.Schedules
}

// ── Alive ───────────────────────────────────────────────────────────────────

func Alive(t *testing.T, rt sdk.Runtime) bool {
	t.Helper()
	_, err := roundTrip(rt, sdk.KitHealthMsg{}, 5*time.Second)
	return err == nil
}

// ── EvalModule ──────────────────────────────────────────────────────────────

// EvalModule evaluates code as an ES module (supports import statements).
// Different from Deploy which uses EvalTS (no import support).
func EvalModule(t *testing.T, rt sdk.Runtime, source, code string) {
	t.Helper()
	payload, err := roundTrip(rt, sdk.KitEvalModuleMsg{Source: source, Code: code}, 15*time.Second)
	if err != nil {
		t.Fatalf("EvalModule(%s): %v", source, err)
	}
	if _, err := decodeResp[sdk.KitEvalModuleResp](payload); err != nil {
		t.Fatalf("EvalModule(%s): %v", source, err)
	}
}

// ── WaitForPlugin ───────────────────────────────────────────────────────────

// WaitForPlugin waits for a plugin to register by subscribing to the plugin.registered event.
// Replaces time.Sleep(3s) in e2e tests.
func WaitForPlugin(t *testing.T, rt sdk.Runtime, pluginName string, timeout time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan struct{}, 1)
	unsub, err := sdk.SubscribeTo[sdk.PluginRegisteredEvent](rt, ctx, "plugin.registered",
		func(evt sdk.PluginRegisteredEvent, _ sdk.Message) {
			if evt.Name == pluginName {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		})
	if err != nil {
		t.Fatalf("WaitForPlugin: subscribe: %v", err)
	}
	defer unsub()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		t.Fatalf("WaitForPlugin(%s): timeout after %v", pluginName, timeout)
	}
}

// ── PublishAndWait (raw) ────────────────────────────────────────────────────

func PublishAndWait(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) json.RawMessage {
	t.Helper()
	payload, err := roundTrip(rt, msg, timeout)
	if err != nil {
		t.Fatalf("PublishAndWait %s: %v", msg.BusTopic(), err)
	}
	return payload
}
