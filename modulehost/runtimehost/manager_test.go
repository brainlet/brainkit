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
		{Source: "b.ts", Code: "b", Order: 2, PackageName: "pkg", ArtifactKind: types.DeployArtifactNormalizedJS},
		{Source: "a.ts", Code: "a", Order: 1},
		{Source: "c.ts", Code: "c", Order: 3, ArtifactKind: types.DeployArtifactNormalizedJS},
	}}
	host := &fakeHost{hasRuntime: true}
	manager := New(host, nil, nil)

	manager.RedeployPersistedDeployments(types.KernelConfig{Store: store})

	if host.seed != 3 {
		t.Fatalf("deploy order seed = %d, want 3", host.seed)
	}
	if len(host.deploys) != 3 {
		t.Fatalf("deploy count = %d, want 3", len(host.deploys))
	}
	if host.deploys[0].source != "a.ts" || host.deploys[1].source != "b.ts" || host.deploys[2].source != "c.ts" {
		t.Fatalf("deploy order = %#v", host.deploys)
	}
	if !host.deploys[0].cfg.Restoring || !host.deploys[1].cfg.Restoring || !host.deploys[2].cfg.Restoring {
		t.Fatalf("redeploys must use WithRestoring: %#v", host.deploys)
	}
	if host.deploys[1].cfg.PackageName != "pkg" {
		t.Fatalf("package name = %q, want pkg", host.deploys[1].cfg.PackageName)
	}
	if host.deploys[1].cfg.EffectiveArtifactKind() != types.DeployArtifactNormalizedJS {
		t.Fatalf("package redeploy must preserve normalized JS artifact boundary: %#v", host.deploys[1].cfg)
	}
	if host.deploys[2].cfg.EffectiveArtifactKind() != types.DeployArtifactNormalizedJS {
		t.Fatalf("non-package normalized artifact redeploy must preserve artifact kind: %#v", host.deploys[2].cfg)
	}
	if host.deploys[0].cfg.EffectiveArtifactKind() == types.DeployArtifactNormalizedJS {
		t.Fatalf("raw runtime redeploy should not be marked normalized JS: %#v", host.deploys[0].cfg)
	}
}

