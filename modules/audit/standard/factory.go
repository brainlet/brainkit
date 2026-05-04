// Package standard registers the audit module factory used by
// server/standard YAML assembly.
package standard

import (
	"fmt"
	"path/filepath"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/audit"
	"github.com/brainlet/brainkit/modules/audit/auditmsg"
	"github.com/brainlet/brainkit/modules/audit/stores"
)

// YAML is the config shape decoded by the registry factory. Empty Path falls
// back to `<FSRoot>/audit.db`. Other backends can be selected via Type.
type YAML struct {
	Type             string `yaml:"type"`
	Path             string `yaml:"path"`
	ConnectionString string `yaml:"connection_string"`
	Verbose          bool   `yaml:"verbose"`
}

// Factory is the registered ModuleFactory for audit.
type Factory struct{}

// Build opens the audit store and returns the module. OwnStore is always true:
// the factory opened it, so the factory's module closes it.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	store, err := openAuditStore(ctx, y)
	if err != nil {
		return nil, err
	}
	return audit.NewModule(audit.Config{Store: store, Verbose: y.Verbose, OwnStore: true}), nil
}

func openAuditStore(ctx bkmodule.BuildContext, y YAML) (audit.Store, error) {
	switch y.Type {
	case "", "sqlite":
		path := y.Path
		if path == "" {
			path = filepath.Join(ctx.FSRoot, "audit.db")
		}
		s, err := stores.NewSQLite(path)
		if err != nil {
			return nil, fmt.Errorf("audit: open sqlite %q: %w", path, err)
		}
		return s, nil
	case "postgres":
		s, err := stores.NewPostgres(y.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("audit: open postgres: %w", err)
		}
		return s, nil
	default:
		return nil, fmt.Errorf("audit: unknown store type %q (want sqlite or postgres)", y.Type)
	}
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "audit",
		Status:  bkmodule.StatusStable,
		Summary: "Persistent audit log with query/stats/prune bus commands.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[auditmsg.AuditPruneMsg, auditmsg.AuditPruneResp](),
			bkmodule.CommandMessage[auditmsg.AuditQueryMsg, auditmsg.AuditQueryResp](),
			bkmodule.CommandMessage[auditmsg.AuditStatsMsg, auditmsg.AuditStatsResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[bkmodule.LeaseFunc[audit.Verbosity]](bkmodule.CapabilityAuditVerbosityLease),
			bkmodule.RequiredCapabilityOf[bkmodule.LeaseFunc[audit.Store]](bkmodule.CapabilityAuditStoreLease),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindStore, "audit.store", "Attached persistent audit event store."),
		},
	}
}

func init() { bkmodule.Register("audit", Factory{}) }
