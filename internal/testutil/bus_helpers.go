package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/bus/caller"
	"github.com/brainlet/brainkit/sdk"
)

// callerHolder is implemented by *brainkit.Kit. Used so roundTrip keeps its
// sdk.Runtime signature without importing brainkit here.
type callerHolder interface {
	Caller() *caller.Caller
}

// roundTrip sends msg via the Kit's shared-inbox Caller and returns the raw
// reply payload. Requires rt to expose Caller() (every *brainkit.Kit does).
func roundTrip(rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, error) {
	holder, ok := rt.(callerHolder)
	if !ok {
		return nil, fmt.Errorf("testutil.roundTrip: runtime does not expose a Caller")
	}
	c := holder.Caller()
	if c == nil {
		return nil, fmt.Errorf("testutil.roundTrip: caller not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal %T: %w", msg, err)
	}
	reply, err := c.Call(ctx, msg.BusTopic(), payload, caller.Config{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", msg.BusTopic(), err)
	}
	return reply, nil
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
	return DeployWithOpts(rt, source, code, "")
}

func DeployWithOpts(rt sdk.Runtime, source, code, packageName string) error {
	name := packageName
	if name == "" {
		name = strings.TrimSuffix(source, ".ts")
	}
	msg := sdk.PackageDeployMsg{
		Manifest: inlineManifest(name, source),
		Files:    map[string]string{source: code},
	}
	// AI-backed fixtures run top-level awaits inside the deploy
	// (Agent.generate, embedder probe, semantic recall). Budget
	// 60s to cover chained OpenAI calls + vector index creation.
	payload, err := roundTrip(rt, msg, 60*time.Second)
	if err != nil {
		return err
	}
	_, err = decodeResp[sdk.PackageDeployResp](payload)
	return err
}

func DeployWithResources(t *testing.T, rt sdk.Runtime, source, code string) []sdk.ResourceInfo {
	t.Helper()
	name := strings.TrimSuffix(source, ".ts")
	msg := sdk.PackageDeployMsg{
		Manifest: inlineManifest(name, source),
		Files:    map[string]string{source: code},
	}
	payload, err := roundTrip(rt, msg, 15*time.Second)
	if err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
	resp, err := decodeResp[sdk.PackageDeployResp](payload)
	if err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
	return resp.Resources
}

func inlineManifest(name, entry string) json.RawMessage {
	m := map[string]string{"name": name, "entry": entry}
	raw, _ := json.Marshal(m)
	return raw
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
	payload, err := roundTrip(rt, sdk.KitEvalMsg{Source: source, Code: code, Mode: "ts"}, 15*time.Second)
	if err != nil {
		return "", err
	}
	resp, err := decodeResp[sdk.KitEvalResp](payload)
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
	name := strings.TrimSuffix(source, ".ts")
	payload, err := roundTrip(rt, sdk.PackageTeardownMsg{Name: name}, 10*time.Second)
	if err != nil {
		t.Fatalf("Teardown(%s): %v", source, err)
	}
	if _, err := decodeResp[sdk.PackageTeardownResp](payload); err != nil {
		t.Fatalf("Teardown(%s): %v", source, err)
	}
}

// ── ListDeployments ─────────────────────────────────────────────────────────

func ListDeployments(t *testing.T, rt sdk.Runtime) []sdk.DeployedPackageInfo {
	t.Helper()
	payload, err := roundTrip(rt, sdk.PackageListDeployedMsg{}, 10*time.Second)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	resp, err := decodeResp[sdk.PackageListDeployedResp](payload)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	return resp.Packages
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
	payload, err := roundTrip(rt, sdk.KitEvalMsg{Source: source, Code: code, Mode: "module"}, 15*time.Second)
	if err != nil {
		t.Fatalf("EvalModule(%s): %v", source, err)
	}
	if _, err := decodeResp[sdk.KitEvalResp](payload); err != nil {
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
