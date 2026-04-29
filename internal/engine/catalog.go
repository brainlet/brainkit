package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/transport"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
)

type commandSpec struct {
	topic        string
	validate     func(json.RawMessage) error
	invokeKernel func(context.Context, *Kernel, json.RawMessage) (json.RawMessage, error)
	invokeNode   func(context.Context, *Node, json.RawMessage) (json.RawMessage, error)
}

// CommandSpec is the opaque registration handle produced by MakeCommand.
// Modules build one via module.Command and mount it through module.Host.Commands.
type CommandSpec = commandSpec

// MakeCommand builds a CommandSpec from a handler that only sees context + Req.
// Used by the public brainkit.Command generic wrapper — handlers capture any Kit
// / Module state they need through closures, so they don't need a *Kernel arg.
func MakeCommand[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, Req) (*Resp, error)) CommandSpec {
	var zero Req
	topic := zero.BusTopic()
	invoke := func(ctx context.Context, kernel *Kernel, payload json.RawMessage) (json.RawMessage, error) {
		decoded, err := decodeCommand[Req](payload, topic)
		if err != nil {
			return nil, err
		}
		cmdStart := time.Now()
		out, err := handler(ctx, decoded)
		cmdDuration := time.Since(cmdStart)
		callerID := transport.CallerIDFromContext(ctx)
		kernel.audit.BusCommandCompleted(topic, callerID, cmdDuration)
		if err != nil {
			return nil, err
		}
		if out == nil {
			return nil, nil
		}
		return json.Marshal(out)
	}
	return commandSpec{
		topic: topic,
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, topic)
			return err
		},
		invokeKernel: invoke,
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			return invoke(ctx, node.Kernel, payload)
		},
	}
}

func moduleCommand(spec bkmodule.CommandSpec) commandSpec {
	topic := spec.Topic
	if topic == "" {
		topic = spec.Name
	}
	return commandSpec{
		topic: topic,
		invokeKernel: func(ctx context.Context, _ *Kernel, payload json.RawMessage) (json.RawMessage, error) {
			return spec.Handle(ctx, payload)
		},
		invokeNode: func(ctx context.Context, _ *Node, payload json.RawMessage) (json.RawMessage, error) {
			return spec.Handle(ctx, payload)
		},
	}
}

func kernelCommand[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, *Kernel, Req) (*Resp, error)) commandSpec {
	var req Req
	return commandSpec{
		topic: req.BusTopic(),
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, req.BusTopic())
			return err
		},
		invokeKernel: func(ctx context.Context, kernel *Kernel, payload json.RawMessage) (json.RawMessage, error) {
			decoded, err := decodeCommand[Req](payload, req.BusTopic())
			if err != nil {
				return nil, err
			}
			cmdStart := time.Now()
			out, err := handler(ctx, kernel, decoded)
			cmdDuration := time.Since(cmdStart)
			callerID := transport.CallerIDFromContext(ctx)
			if err != nil {
				kernel.audit.BusCommandCompleted(req.BusTopic(), callerID, cmdDuration)
				return nil, err
			}
			kernel.audit.BusCommandCompleted(req.BusTopic(), callerID, cmdDuration)
			if out == nil {
				return nil, nil
			}
			return json.Marshal(out)
		},
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			return kernelCommand(handler).invokeKernel(ctx, node.Kernel, payload)
		},
	}
}

func nodeCommand[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, *Node, Req) (*Resp, error)) commandSpec {
	var req Req
	return commandSpec{
		topic: req.BusTopic(),
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, req.BusTopic())
			return err
		},
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			decoded, err := decodeCommand[Req](payload, req.BusTopic())
			if err != nil {
				return nil, err
			}
			out, err := handler(ctx, node, decoded)
			if err != nil {
				return nil, err
			}
			return json.Marshal(out)
		},
	}
}

func decodeCommand[T any](payload json.RawMessage, topic string) (T, error) {
	var out T
	if len(payload) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, transport.NewDecodeFailure(topic, err)
	}
	return out, nil
}

