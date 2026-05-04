package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
)

func TestWorkflowModuleRestartsActiveWorkflowsWhenStoreCapabilityPresent(t *testing.T) {
	host := newWorkflowTestHost(t)
	var calls []string
	host.provide(t, bkmodule.CapabilityCallJS, func(_ context.Context, fn string, _ any) (json.RawMessage, error) {
		calls = append(calls, fn)
		return json.RawMessage(`{"restarted":1,"errors":[{"workflow":"daily","error":"boom"}]}`), nil
	})
	host.provide(t, bkmodule.CapabilityKitStore, &workflowFakeStore{})

	var reported []types.ErrorContext
	host.provide(t, bkmodule.CapabilityReportError, func(_ error, ctx types.ErrorContext) {
		reported = append(reported, ctx)
	})

	if err := New().Mount(context.Background(), host); err != nil {
		t.Fatalf("mount workflow: %v", err)
	}
	if len(host.commands) != 8 {
		t.Fatalf("mounted command count = %d, want 8", len(host.commands))
	}
	if len(calls) != 1 || calls[0] != "__brainkit.workflow.restartActive" {
		t.Fatalf("restart calls = %#v", calls)
	}
	if len(reported) != 1 {
		t.Fatalf("reported recovery errors = %#v, want 1", reported)
	}
	if reported[0].Operation != "RestartWorkflow" || reported[0].Component != "workflow" || reported[0].Source != "daily" {
		t.Fatalf("reported context = %#v", reported[0])
	}
}

func TestWorkflowModuleSkipsActiveRestartWithoutStoreCapability(t *testing.T) {
	host := newWorkflowTestHost(t)
	var calls int
	host.provide(t, bkmodule.CapabilityCallJS, func(_ context.Context, _ string, _ any) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"restarted":0}`), nil
	})

	if err := New().Mount(context.Background(), host); err != nil {
		t.Fatalf("mount workflow: %v", err)
	}
	if calls != 0 {
		t.Fatalf("restart calls = %d, want 0 without %s", calls, bkmodule.CapabilityKitStore)
	}
}

func TestWorkflowModuleReportsActiveRestartFailure(t *testing.T) {
	host := newWorkflowTestHost(t)
	host.provide(t, bkmodule.CapabilityCallJS, func(context.Context, string, any) (json.RawMessage, error) {
		return nil, errors.New("restart failed")
	})
	host.provide(t, bkmodule.CapabilityKitStore, &workflowFakeStore{})

	var gotErr error
	var gotCtx types.ErrorContext
	host.provide(t, bkmodule.CapabilityReportError, func(err error, ctx types.ErrorContext) {
		gotErr = err
		gotCtx = ctx
	})

	if err := New().Mount(context.Background(), host); err != nil {
		t.Fatalf("mount workflow: %v", err)
	}
	if gotErr == nil {
		t.Fatalf("expected recovery error to be reported")
	}
	if gotCtx.Operation != "RestartActiveWorkflows" || gotCtx.Component != "workflow" {
		t.Fatalf("reported context = %#v", gotCtx)
	}
}

type workflowTestHost struct {
	t        *testing.T
	scope    bkmodule.Scope
	caps     *bkmodule.CapabilityRegistry
	commands []bkmodule.CommandSpec
	logger   *slog.Logger
}

func newWorkflowTestHost(t *testing.T) *workflowTestHost {
	t.Helper()
	return &workflowTestHost{
		t:      t,
		scope:  bkmodule.NewScope("workflow-test"),
		caps:   bkmodule.NewCapabilityRegistry(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (h *workflowTestHost) provide(t *testing.T, name string, value any) {
	t.Helper()
	if _, err := h.caps.Provide(context.Background(), name, value); err != nil {
		t.Fatalf("provide %s: %v", name, err)
	}
}

func (h *workflowTestHost) Scope() bkmodule.Scope                 { return h.scope }
func (h *workflowTestHost) Messages() bkmodule.MessageHost        { return workflowNoopMessages{} }
func (h *workflowTestHost) Commands() bkmodule.CommandHost        { return (*workflowTestCommands)(h) }
func (h *workflowTestHost) Tools() bkmodule.ToolHost              { return workflowNoopTools{} }
func (h *workflowTestHost) Capabilities() bkmodule.CapabilityHost { return h.caps }
func (h *workflowTestHost) Logger() *slog.Logger                  { return h.logger }

type workflowTestCommands workflowTestHost

func (h *workflowTestCommands) Handle(spec bkmodule.CommandSpec) (bkmodule.Handle, error) {
	host := (*workflowTestHost)(h)
	host.commands = append(host.commands, spec)
	return bkmodule.HandleFunc(func(context.Context) error { return nil }), nil
}

func (h *workflowTestCommands) Has(topic string) bool {
	host := (*workflowTestHost)(h)
	for _, cmd := range host.commands {
		if cmd.Topic == topic {
			return true
		}
	}
	return false
}

type workflowNoopMessages struct{}

func (workflowNoopMessages) PublishRaw(context.Context, string, json.RawMessage) (string, error) {
	return "", nil
}
func (workflowNoopMessages) SubscribeRaw(context.Context, string, func(sdk.Message)) (bkmodule.Handle, error) {
	return bkmodule.HandleFunc(func(context.Context) error { return nil }), nil
}
func (workflowNoopMessages) ReplyRaw(context.Context, string, string, json.RawMessage, bool) error {
	return nil
}

type workflowNoopTools struct{}

func (workflowNoopTools) Register(context.Context, bkmodule.ToolSpec) (bkmodule.Handle, error) {
	return bkmodule.HandleFunc(func(context.Context) error { return nil }), nil
}

type workflowFakeStore struct{}

func (s *workflowFakeStore) SaveDeployment(types.PersistedDeployment) error { return nil }
func (s *workflowFakeStore) LoadDeployments() ([]types.PersistedDeployment, error) {
	return nil, nil
}
func (s *workflowFakeStore) LoadDeployment(string) (types.PersistedDeployment, error) {
	return types.PersistedDeployment{}, nil
}
func (s *workflowFakeStore) DeleteDeployment(string) error { return nil }
func (s *workflowFakeStore) SaveSchedule(types.PersistedSchedule) error {
	return nil
}
func (s *workflowFakeStore) LoadSchedules() ([]types.PersistedSchedule, error) { return nil, nil }
func (s *workflowFakeStore) DeleteSchedule(string) error                       { return nil }
func (s *workflowFakeStore) ClaimScheduleFire(string, time.Time) (bool, error) { return true, nil }
func (s *workflowFakeStore) SaveInstalledPlugin(types.InstalledPlugin) error   { return nil }
func (s *workflowFakeStore) LoadInstalledPlugins() ([]types.InstalledPlugin, error) {
	return nil, nil
}
func (s *workflowFakeStore) DeleteInstalledPlugin(string) error { return nil }
func (s *workflowFakeStore) SaveRunningPlugin(types.RunningPluginRecord) error {
	return nil
}
func (s *workflowFakeStore) LoadRunningPlugins() ([]types.RunningPluginRecord, error) {
	return nil, nil
}
func (s *workflowFakeStore) DeleteRunningPlugin(string) error { return nil }
func (s *workflowFakeStore) Close() error                     { return nil }
