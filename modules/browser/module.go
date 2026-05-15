// Package browser owns browser/CDP lifecycle for concrete Mastra browser providers.
package browser

import (
	"context"
	"fmt"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	browsercap "github.com/brainlet/brainkit/modulecap/browser"
	"github.com/brainlet/brainkit/modules/browser/browsermsg"
)

// Module exposes browser.session.* commands and a browser manager capability.
type Module struct {
	cfg     Config
	manager *Manager
}

// NewModule builds the browser lifecycle module.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

// New is a shorthand for NewModule.
func New(cfg Config) *Module { return NewModule(cfg) }

func (m *Module) ID() string              { return "browser" }
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusWIP }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	manager := NewManager(m.cfg)
	m.manager = manager
	host.Scope().Defer(func(closeCtx context.Context) error {
		err := manager.CloseContext(closeCtx)
		if err == nil && m.manager == manager {
			m.manager = nil
		}
		return err
	})
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindProcess, "browser.manager", "Browser/CDP process lifecycle owner."))
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindRuntime, "browser.sessions", "Tracked browser/CDP sessions."))
	if _, err := host.Capabilities().Provide(ctx, bkmodule.CapabilityBrowserManager, browsercap.Manager(manager)); err != nil {
		return fmt.Errorf("browser: %w", err)
	}
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "browser", func() any {
			return manager.DebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("browser: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.Launch),
		bkmodule.Command(m.CloseSession),
		bkmodule.Command(m.List),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

func (m *Module) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.CloseContext(ctx)
}

func (m *Module) CloseContext(ctx context.Context) error {
	if m.manager == nil {
		return nil
	}
	err := m.manager.CloseContext(ctx)
	if err == nil {
		m.manager = nil
	}
	return err
}

func (m *Module) Launch(ctx context.Context, req browsermsg.BrowserSessionLaunchMsg) (*browsermsg.BrowserSessionLaunchResp, error) {
	if m.manager == nil {
		return nil, fmt.Errorf("browser: module is not mounted")
	}
	session, err := m.manager.Launch(ctx, browsercap.LaunchRequest{
		Provider:       req.Provider,
		ThreadID:       req.ThreadID,
		Scope:          req.Scope,
		ExecutablePath: req.ExecutablePath,
		ProfileDir:     req.ProfileDir,
		Headless:       req.Headless,
		Args:           req.Args,
	})
	if err != nil {
		return nil, err
	}
	return &browsermsg.BrowserSessionLaunchResp{Session: session}, nil
}

func (m *Module) CloseSession(ctx context.Context, req browsermsg.BrowserSessionCloseMsg) (*browsermsg.BrowserSessionCloseResp, error) {
	if m.manager == nil {
		return nil, fmt.Errorf("browser: module is not mounted")
	}
	if err := m.manager.Close(ctx, req.ID); err != nil {
		return nil, err
	}
	return &browsermsg.BrowserSessionCloseResp{Closed: true}, nil
}

func (m *Module) List(context.Context, browsermsg.BrowserSessionListMsg) (*browsermsg.BrowserSessionListResp, error) {
	if m.manager == nil {
		return nil, fmt.Errorf("browser: module is not mounted")
	}
	return &browsermsg.BrowserSessionListResp{Sessions: m.manager.List()}, nil
}

// Factory is the registered ModuleFactory for browser.
type Factory struct{}

// YAML is the light config shape for browser lifecycle ownership.
type YAML struct {
	ExecutablePath string        `yaml:"executable_path"`
	ProfileRoot    string        `yaml:"profile_root"`
	Headless       bool          `yaml:"headless"`
	LaunchTimeout  time.Duration `yaml:"launch_timeout"`
	ExtraArgs      []string      `yaml:"extra_args"`
}

func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return NewModule(Config{
		ExecutablePath: y.ExecutablePath,
		ProfileRoot:    y.ProfileRoot,
		Headless:       y.Headless,
		LaunchTimeout:  y.LaunchTimeout,
		ExtraArgs:      y.ExtraArgs,
	}), nil
}

func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "browser",
		Status:  bkmodule.StatusWIP,
		Summary: "Browser/CDP lifecycle owner for concrete Mastra browser providers.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[browsermsg.BrowserSessionLaunchMsg, browsermsg.BrowserSessionLaunchResp](),
			bkmodule.CommandMessage[browsermsg.BrowserSessionCloseMsg, browsermsg.BrowserSessionCloseResp](),
			bkmodule.CommandMessage[browsermsg.BrowserSessionListMsg, browsermsg.BrowserSessionListResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.ProvidedCapabilityOf[browsercap.Manager](bkmodule.CapabilityBrowserManager),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindProcess, "browser.manager", "Browser/CDP process lifecycle owner."),
			bkmodule.Resource(bkmodule.ResourceKindRuntime, "browser.sessions", "Tracked browser/CDP sessions."),
		},
	}
}

func init() { bkmodule.Register("browser", Factory{}) }