type commandRegistry struct {
	ordered []commandSpec
	byTopic map[string]commandSpec
}

func (r *commandRegistry) Lookup(topic string) (commandSpec, bool) {
	spec, ok := r.byTopic[topic]
	return spec, ok
}

func (r *commandRegistry) HasCommand(topic string) bool {
	_, ok := r.byTopic[topic]
	return ok
}

func (r *commandRegistry) Validate(topic string, payload json.RawMessage) error {
	spec, ok := r.byTopic[topic]
	if !ok || spec.validate == nil {
		return nil
	}
	return spec.validate(payload)
}

func (r *commandRegistry) Add(spec commandSpec) error {
	if spec.topic == "" {
		return fmt.Errorf("command topic is required")
	}
	if _, exists := r.byTopic[spec.topic]; exists {
		return fmt.Errorf("duplicate command topic registered: %s", spec.topic)
	}
	r.byTopic[spec.topic] = spec
	r.ordered = append(r.ordered, spec)
	return nil
}

func (r *commandRegistry) Remove(topic string) {
	delete(r.byTopic, topic)
	for i, spec := range r.ordered {
		if spec.topic == topic {
			copy(r.ordered[i:], r.ordered[i+1:])
			r.ordered = r.ordered[:len(r.ordered)-1]
			return
		}
	}
}

func (r *commandRegistry) BindingsForNode(node *Node) []transport.RawCommandBinding {
	bindings := make([]transport.RawCommandBinding, 0, len(r.ordered))
	for _, spec := range r.ordered {
		spec := spec
		if spec.invokeNode == nil && spec.invokeKernel == nil {
			continue
		}
		bindings = append(bindings, transport.RawCommandBinding{
			Name:  spec.topic,
			Topic: spec.topic,
			Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
				if spec.invokeNode != nil {
					return spec.invokeNode(ctx, node, payload)
				}
				return spec.invokeKernel(ctx, node.Kernel, payload)
			},
		})
	}
	return bindings
}

func buildCommandCatalog() *commandRegistry {
	// The core boot catalog is intentionally empty. Built-in bus command
	// surfaces are owned by hot-mountable modules; root reference commands are
	// registered separately from brainkit.New before module mounting.
	specs := []commandSpec{}

	byTopic := make(map[string]commandSpec, len(specs))
	for _, spec := range specs {
		if strings.HasSuffix(spec.topic, ".result") {
			panic(fmt.Sprintf("invalid command topic registered: %s", spec.topic))
		}
		if _, exists := byTopic[spec.topic]; exists {
			panic(fmt.Sprintf("duplicate command topic registered: %s", spec.topic))
		}
		byTopic[spec.topic] = spec
	}

	return &commandRegistry{
		ordered: specs,
		byTopic: byTopic,
	}
}

// commandBindingsForKernel generates router bindings for a standalone Kernel.
// Kernel-only commands are bound; node-only and unconfigured-domain commands are skipped.
func commandBindingsForKernel(kernel *Kernel) []transport.RawCommandBinding {
	bindings := make([]transport.RawCommandBinding, 0, len(kernel.catalog.ordered))
	for _, spec := range kernel.catalog.ordered {
		spec := spec
		if spec.invokeKernel == nil {
			continue // node-only command
		}
		bindings = append(bindings, transport.RawCommandBinding{
			Name:  spec.topic,
			Topic: spec.topic,
			Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
				return spec.invokeKernel(ctx, kernel, payload)
			},
		})
	}
	return bindings
}

// commandBindingsForNode generates router bindings for a Node.
// Includes both kernel commands (delegated to node.Kernel) and node-specific commands.
func commandBindingsForNode(node *Node) []transport.RawCommandBinding {
	return node.Kernel.catalog.BindingsForNode(node)
}

// RegisterCommand adds a command to the per-instance catalog.
// Called by modules during mount to register their bus commands.
// Panics on duplicate topic (same as core catalog construction).
func (k *Kernel) RegisterCommand(spec commandSpec) {
	if err := k.catalog.Add(spec); err != nil {
		panic(err.Error())
	}
}
