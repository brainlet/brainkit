package mcp

import (
	"testing"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
)

func TestMCP(t *testing.T) {
	binary := testutil.BuildTestMCP(t)
	env := suite.Full(t, suite.WithMCP(map[string]mcppkg.ServerConfig{
		"testmcp": {Command: binary},
	}))
	Run(t, env)
}
