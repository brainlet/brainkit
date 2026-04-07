package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/sdk/messages"
)

// checkBusPermission checks if the current deployment source can perform a bus action.
// Returns nil if allowed, error if denied. No-op if RBAC is not configured.
func (k *Kernel) checkBusPermission(source, topic, action string) error {
	if k.rbac == nil || source == "" {
		return nil // no RBAC or direct Go call — always allowed
	}
	if rbac.IsOwnMailbox(source, topic) {
		return nil // own mailbox is always accessible
	}

	role := k.rbac.RoleForSource(source)
	var allowed bool
	switch action {
	case "publish":
		allowed = role.Bus.Publish.Allows(topic)
	case "subscribe":
		allowed = role.Bus.Subscribe.Allows(topic)
	case "emit":
		allowed = role.Bus.Emit.Allows(topic)
	default:
		return nil
	}

	if !allowed {
		k.emitPermissionDenied(source, topic, action, role.Name)
		return &sdkerrors.PermissionDeniedError{Source: source, Action: action, Topic: topic, Role: role.Name}
	}
	return nil
}

// checkCommandPermission checks if the current deployment source can call a catalog command.
func (k *Kernel) checkCommandPermission(source, command string) error {
	if k.rbac == nil || source == "" {
		return nil
	}
	role := k.rbac.RoleForSource(source)
	if !role.Commands.AllowsCommand(command) {
		k.emitPermissionDenied(source, command, "command", role.Name)
		return &sdkerrors.PermissionDeniedError{Source: source, Action: "command", Topic: command, Role: role.Name}
	}
	return nil
}

// checkRegistrationPermission checks if the current deployment source can register a resource type.
func (k *Kernel) checkRegistrationPermission(source, resourceType string) error {
	if k.rbac == nil || source == "" {
		return nil
	}
	role := k.rbac.RoleForSource(source)
	switch resourceType {
	case "tool":
		if !role.Registration.Tools {
			k.emitPermissionDenied(source, resourceType, "register", role.Name)
			return &sdkerrors.PermissionDeniedError{Source: source, Action: "register", Topic: resourceType, Role: role.Name}
		}
	case "agent":
		if !role.Registration.Agents {
			k.emitPermissionDenied(source, resourceType, "register", role.Name)
			return &sdkerrors.PermissionDeniedError{Source: source, Action: "register", Topic: resourceType, Role: role.Name}
		}
	}
	return nil
}

// currentDeploymentSource returns the deployment source currently executing on the JS thread.
// Set by the subscribe callback in bridges.go when a handler fires.
// Returns "" if not in a deployment context (direct Go EvalTS or Go caller).
// Go-side tracking avoids re-entrant ctx.Eval issues in bridge callbacks.
func (k *Kernel) currentDeploymentSource() string {
	return k.currentSource
}

// setCurrentSource sets the active deployment source for RBAC enforcement.
// Only called from the JS thread (qctx.Schedule callbacks and Deploy), which is
// single-threaded — no mutex needed. Using k.mu here caused GoChannel message
// delivery reordering because __go_brainkit_bus_reply runs on the same goroutine
// and GoChannel Publish interacts with k.mu (via subscriber callbacks).
func (k *Kernel) setCurrentSource(source string) {
	k.currentSource = source
}

// emitPermissionDenied publishes the bus.permission.denied event.
func (k *Kernel) emitPermissionDenied(source, topic, action, roleName string) {
	payload, _ := json.Marshal(messages.PermissionDeniedEvent{
		Source: source, Topic: topic, Action: action,
		Role: roleName, Reason: fmt.Sprintf("%s denied %s on %s", roleName, action, topic),
	})
	// Publish directly — don't RBAC check the audit event itself
	k.remote.PublishRaw(context.Background(), "bus.permission.denied", payload)
}
