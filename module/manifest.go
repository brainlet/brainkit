package module

import (
	"reflect"
	"sort"

	"github.com/brainlet/brainkit/sdk"
)

// MessageKind identifies how a module uses a bus topic.
type MessageKind string

const (
	MessageKindCommand      MessageKind = "command"
	MessageKindEvent        MessageKind = "event"
	MessageKindSubscription MessageKind = "subscription"
)

// MessageDescriptor describes one bus topic owned or consumed by a module.
type MessageDescriptor struct {
	Topic    string      `json:"topic"`
	Kind     MessageKind `json:"kind,omitempty"`
	Request  string      `json:"request,omitempty"`
	Response string      `json:"response,omitempty"`
	Summary  string      `json:"summary,omitempty"`
}

// CapabilityDirection identifies whether a module provides or requires a
// capability through module.Host.Capabilities.
type CapabilityDirection string

const (
	CapabilityProvided CapabilityDirection = "provided"
	CapabilityRequired CapabilityDirection = "required"
	CapabilityOptional CapabilityDirection = "optional"
)

// CapabilityDescriptor describes one named module host capability.
type CapabilityDescriptor struct {
	Name      string              `json:"name"`
	Direction CapabilityDirection `json:"direction,omitempty"`
	Type      string              `json:"type,omitempty"`
	Summary   string              `json:"summary,omitempty"`
}

// CapabilityGroups is the manifest view optimized for humans and operators.
// The flat Capabilities list remains the source of truth; NormalizeDescriptor
// derives this grouped view so JSON consumers do not need to re-bucket it.
type CapabilityGroups struct {
	Required []CapabilityDescriptor `json:"required,omitempty"`
	Optional []CapabilityDescriptor `json:"optional,omitempty"`
	Provided []CapabilityDescriptor `json:"provided,omitempty"`
}

// ResourceKind identifies a non-bus resource owned by a module scope.
type ResourceKind string

const (
	ResourceKindTool       ResourceKind = "tool"
	ResourceKindCapability ResourceKind = "capability"
	ResourceKindHook       ResourceKind = "hook"
	ResourceKindStore      ResourceKind = "store"
	ResourceKindProcess    ResourceKind = "process"
	ResourceKindHTTP       ResourceKind = "http"
	ResourceKindRuntime    ResourceKind = "runtime"
	ResourceKindScheduler  ResourceKind = "scheduler"
	ResourceKindCustom     ResourceKind = "custom"
)

