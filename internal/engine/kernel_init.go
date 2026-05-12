package engine

import (
	"context"
	"fmt"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulehost/providerhost"
	"github.com/brainlet/brainkit/modulehost/storagehost"
	"github.com/brainlet/brainkit/modulehost/transporthost"
)

func (k *Kernel) initProviders(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	host, err := providerhost.NewManager(cfg, providerhost.Hooks{
		HasJSRuntime:    k.HasJSRuntime,
		EvalJS:          k.EvalJS,
		CallJS:          k.CallJS,
		ShutdownContext: k.shutdownCtx,
	})
	if err != nil {
		return err
	}
	k.providerHost = host
	k.storageHost = storagehost.NewManager(host.Registry(), k.HasJSRuntime)

	// Register all storages and vectors in the provider registry
	if err := k.registerStorages(cfg, bridgeURLs); err != nil {
		return fmt.Errorf("brainkit: register storages: %w", err)
	}
	if err := k.registerVectors(cfg, bridgeURLs); err != nil {
		return fmt.Errorf("brainkit: register vectors: %w", err)
	}
	return nil
}

func (k *Kernel) initTransport(cfg types.KernelConfig) error {
	host, err := transporthost.New(cfg, k.logger)
	if err != nil {
		return err
	}
	k.transportHost = host

	if !cfg.DeferRouterStart {
		// Legacy standalone path — register + start immediately. brainkit.New
		// sets DeferRouterStart=true so the router starts via Kernel.StartRouter
		// after Kit-scoped modules have registered their commands.
		k.transportHost.RegisterCommands(commandBindingsForKernel(k))
		if err := k.transportHost.Start(context.Background()); err != nil {
			return err
		}
	}

	return nil
}

// StartRouter finalizes kernel-only command bindings and starts the message
// router. Safe to call once after NewKernel(DeferRouterStart=true); no-op if
// the router has already been started.
func (k *Kernel) StartRouter(ctx context.Context) error {
	if k.transportHost == nil {
		return fmt.Errorf("brainkit: router not initialized")
	}
	if k.transportHost.IsRunning() {
		return nil
	}
	k.transportHost.RegisterCommands(commandBindingsForKernel(k))
	return k.transportHost.Start(ctx)
}

func (k *Kernel) initPersistence(cfg types.KernelConfig) {
	if k.runtimeHost != nil {
		k.runtimeHost.InitPersistence(cfg)
	}
}

func (k *Kernel) initAudit(cfg types.KernelConfig) func() {
	// Always create the Recorder — it's nil-safe without a store (Record
	// calls no-op until the audit module attaches one via a scoped lease). The
	// audit module owns the store wiring; the Recorder stays in core so
	// every subsystem can record unconditionally.
	k.audit = auditpkg.NewRecorderWithConfig(auditpkg.RecorderConfig{
		RuntimeID: cfg.RuntimeID, Namespace: cfg.Namespace,
	})
	return nil
}
