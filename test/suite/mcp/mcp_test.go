package mcp

import (
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	mcppkg "github.com/brainlet/brainkit/modules/mcp"
	"github.com/brainlet/brainkit/test/suite"
)

func TestMCP(t *testing.T) {
	binary := testutil.BuildTestMCP(t)
	env := suite.Full(t, suite.WithMCP(map[string]mcppkg.ServerConfig{
		"testmcp": {Command: binary},
	}))
	Run(t, env)
}
