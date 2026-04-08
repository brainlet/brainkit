package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/sdk"
)

// RBACAdminDomain handles rbac.assign, rbac.revoke, rbac.list, rbac.roles bus commands.
type RBACAdminDomain struct {
	manager *rbac.Manager
}

func newRBACAdminDomain(manager *rbac.Manager) *RBACAdminDomain {
	return &RBACAdminDomain{manager: manager}
}

func (d *RBACAdminDomain) Assign(_ context.Context, req sdk.RBACAssignMsg) (*sdk.RBACAssignResp, error) {
	if d.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "rbac"}
	}
	if req.Source == "" {
		return nil, &sdkerrors.ValidationError{Field: "source", Message: "is required"}
	}
	if err := d.manager.Assign(req.Source, req.Role); err != nil {
		return nil, err
	}
	return &sdk.RBACAssignResp{Assigned: true}, nil
}

func (d *RBACAdminDomain) Revoke(_ context.Context, req sdk.RBACRevokeMsg) (*sdk.RBACRevokeResp, error) {
	if d.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "rbac"}
	}
	d.manager.Revoke(req.Source)
	return &sdk.RBACRevokeResp{Revoked: true}, nil
}

func (d *RBACAdminDomain) List(_ context.Context, _ sdk.RBACListMsg) (*sdk.RBACListResp, error) {
	if d.manager == nil {
		return &sdk.RBACListResp{Assignments: []sdk.RBACAssignmentInfo{}}, nil
	}
	assignments := d.manager.ListAssignments()
	infos := make([]sdk.RBACAssignmentInfo, len(assignments))
	for i, a := range assignments {
		infos[i] = sdk.RBACAssignmentInfo{
			Source: a.Source, Role: a.Role,
			AssignedAt: a.AssignedAt.Format(time.RFC3339),
		}
	}
	return &sdk.RBACListResp{Assignments: infos}, nil
}

func (d *RBACAdminDomain) Roles(_ context.Context, _ sdk.RBACRolesMsg) (*sdk.RBACRolesResp, error) {
	if d.manager == nil {
		return &sdk.RBACRolesResp{Roles: json.RawMessage("[]")}, nil
	}
	roles := d.manager.ListRoles()
	data, _ := json.Marshal(roles)
	return &sdk.RBACRolesResp{Roles: data}, nil
}
