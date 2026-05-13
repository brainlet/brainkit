// Package standard registers the audit module factory used by standard
// observability/full YAML assembly.
package standard

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/audit"
	"github.com/brainlet/brainkit/modules/audit/auditmsg"
	auditsqlite "github.com/brainlet/brainkit/modules/audit/stores/sqlite"
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

// StoreOpener opens an audit store from YAML. Extra backends can be registered
// by side-effect packages without making the standard SQLite path link them.
type StoreOpener func(bkmodule.BuildContext, YAML) (audit.Store, error)

var (
	storeOpenerMu sync.RWMutex
	storeOpeners  = map[string]StoreOpener{}
)

// RegisterStore registers an audit standard YAML backend. It is intended for
// optional backend packages such as modules/audit/standard/postgres.
func RegisterStore(typ string, opener StoreOpener) {
	if typ == "" {
		panic("audit/standard: store type is required")
	}
	if opener == nil {
		panic(fmt.Sprintf("audit/standard: RegisterStore(%q): opener is nil", typ))
	}
	storeOpenerMu.Lock()
	defer storeOpenerMu.Unlock()
	if _, exists := storeOpeners[typ]; exists {
		panic(fmt.Sprintf("audit/standard: RegisterStore(%q): already registered", typ))
	}
	storeOpeners[typ] = opener
}

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
		s, err := auditsqlite.New(path)
		if err != nil {
			return nil, fmt.Errorf("audit: open sqlite %q: %w", path, err)
		}
		return s, nil
	default:
		storeOpenerMu.RLock()
		opener := storeOpeners[y.Type]
		storeOpenerMu.RUnlock()
		if opener == nil {
			return nil, fmt.Errorf("audit: unknown store type %q (registered: sqlite%s)", y.Type, registeredStoreTypesSuffix())
		}
		return opener(ctx, y)
	}
}

func registeredStoreTypesSuffix() string {
	storeOpenerMu.RLock()
	defer storeOpenerMu.RUnlock()
	if len(storeOpeners) == 0 {
		return "; import github.com/brainlet/brainkit/modules/audit/standard/postgres to enable postgres"
	}
	types := make([]string, 0, len(storeOpeners))
	for typ := range storeOpeners {
		types = append(types, typ)
	}
	sort.Strings(types)
	out := ""
	for _, typ := range types {
		out += ", " + typ
	}
	return out
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
