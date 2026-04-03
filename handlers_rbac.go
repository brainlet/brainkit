package brainkit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk/messages"
)

// RBACAdminDomain handles rbac.assign, rbac.revoke, rbac.list, rbac.roles bus commands.
type RBACAdminDomain struct {
	manager *rbac.Manager
}

func newRBACAdminDomain(manager *rbac.Manager) *RBACAdminDomain {
	return &RBACAdminDomain{manager: manager}
}

func (d *RBACAdminDomain) Assign(_ context.Context, req messages.RBACAssignMsg) (*messages.RBACAssignResp, error) {
	if d.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "rbac"}
	}
	if req.Source == "" {
		return nil, &sdkerrors.ValidationError{Field: "source", Message: "is required"}
	}
	if err := d.manager.Assign(req.Source, req.Role); err != nil {
		return nil, err
	}
	return &messages.RBACAssignResp{Assigned: true}, nil
}

func (d *RBACAdminDomain) Revoke(_ context.Context, req messages.RBACRevokeMsg) (*messages.RBACRevokeResp, error) {
	if d.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "rbac"}
	}
	d.manager.Revoke(req.Source)
	return &messages.RBACRevokeResp{Revoked: true}, nil
}

func (d *RBACAdminDomain) List(_ context.Context, _ messages.RBACListMsg) (*messages.RBACListResp, error) {
	if d.manager == nil {
		return &messages.RBACListResp{Assignments: []messages.RBACAssignmentInfo{}}, nil
	}
	assignments := d.manager.ListAssignments()
	infos := make([]messages.RBACAssignmentInfo, len(assignments))
	for i, a := range assignments {
		infos[i] = messages.RBACAssignmentInfo{
			Source: a.Source, Role: a.Role,
			AssignedAt: a.AssignedAt.Format(time.RFC3339),
		}
	}
	return &messages.RBACListResp{Assignments: infos}, nil
}

func (d *RBACAdminDomain) Roles(_ context.Context, _ messages.RBACRolesMsg) (*messages.RBACRolesResp, error) {
	if d.manager == nil {
		return &messages.RBACRolesResp{Roles: json.RawMessage("[]")}, nil
	}
	roles := d.manager.ListRoles()
	data, _ := json.Marshal(roles)
	return &messages.RBACRolesResp{Roles: data}, nil
}
