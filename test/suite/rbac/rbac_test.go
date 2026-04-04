package rbac

import (
	"testing"

	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/test/suite"
)

func TestRBAC(t *testing.T) {
	env := suite.Full(t, suite.WithRBAC(map[string]rbac.Role{
		"admin":    rbac.RoleAdmin,
		"service":  rbac.RoleService,
		"gateway":  rbac.RoleGateway,
		"observer": rbac.RoleObserver,
	}, "service"), suite.WithPersistence())
	Run(t, env)
}