func TestSubscribeToDeploymentPropagationMirrorsRemoteEvents(t *testing.T) {
	store := &fakeStore{deploymentBySource: map[string]types.PersistedDeployment{
		"remote.ts": {Source: "remote.ts", Code: "code", PackageName: "pkg", ArtifactKind: types.DeployArtifactNormalizedJS},
	}}
	host := &fakeHost{hasRuntime: true}
	handlers := map[string]func(sdk.Message){}
	manager := New(host, nil, func(_ context.Context, topic string, handler func(sdk.Message)) (PropagationHandle, error) {
		handlers[topic] = handler
		return fakePropagationHandle{}, nil
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
	if host.deploys[0].cfg.EffectiveArtifactKind() != types.DeployArtifactNormalizedJS || host.deploys[0].cfg.PackageName != "pkg" {
		t.Fatalf("package deploy propagation must preserve package artifact metadata: %#v", host.deploys[0].cfg)
	}

	teardownPayload, _ := json.Marshal(systemmsg.KitTeardownedEvent{Source: "remote.ts", RuntimeID: "other"})
	handlers[systemmsg.TopicKitTeardowned](sdk.Message{Payload: teardownPayload})
	if len(host.teardowns) != 1 || host.teardowns[0] != "remote.ts" {
		t.Fatalf("teardown propagation = %#v", host.teardowns)
	}
}

func TestDeploymentPropagationSubscriptionsReplaceAndClose(t *testing.T) {
	store := &fakeStore{}
	host := &fakeHost{hasRuntime: true}
	var unsubscribed []string
	manager := New(host, nil, func(_ context.Context, topic string, _ func(sdk.Message)) (PropagationHandle, error) {
		return fakePropagationHandle{
			close: func(context.Context) error {
				unsubscribed = append(unsubscribed, topic)
				return nil
			},
		}, nil
	})

	cfg := types.KernelConfig{RuntimeID: "self", Store: store}
	manager.SubscribeToDeploymentPropagation(cfg)
	if len(unsubscribed) != 0 {
		t.Fatalf("initial subscription unsubscribed early: %#v", unsubscribed)
	}
	if got := manager.DebugSnapshot().PropagationSubscriptions; got != 2 {
		t.Fatalf("propagation subscriptions = %d, want 2", got)
	}

	manager.SubscribeToDeploymentPropagation(cfg)
	if len(unsubscribed) != 2 {
		t.Fatalf("replace unsubscribed %d handlers, want 2: %#v", len(unsubscribed), unsubscribed)
	}
	if got := manager.DebugSnapshot().PropagationSubscriptions; got != 2 {
		t.Fatalf("propagation subscriptions after replace = %d, want 2", got)
	}

	if err := manager.Close(); err != nil {
		t.Fatalf("close manager: %v", err)
	}
	if got := manager.DebugSnapshot().PropagationSubscriptions; got != 0 {
		t.Fatalf("propagation subscriptions after close = %d, want 0", got)
	}
	if len(unsubscribed) != 4 {
		t.Fatalf("close unsubscribed %d handlers total, want 4: %#v", len(unsubscribed), unsubscribed)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("close manager twice: %v", err)
	}
	if len(unsubscribed) != 4 {
		t.Fatalf("second close changed unsubscribe count: %#v", unsubscribed)
	}
}

func TestDeploymentPropagationCloseRetainsTimedOutSubscriptionForRetry(t *testing.T) {
	store := &fakeStore{}
	host := &fakeHost{hasRuntime: true}
	release := make(chan struct{})
	closeAttempts := 0
	manager := New(host, nil, func(_ context.Context, _ string, _ func(sdk.Message)) (PropagationHandle, error) {
		return fakePropagationHandle{
			close: func(ctx context.Context) error {
				closeAttempts++
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}, nil
	})

	manager.SubscribeToDeploymentPropagation(types.KernelConfig{RuntimeID: "self", Store: store})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := manager.CloseContext(ctx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("close err = %v, want deadline exceeded", err)
	}
	if got := manager.DebugSnapshot().PropagationSubscriptions; got != 2 {
		t.Fatalf("timed-out subscriptions must be retained for retry, got %d", got)
	}

	close(release)
	if err := manager.CloseContext(context.Background()); err != nil {
		t.Fatalf("retry close manager: %v", err)
	}
	if got := manager.DebugSnapshot().PropagationSubscriptions; got != 0 {
		t.Fatalf("subscriptions after retry close = %d, want 0", got)
	}
	if closeAttempts < 3 {
		t.Fatalf("close attempts = %d, want retry attempts", closeAttempts)
	}
}

func TestDeploymentPropagationHandlerUsesCancelableSubscriptionContext(t *testing.T) {
	store := &fakeStore{deploymentBySource: map[string]types.PersistedDeployment{
		"remote.ts": {Source: "remote.ts", Code: "code"},
	}}
	deployStarted := make(chan struct{})
	deployDone := make(chan error, 1)
	host := &fakeHost{
		hasRuntime: true,
		deployHook: func(ctx context.Context) error {
			close(deployStarted)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	handlers := map[string]func(sdk.Message){}
	manager := New(host, nil, func(_ context.Context, topic string, handler func(sdk.Message)) (PropagationHandle, error) {
		handlers[topic] = handler
		if topic != systemmsg.TopicKitDeployed {
			return fakePropagationHandle{}, nil
		}
		return fakePropagationHandle{
			close: func(ctx context.Context) error {
				select {
				case err := <-deployDone:
					return err
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}, nil
	})

	manager.SubscribeToDeploymentPropagation(types.KernelConfig{RuntimeID: "self", Store: store})
	deployPayload, _ := json.Marshal(systemmsg.KitDeployedEvent{Source: "remote.ts", RuntimeID: "other"})
	go func() {
		handlers[systemmsg.TopicKitDeployed](sdk.Message{Payload: deployPayload})
		deployDone <- nil
	}()
	select {
	case <-deployStarted:
	case <-time.After(time.Second):
		t.Fatal("deploy propagation did not start")
	}

	if err := manager.CloseContext(context.Background()); err != nil {
		t.Fatalf("close manager: %v", err)
	}
	if len(host.deploys) != 1 {
		t.Fatalf("deploy count = %d, want 1", len(host.deploys))
	}
}

type fakeHost struct {
	hasRuntime bool
	seed       int32
	deploys    []fakeDeploy
	teardowns  []string
	deployHook func(context.Context) error
}

type fakeDeploy struct {
	source string
	code   string
	cfg    types.DeployConfig
}

func (h *fakeHost) HasJSRuntime() bool { return h.hasRuntime }

func (h *fakeHost) SetDeployOrderSeed(seed int32) { h.seed = seed }

func (h *fakeHost) Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	var cfg types.DeployConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	h.deploys = append(h.deploys, fakeDeploy{source: source, code: code, cfg: cfg})
	if h.deployHook != nil {
		return nil, h.deployHook(ctx)
	}
	return nil, nil
}

func (h *fakeHost) Teardown(_ context.Context, source string) (int, error) {
	h.teardowns = append(h.teardowns, source)
	return 1, nil
}

func (h *fakeHost) ListDeployments() []runtimecap.DeploymentInfo { return nil }

type fakePropagationHandle struct {
	close func(context.Context) error
}

func (h fakePropagationHandle) Close(ctx context.Context) error {
	if h.close == nil {
		return nil
	}
	return h.close(ctx)
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
