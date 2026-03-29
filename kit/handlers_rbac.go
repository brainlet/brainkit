package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
)

// RBACDomain handles rbac.assign/revoke/list/roles bus commands.
type RBACDomain struct {
	kit *Kernel
}

func newRBACDomain(k *Kernel) *RBACDomain {
	return &RBACDomain{kit: k}
}

func (d *RBACDomain) Assign(_ context.Context, req messages.RBACAssignMsg) (*messages.RBACAssignResp, error) {
	if d.kit.rbac == nil {
		return nil, fmt.Errorf("rbac: not configured (no Roles in KernelConfig)")
	}
	if req.Source == "" {
		return nil, fmt.Errorf("rbac.assign: source is required")
	}
	if err := d.kit.rbac.Assign(req.Source, req.Role); err != nil {
		return nil, err
	}
	return &messages.RBACAssignResp{Assigned: true}, nil
}

func (d *RBACDomain) Revoke(_ context.Context, req messages.RBACRevokeMsg) (*messages.RBACRevokeResp, error) {
	if d.kit.rbac == nil {
		return nil, fmt.Errorf("rbac: not configured")
	}
	d.kit.rbac.Revoke(req.Source)
	return &messages.RBACRevokeResp{Revoked: true}, nil
}

func (d *RBACDomain) List(_ context.Context, _ messages.RBACListMsg) (*messages.RBACListResp, error) {
	if d.kit.rbac == nil {
		return &messages.RBACListResp{Assignments: []messages.RBACAssignmentInfo{}}, nil
	}
	assignments := d.kit.rbac.ListAssignments()
	infos := make([]messages.RBACAssignmentInfo, len(assignments))
	for i, a := range assignments {
		infos[i] = messages.RBACAssignmentInfo{
			Source:     a.Source,
			Role:       a.Role,
			AssignedAt: a.AssignedAt.Format(time.RFC3339),
		}
	}
	return &messages.RBACListResp{Assignments: infos}, nil
}

func (d *RBACDomain) Roles(_ context.Context, _ messages.RBACRolesMsg) (*messages.RBACRolesResp, error) {
	if d.kit.rbac == nil {
		return &messages.RBACRolesResp{Roles: json.RawMessage("[]")}, nil
	}
	roles := d.kit.rbac.ListRoles()
	data, _ := json.Marshal(roles)
	return &messages.RBACRolesResp{Roles: data}, nil
}
