package gateway

import (
	"context"

	bkmodule "github.com/brainlet/brainkit/module"
)

// ID reports the hot-mount module identifier.
func (gw *Gateway) ID() string { return "gateway" }

// Status reports maturity (stable).
func (gw *Gateway) Status() bkmodule.Status { return bkmodule.StatusStable }

func (gw *Gateway) Mount(_ context.Context, host bkmodule.Host) error {
	gw.SetRuntime(host.Runtime())
	if err := gw.Start(); err != nil {
		return err
	}
	host.Scope().Defer(func(context.Context) error { return gw.Close() })
	return nil
}

// Close stops the HTTP server and unsubscribes bus route commands.
func (gw *Gateway) Close() error {
	return gw.Stop()
}
