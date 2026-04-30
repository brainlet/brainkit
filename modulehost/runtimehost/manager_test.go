package runtimehost

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/systemmsg"
)

func TestRedeployPersistedDeploymentsSortsAndRestores(t *testing.T) {
	store := &fakeStore{deployments: []types.PersistedDeployment{
		{Source: "b.ts", Code: "b", Order: 2, PackageName: "pkg"},
		{Source: "a.ts", Code: "a", Order: 1},
	}}
	host := &fakeHost{hasRuntime: true}
	manager := New(host, nil, nil)

	manager.RedeployPersistedDeployments(types.KernelConfig{Store: store})

	if host.seed != 2 {
		t.Fatalf("deploy order seed = %d, want 2", host.seed)
	}
	if len(host.deploys) != 2 {
		t.Fatalf("deploy count = %d, want 2", len(host.deploys))
	}
	if host.deploys[0].source != "a.ts" || host.deploys[1].source != "b.ts" {
		t.Fatalf("deploy order = %#v", host.deploys)
	}
	if !host.deploys[0].cfg.Restoring || !host.deploys[1].cfg.Restoring {
		t.Fatalf("redeploys must use WithRestoring: %#v", host.deploys)
	}
	if host.deploys[1].cfg.PackageName != "pkg" {
		t.Fatalf("package name = %q, want pkg", host.deploys[1].cfg.PackageName)
	}
}

func TestSubscribeToDeploymentPropagationMirrorsRemoteEvents(t *testing.T) {
	store := &fakeStore{deploymentBySource: map[string]types.PersistedDeployment{
		"remote.ts": {Source: "remote.ts", Code: "code"},
	}}
	host := &fakeHost{hasRuntime: true}
	handlers := map[string]func(sdk.Message){}
	manager := New(host, nil, func(_ context.Context, topic string, handler func(sdk.Message)) (func(), error) {
		handlers[topic] = handler
		return func() {}, nil
	})

	manager.SubscribeToDeploymentPropagation(types.KernelConfig{
		RuntimeID: "self",
		Store:     store,
	})

	deployPayload, _ := json.Marshal(systemmsg.KitDeployedEvent{Source: "remote.ts", RuntimeID: "other"})
	handlers[systemmsg.TopicKitDeployed](sdk.Message{Payload: deployPayload})
	if len(host.deploys) != 1 || host.deploys[0].source != "remote.ts" || !host.deploys[0].cfg.Restoring {
		t.Fatalf("deploy propagation = %#v", host.deploys)
	}

	teardownPayload, _ := json.Marshal(systemmsg.KitTeardownedEvent{Source: "remote.ts", RuntimeID: "other"})
	handlers[systemmsg.TopicKitTeardowned](sdk.Message{Payload: teardownPayload})
	if len(host.teardowns) != 1 || host.teardowns[0] != "remote.ts" {
		t.Fatalf("teardown propagation = %#v", host.teardowns)
	}
}

func TestRestartActiveWorkflowsReportsJSFailures(t *testing.T) {
	host := &fakeHost{hasRuntime: true, callErr: errors.New("boom")}
	var gotErr error
	var gotCtx types.ErrorContext
	manager := New(host, nil, nil)

	manager.RestartActiveWorkflows(types.KernelConfig{
		ErrorHandler: func(err error, ctx types.ErrorContext) {
			gotErr = err
			gotCtx = ctx
		},
	})

	if gotErr == nil {
		t.Fatalf("expected restart failure to be reported")
	}
	if gotCtx.Operation != "RestartActiveWorkflows" || gotCtx.Component != "kernel" {
		t.Fatalf("error context = %#v", gotCtx)
	}
}

type fakeHost struct {
	hasRuntime bool
	seed       int32
	callErr    error
	deploys    []fakeDeploy
	teardowns  []string
}

type fakeDeploy struct {
	source string
	code   string
	cfg    types.DeployConfig
}

func (h *fakeHost) HasJSRuntime() bool { return h.hasRuntime }

func (h *fakeHost) SetDeployOrderSeed(seed int32) { h.seed = seed }

func (h *fakeHost) Deploy(_ context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	var cfg types.DeployConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	h.deploys = append(h.deploys, fakeDeploy{source: source, code: code, cfg: cfg})
	return nil, nil
}

func (h *fakeHost) Teardown(_ context.Context, source string) (int, error) {
	h.teardowns = append(h.teardowns, source)
	return 1, nil
}

func (h *fakeHost) ListDeployments() []runtimecap.DeploymentInfo { return nil }

func (h *fakeHost) CallJS(context.Context, string, any) (json.RawMessage, error) {
	if h.callErr != nil {
		return nil, h.callErr
	}
	return json.RawMessage(`{"restarted":0}`), nil
}

type fakeStore struct {
	deployments        []types.PersistedDeployment
	deploymentBySource map[string]types.PersistedDeployment
}

func (s *fakeStore) SaveDeployment(types.PersistedDeployment) error { return nil }
func (s *fakeStore) LoadDeployments() ([]types.PersistedDeployment, error) {
	return s.deployments, nil
}
func (s *fakeStore) LoadDeployment(source string) (types.PersistedDeployment, error) {
	return s.deploymentBySource[source], nil
}
func (s *fakeStore) DeleteDeployment(string) error { return nil }
func (s *fakeStore) SaveSchedule(types.PersistedSchedule) error {
	return nil
}
func (s *fakeStore) LoadSchedules() ([]types.PersistedSchedule, error) { return nil, nil }
func (s *fakeStore) DeleteSchedule(string) error                       { return nil }
func (s *fakeStore) ClaimScheduleFire(string, time.Time) (bool, error) { return true, nil }
func (s *fakeStore) SaveInstalledPlugin(types.InstalledPlugin) error   { return nil }
func (s *fakeStore) LoadInstalledPlugins() ([]types.InstalledPlugin, error) {
	return nil, nil
}
func (s *fakeStore) DeleteInstalledPlugin(string) error { return nil }
func (s *fakeStore) SaveRunningPlugin(types.RunningPluginRecord) error {
	return nil
}
func (s *fakeStore) LoadRunningPlugins() ([]types.RunningPluginRecord, error) {
	return nil, nil
}
func (s *fakeStore) DeleteRunningPlugin(string) error { return nil }
func (s *fakeStore) Close() error                     { return nil }
