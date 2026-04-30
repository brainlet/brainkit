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
)

// CapabilityDescriptor describes one named module host capability.
type CapabilityDescriptor struct {
	Name      string              `json:"name"`
	Direction CapabilityDirection `json:"direction,omitempty"`
	Type      string              `json:"type,omitempty"`
	Summary   string              `json:"summary,omitempty"`
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
// falls back to ID/status/dependency interfaces on the module value itself.
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
	if len(desc.Requires) == 0 {
		if deps, ok := mod.(DependencyReporter); ok {
			desc.Requires = deps.Dependencies()
		}
	}
	return NormalizeDescriptor(id, desc)
}

// DependenciesOf returns module dependencies using descriptor metadata as the
// source of truth, with DependencyReporter retained as a fallback for modules
// that have not added descriptors yet.
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
