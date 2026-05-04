package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	controlmod "github.com/brainlet/brainkit/modules/control"
	evalmod "github.com/brainlet/brainkit/modules/eval"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	healthmod "github.com/brainlet/brainkit/modules/health"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
	"github.com/brainlet/brainkit/sdk"
)

// callerHolder is implemented by *brainkit.Kit. Used so roundTrip keeps its
// sdk.Runtime signature without importing brainkit here.
type callerHolder interface {
	Caller() *sdk.Caller
}

type moduleKit interface {
	Module(id string) (bkmodule.Module, bool)
	Mount(context.Context, bkmodule.Module) error
}

func ensureHealthModule(rt sdk.Runtime) bool {
	k, ok := rt.(moduleKit)
	if !ok {
		return true
	}
	if _, ok := k.Module("health"); ok {
		return true
	}
	if _, err := roundTrip(rt, healthmod.KitHealthMsg{}, time.Millisecond); errors.Is(err, sdk.ErrCallerClosed) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = k.Mount(ctx, healthmod.New())
	return true
}

func ensureControlModule(rt sdk.Runtime) bool {
	k, ok := rt.(moduleKit)
	if !ok {
		return true
	}
	if _, ok := k.Module("control"); ok {
		return true
	}
	if _, err := roundTrip(rt, controlmod.ClusterPeersMsg{}, time.Millisecond); errors.Is(err, sdk.ErrCallerClosed) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = k.Mount(ctx, controlmod.New())
	return true
}

func ensureEvalModule(rt sdk.Runtime) bool {
	k, ok := rt.(moduleKit)
	if !ok {
		return true
	}
	if _, ok := k.Module("eval"); ok {
		return true
	}
	if _, err := roundTrip(rt, evalmsg.KitEvalMsg{}, time.Millisecond); errors.Is(err, sdk.ErrCallerClosed) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = k.Mount(ctx, evalmod.New())
	return true
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
	reply, err := c.Call(ctx, msg.BusTopic(), payload, sdk.CallerConfig{})
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
	msg := packagemsg.PackageDeployMsg{
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
	_, err = decodeResp[packagemsg.PackageDeployResp](payload)
	return err
}

func DeployWithResources(t *testing.T, rt sdk.Runtime, source, code string) []sdk.ResourceInfo {
	t.Helper()
	name := strings.TrimSuffix(source, ".ts")
	msg := packagemsg.PackageDeployMsg{
		Manifest: inlineManifest(name, source),
		Files:    map[string]string{source: code},
	}
	payload, err := roundTrip(rt, msg, 15*time.Second)
	if err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
	resp, err := decodeResp[packagemsg.PackageDeployResp](payload)
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
	if !ensureEvalModule(rt) {
		return "", sdk.ErrCallerClosed
	}
	payload, err := roundTrip(rt, evalmsg.KitEvalMsg{Source: source, Code: code, Mode: "ts"}, 15*time.Second)
	if err != nil {
		return "", err
	}
	resp, err := decodeResp[evalmsg.KitEvalResp](payload)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

// ── SetDraining ─────────────────────────────────────────────────────────────

func SetDraining(t *testing.T, rt sdk.Runtime, draining bool) {
	t.Helper()
	ensureControlModule(rt)
	_, err := roundTrip(rt, controlmod.KitSetDrainingMsg{Draining: draining}, 5*time.Second)
	if err != nil {
		t.Fatalf("SetDraining: %v", err)
	}
}

// ── Teardown ────────────────────────────────────────────────────────────────

func Teardown(t *testing.T, rt sdk.Runtime, source string) {
	t.Helper()
	name := strings.TrimSuffix(source, ".ts")
	payload, err := roundTrip(rt, packagemsg.PackageTeardownMsg{Name: name}, 10*time.Second)
	if err != nil {
		t.Fatalf("Teardown(%s): %v", source, err)
	}
	if _, err := decodeResp[packagemsg.PackageTeardownResp](payload); err != nil {
		t.Fatalf("Teardown(%s): %v", source, err)
	}
}

// ── ListDeployments ─────────────────────────────────────────────────────────

func ListDeployments(t *testing.T, rt sdk.Runtime) []packagemsg.DeployedPackageInfo {
	t.Helper()
	payload, err := roundTrip(rt, packagemsg.PackageListDeployedMsg{}, 10*time.Second)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	resp, err := decodeResp[packagemsg.PackageListDeployedResp](payload)
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
	payload, err := roundTrip(rt, schedulemsg.ScheduleCreateMsg{
		Expression: expression, Topic: topic, Payload: schedPayload,
	}, 10*time.Second)
	if err != nil {
		return "", err
	}
	resp, err := decodeResp[schedulemsg.ScheduleCreateResp](payload)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ── Unschedule ──────────────────────────────────────────────────────────────

func Unschedule(t *testing.T, rt sdk.Runtime, id string) {
	t.Helper()
	payload, err := roundTrip(rt, schedulemsg.ScheduleCancelMsg{ID: id}, 5*time.Second)
	if err != nil {
		t.Fatalf("Unschedule(%s): %v", id, err)
	}
	if _, err := decodeResp[schedulemsg.ScheduleCancelResp](payload); err != nil {
		t.Fatalf("Unschedule(%s): %v", id, err)
	}
}

// ── ListSchedules ───────────────────────────────────────────────────────────

func ListSchedules(t *testing.T, rt sdk.Runtime) []schedulemsg.ScheduleInfo {
	t.Helper()
	payload, err := roundTrip(rt, schedulemsg.ScheduleListMsg{}, 5*time.Second)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	resp, err := decodeResp[schedulemsg.ScheduleListResp](payload)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	return resp.Schedules
}

// ── Alive ───────────────────────────────────────────────────────────────────

func Alive(t *testing.T, rt sdk.Runtime) bool {
	t.Helper()
	if !ensureHealthModule(rt) {
		return false
	}
	_, err := roundTrip(rt, healthmod.KitHealthMsg{}, 5*time.Second)
	return err == nil
}

// ── EvalModule ──────────────────────────────────────────────────────────────

// EvalModule evaluates code as an ES module (supports import statements).
// Different from Deploy which uses EvalTS (no import support).
func EvalModule(t *testing.T, rt sdk.Runtime, source, code string) {
	t.Helper()
	ensureEvalModule(rt)
	payload, err := roundTrip(rt, evalmsg.KitEvalMsg{Source: source, Code: code, Mode: "module"}, 15*time.Second)
	if err != nil {
		t.Fatalf("EvalModule(%s): %v", source, err)
	}
	if _, err := decodeResp[evalmsg.KitEvalResp](payload); err != nil {
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
	unsub, err := sdk.SubscribeTo[pluginmsg.PluginRegisteredEvent](rt, ctx, "plugin.registered",
		func(evt pluginmsg.PluginRegisteredEvent, _ sdk.Message) {
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
