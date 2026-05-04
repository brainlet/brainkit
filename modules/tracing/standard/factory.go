// Package standard registers the tracing module factory used by
// server/standard YAML assembly.
package standard

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/tracing"
	"github.com/brainlet/brainkit/modules/tracing/tracingmsg"

	_ "modernc.org/sqlite"
)

// YAML is the config shape decoded by the registry factory. Empty Path falls
// back to `<FSRoot>/tracing.db`. Zero Retention disables cleanup.
type YAML struct {
	Path      string        `yaml:"path"`
	Retention time.Duration `yaml:"retention"`
}

// Factory is the registered ModuleFactory for tracing.
type Factory struct{}

// Build opens the SQLite-backed trace store and returns the module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	path := y.Path
	if path == "" {
		path = filepath.Join(ctx.FSRoot, "tracing.db")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("tracing: open db %q: %w", path, err)
	}
	var opts []tracing.SQLiteTraceStoreOption
	if y.Retention > 0 {
		opts = append(opts, tracing.WithRetention(y.Retention))
	}
	store, err := tracing.NewSQLiteTraceStore(db, opts...)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("tracing: init store %q: %w", path, err)
	}
	return tracing.New(tracing.Config{Store: store}), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "tracing",
		Status:  bkmodule.StatusBeta,
		Summary: "Persistent span store with trace.get / trace.list.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[tracingmsg.TraceGetMsg, tracingmsg.TraceGetResp](),
			bkmodule.CommandMessage[tracingmsg.TraceListMsg, tracingmsg.TraceListResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
			bkmodule.RequiredCapabilityOf[bkmodule.LeaseFunc[tracing.TraceStore]](bkmodule.CapabilityTraceStoreLease),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindStore, "tracing.store", "Attached persistent trace store."),
		},
	}
}

func init() { bkmodule.Register("tracing", Factory{}) }