// ResourceDescriptor describes a non-bus resource owned by a module. Resources
// are intentionally generic so modules can manifest stores, hooks, processes,
// HTTP listeners, schedulers, and domain-specific handles without new host APIs.
type ResourceDescriptor struct {
	Kind     ResourceKind      `json:"kind"`
	Name     string            `json:"name"`
	Summary  string            `json:"summary,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Resource describes a generic module-owned resource.
func Resource(kind ResourceKind, name string, summary ...string) ResourceDescriptor {
	return ResourceDescriptor{
		Kind:    kind,
		Name:    name,
		Summary: first(summary),
	}
}

// ResourceWithMetadata describes a generic resource with stable string
// metadata for CLI/JSON introspection.
func ResourceWithMetadata(kind ResourceKind, name string, metadata map[string]string, summary ...string) ResourceDescriptor {
	return ResourceDescriptor{
		Kind:     kind,
		Name:     name,
		Summary:  first(summary),
		Metadata: cloneStringMap(metadata),
	}
}

// ToolResource describes a tool registered through module.Host.Tools.
func ToolResource(spec ToolSpec) ResourceDescriptor {
	metadata := map[string]string{}
	if spec.ShortName != "" {
		metadata["short_name"] = spec.ShortName
	}
	if spec.Owner != "" {
		metadata["owner"] = spec.Owner
	}
	if spec.Package != "" {
		metadata["package"] = spec.Package
	}
	if spec.Version != "" {
		metadata["version"] = spec.Version
	}
	if spec.Local {
		metadata["local"] = "true"
	}
	return ResourceWithMetadata(ResourceKindTool, spec.Name, metadata, spec.Description)
}

// CommandMessage describes a request/reply bus command.
func CommandMessage[Req sdk.BrainkitMessage, Resp any](summary ...string) MessageDescriptor {
	var req Req
	return MessageDescriptor{
		Topic:    req.BusTopic(),
		Kind:     MessageKindCommand,
		Request:  typeName[Req](),
		Response: typeName[Resp](),
		Summary:  first(summary),
	}
}

// EventMessage describes a fire-and-forget event topic a module emits.
func EventMessage[Evt sdk.BrainkitMessage](summary ...string) MessageDescriptor {
	var evt Evt
	return MessageDescriptor{
		Topic:   evt.BusTopic(),
		Kind:    MessageKindEvent,
		Request: typeName[Evt](),
		Summary: first(summary),
	}
}

// SubscriptionMessage describes a topic a module subscribes to.
func SubscriptionMessage[Msg sdk.BrainkitMessage](summary ...string) MessageDescriptor {
	var msg Msg
	return MessageDescriptor{
		Topic:   msg.BusTopic(),
		Kind:    MessageKindSubscription,
		Request: typeName[Msg](),
		Summary: first(summary),
	}
}

// SubscriptionMessageWithResponse describes a request/reply topic served by a
// raw subscription instead of CommandHost. This covers modules that need custom
// transport behavior but still own a typed bus request surface.
func SubscriptionMessageWithResponse[Msg sdk.BrainkitMessage, Resp any](summary ...string) MessageDescriptor {
	var msg Msg
	return MessageDescriptor{
		Topic:    msg.BusTopic(),
		Kind:     MessageKindSubscription,
		Request:  typeName[Msg](),
		Response: typeName[Resp](),
		Summary:  first(summary),
	}
}

// RequiredCapability describes a capability consumed by a module.
func RequiredCapability(name string, value any, summary ...string) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:      name,
		Direction: CapabilityRequired,
		Type:      valueTypeName(value),
		Summary:   first(summary),
	}
}

// RequiredCapabilityOf describes a consumed capability with a generic type.
func RequiredCapabilityOf[T any](name string, summary ...string) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:      name,
		Direction: CapabilityRequired,
		Type:      typeName[T](),
		Summary:   first(summary),
	}
}

// OptionalCapability describes a capability used when available by a module.
func OptionalCapability(name string, value any, summary ...string) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:      name,
		Direction: CapabilityOptional,
		Type:      valueTypeName(value),
		Summary:   first(summary),
	}
}

// OptionalCapabilityOf describes an optional capability with a generic type.
func OptionalCapabilityOf[T any](name string, summary ...string) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:      name,
		Direction: CapabilityOptional,
		Type:      typeName[T](),
		Summary:   first(summary),
	}
}

// ProvidedCapability describes a capability exported by a module.
func ProvidedCapability(name string, value any, summary ...string) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:      name,
		Direction: CapabilityProvided,
		Type:      valueTypeName(value),
		Summary:   first(summary),
	}
}

// ProvidedCapabilityOf describes an exported capability with a generic type.
func ProvidedCapabilityOf[T any](name string, summary ...string) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:      name,
		Direction: CapabilityProvided,
		Type:      typeName[T](),
		Summary:   first(summary),
	}
}

// DescribeModule returns normalized metadata for a module instance. It prefers
// module-owned metadata, then the registered factory's metadata, and finally
// falls back to status metadata on the module value itself.
func DescribeModule(mod Module) Descriptor {
	if mod == nil {
		return Descriptor{}
	}
	id := mod.ID()
	desc := Descriptor{Name: id}
	if d, ok := mod.(Describer); ok {
		desc = d.Describe()
	} else if f, ok := Lookup(id); ok {
		if d, ok := f.(Describer); ok {
			desc = d.Describe()
		}
	}
	if desc.Name == "" {
		desc.Name = id
	}
	if desc.Status == "" {
		if s, ok := mod.(StatusReporter); ok {
			desc.Status = s.Status()
		}
	}
	return NormalizeDescriptor(id, desc)
}

// DependenciesOf returns module dependencies using descriptor metadata as the
// source of truth.
func DependenciesOf(mod Module) []string {
	desc := DescribeModule(mod)
	out := append([]string(nil), desc.Requires...)
	sort.Strings(out)
	return out
}

// NormalizeDescriptor fills the name when omitted and sorts repeated metadata
// fields for stable CLI and JSON output.
func NormalizeDescriptor(name string, desc Descriptor) Descriptor {
	if desc.Name == "" {
		desc.Name = name
	}
	desc.Provides = uniqueStrings(desc.Provides)
	desc.Requires = uniqueStrings(desc.Requires)
	desc.Commands = normalizeMessages(desc.Commands, MessageKindCommand)
	desc.Events = normalizeMessages(desc.Events, MessageKindEvent)
	desc.Subscriptions = normalizeMessages(desc.Subscriptions, MessageKindSubscription)
	desc.Capabilities = normalizeCapabilities(desc.Capabilities)
	desc.CapabilityGroups = groupCapabilities(desc.Capabilities)
	desc.Resources = normalizeResources(desc.Resources)
	return desc
}

func normalizeMessages(in []MessageDescriptor, kind MessageKind) []MessageDescriptor {
	if len(in) == 0 {
		return nil
	}
	out := make([]MessageDescriptor, 0, len(in))
	seen := map[string]struct{}{}
	for _, msg := range in {
		if msg.Topic == "" {
			continue
		}
		if msg.Kind == "" {
			msg.Kind = kind
		}
		key := string(msg.Kind) + "\x00" + msg.Topic
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, msg)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Topic == out[j].Topic {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Topic < out[j].Topic
	})
	return out
}

func normalizeCapabilities(in []CapabilityDescriptor) []CapabilityDescriptor {
	if len(in) == 0 {
		return nil
	}
	out := make([]CapabilityDescriptor, 0, len(in))
	seen := map[string]struct{}{}
	for _, cap := range in {
		if cap.Name == "" {
			continue
		}
		key := string(cap.Direction) + "\x00" + cap.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cap)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Direction < out[j].Direction
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func groupCapabilities(caps []CapabilityDescriptor) *CapabilityGroups {
	if len(caps) == 0 {
		return nil
	}
	groups := CapabilityGroups{}
	for _, cap := range caps {
		switch cap.Direction {
		case CapabilityRequired:
			groups.Required = append(groups.Required, cap)
		case CapabilityOptional:
			groups.Optional = append(groups.Optional, cap)
		case CapabilityProvided:
			groups.Provided = append(groups.Provided, cap)
		}
	}
	if len(groups.Required) == 0 && len(groups.Optional) == 0 && len(groups.Provided) == 0 {
		return nil
	}
	return &groups
}

func normalizeResources(in []ResourceDescriptor) []ResourceDescriptor {
	if len(in) == 0 {
		return nil
	}
	out := make([]ResourceDescriptor, 0, len(in))
	seen := map[string]struct{}{}
	for _, res := range in {
		if res.Kind == "" || res.Name == "" {
			continue
		}
		key := string(res.Kind) + "\x00" + res.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res.Metadata = cloneStringMap(res.Metadata)
		out = append(out, res)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].Name < out[j].Name
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if k == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func typeName[T any]() string {
	return reflectTypeName(reflect.TypeOf((*T)(nil)).Elem())
}

func valueTypeName(value any) string {
	if value == nil {
		return ""
	}
	return reflectTypeName(reflect.TypeOf(value))
}

func reflectTypeName(t reflect.Type) string {
	if t == nil {
		return ""
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "" || t.Name() == "" {
		return t.String()
	}
	return t.PkgPath() + "." + t.Name()
}
