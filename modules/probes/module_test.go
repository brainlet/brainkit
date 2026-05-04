package probes

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/stretchr/testify/require"
)

func TestCloseContextCancelsActiveProbeAndWaits(t *testing.T) {
	runner := newContextProbeRunner()
	module := New(Config{})
	module.start(runner)

	require.Eventually(t, func() bool {
		return runner.started()
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return module.DebugSnapshot().ActiveSweeps == 1
	}, time.Second, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, module.CloseContext(ctx))
	require.True(t, runner.done())
	require.True(t, errors.Is(runner.err(), context.Canceled))

	snapshot := module.DebugSnapshot()
	require.True(t, snapshot.Closed)
	require.False(t, snapshot.RunnerAttached)
	require.False(t, snapshot.LoopRunning)
	require.Equal(t, int64(0), snapshot.ActiveSweeps)
	require.NotNil(t, snapshot.LastSweepStarted)
	require.NotNil(t, snapshot.LastSweepFinished)
}

func TestCloseContextReturnsDeadlineWhenProbeDoesNotExit(t *testing.T) {
	runner := newGatedProbeRunner()
	module := New(Config{})
	module.start(runner)

	require.Eventually(t, func() bool {
		return runner.started()
	}, time.Second, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	closeErr := make(chan error, 1)
	go func() {
		closeErr <- module.CloseContext(ctx)
	}()
	require.Eventually(t, func() bool {
		return module.DebugSnapshot().Closing
	}, time.Second, 10*time.Millisecond)
	require.ErrorIs(t, <-closeErr, context.DeadlineExceeded)

	snapshot := module.DebugSnapshot()
	require.False(t, snapshot.Closed)
	require.False(t, snapshot.Closing)
	require.True(t, snapshot.RunnerAttached)
	require.Equal(t, int64(1), snapshot.ActiveSweeps)

	runner.release()
	require.NoError(t, module.CloseContext(context.Background()))
	require.Eventually(t, func() bool {
		return module.DebugSnapshot().ActiveSweeps == 0
	}, time.Second, 10*time.Millisecond)
	snapshot = module.DebugSnapshot()
	require.True(t, snapshot.Closed)
	require.False(t, snapshot.RunnerAttached)
	require.False(t, snapshot.LoopRunning)
}

func TestFactoryDescriptorRequiresTypedProbeRunner(t *testing.T) {
	desc := bkmodule.NormalizeDescriptor("probes", Factory{}.Describe())
	var foundProbe, foundLifecycle bool
	for _, cap := range desc.Capabilities {
		switch cap.Name {
		case bkmodule.CapabilityProbeAll:
			foundProbe = true
			require.Equal(t, bkmodule.CapabilityRequired, cap.Direction)
			require.Contains(t, cap.Type, "ProbeRunner")
		case bkmodule.CapabilityLifecycleDebugRegistry:
			foundLifecycle = true
			require.Equal(t, bkmodule.CapabilityOptional, cap.Direction)
		}
	}
	require.True(t, foundProbe)
	require.True(t, foundLifecycle)
}

func TestFactoryYAMLCanDisableInitialProbe(t *testing.T) {
	disabled := false
	built, err := (Factory{}).Build(bkmodule.BuildContext{
		Decode: func(v any) error {
			y := v.(*YAML)
			y.ProbeOnRegister = &disabled
			return nil
		},
	})
	require.NoError(t, err)
	module := built.(*Module)
	require.False(t, module.probeOnRegister())
	require.True(t, New(Config{}).probeOnRegister())
}

type contextProbeRunner struct {
	calls     atomic.Int64
	startedCh chan struct{}
	doneCh    chan struct{}
	startOnce sync.Once
	doneOnce  sync.Once
	errMu     sync.Mutex
	ctxErr    error
}

func newContextProbeRunner() *contextProbeRunner {
	return &contextProbeRunner{
		startedCh: make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

func (r *contextProbeRunner) ProbeAll(ctx context.Context) {
	r.calls.Add(1)
	r.startOnce.Do(func() { close(r.startedCh) })
	<-ctx.Done()
	r.errMu.Lock()
	r.ctxErr = ctx.Err()
	r.errMu.Unlock()
	r.doneOnce.Do(func() { close(r.doneCh) })
}

func (r *contextProbeRunner) started() bool {
	select {
	case <-r.startedCh:
		return true
	default:
		return false
	}
}

func (r *contextProbeRunner) done() bool {
	select {
	case <-r.doneCh:
		return true
	default:
		return false
	}
}

func (r *contextProbeRunner) err() error {
	r.errMu.Lock()
	defer r.errMu.Unlock()
	return r.ctxErr
}

type gatedProbeRunner struct {
	startedCh chan struct{}
	releaseCh chan struct{}
	startOnce sync.Once
	releaseMu sync.Mutex
	released  bool
}

func newGatedProbeRunner() *gatedProbeRunner {
	return &gatedProbeRunner{
		startedCh: make(chan struct{}),
		releaseCh: make(chan struct{}),
	}
}

func (r *gatedProbeRunner) ProbeAll(context.Context) {
	r.startOnce.Do(func() { close(r.startedCh) })
	<-r.releaseCh
}

func (r *gatedProbeRunner) started() bool {
	select {
	case <-r.startedCh:
		return true
	default:
		return false
	}
}

func (r *gatedProbeRunner) release() {
	r.releaseMu.Lock()
	defer r.releaseMu.Unlock()
	if r.released {
		return
	}
	r.released = true
	close(r.releaseCh)
}
