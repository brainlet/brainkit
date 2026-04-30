// Package reference owns the kit.reference bus commands as a hot-mountable
// Kit module.
package reference

import (
	"context"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/reference/referencemsg"
)

// Module exposes kit.reference and kit.reference.list. Construct via New and
// include in brainkit.Config.Modules when the runtime should expose the
// embedded reference corpus over the bus.
type Module struct {
	catalog bkmodule.ReferenceCatalog
}

// New creates the reference module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "reference" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers reference command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	catalog, err := bkmodule.RequireCapability[bkmodule.ReferenceCatalog](host, bkmodule.CapabilityReferenceCatalog)
	if err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	m.catalog = catalog
	host.Scope().Defer(func(context.Context) error {
		m.catalog = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.Get),
		bkmodule.Command(m.List),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close detaches the module from Kit capabilities.
func (m *Module) Close() error {
	m.catalog = nil
	return nil
}

// Factory is the registered ModuleFactory for reference.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the reference module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "reference",
		Status:  bkmodule.StatusStable,
		Summary: "Embedded reference corpus commands (kit.reference, list).",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[referencemsg.KitReferenceListMsg, referencemsg.KitReferenceListResp](),
			bkmodule.CommandMessage[referencemsg.KitReferenceMsg, referencemsg.KitReferenceResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[bkmodule.ReferenceCatalog](bkmodule.CapabilityReferenceCatalog),
		},
	}
}

func init() { bkmodule.Register("reference", Factory{}) }

// Get handles kit.reference.
func (m *Module) Get(_ context.Context, req referencemsg.KitReferenceMsg) (*referencemsg.KitReferenceResp, error) {
	content, err := m.catalog.GetReference(req.Name)
	if err != nil {
		return nil, err
	}
	return &referencemsg.KitReferenceResp{Name: req.Name, Content: content}, nil
}

// List handles kit.reference.list.
func (m *Module) List(context.Context, referencemsg.KitReferenceListMsg) (*referencemsg.KitReferenceListResp, error) {
	refs := m.catalog.ListReferences()
	out := make([]referencemsg.KitReferenceListEntry, 0, len(refs))
	for _, ref := range refs {
		out = append(out, referencemsg.KitReferenceListEntry{
			Name:        ref.Name,
			Kind:        ref.Kind,
			Description: ref.Description,
			Size:        ref.Size,
			Parts:       append([]string(nil), ref.Parts...),
		})
	}
	return &referencemsg.KitReferenceListResp{References: out}, nil
}
