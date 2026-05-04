// Package standard registers the schedules module factory used by
// server/standard YAML assembly.
package standard

import (
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/schedules"
	storesqlite "github.com/brainlet/brainkit/stores/sqlite"
)

// YAML is the config shape decoded by the registry factory.
//
// `path`, when set, opens a dedicated SQLite database at that path. When empty,
// schedules share the Kit's main store when one is available.
type YAML struct {
	Path string `yaml:"path"`
}

// Factory is the registered ModuleFactory for schedules in standard assembly.
type Factory struct{}

// Build opens the dedicated store when Path is set, otherwise leaves Store nil
// so Mount falls back to the shared KitStore capability.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := schedules.Config{}
	if y.Path != "" {
		store, err := storesqlite.New(y.Path)
		if err != nil {
			return nil, fmt.Errorf("schedules: open store %q: %w", y.Path, err)
		}
		cfg.Store = store
		cfg.OwnStore = true
	}
	return schedules.NewModule(cfg), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return schedules.Factory{}.Describe()
}

func init() { bkmodule.Register("schedules", Factory{}) }
