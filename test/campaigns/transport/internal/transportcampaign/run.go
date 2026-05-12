package transportcampaign

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/test/suite/agents"
	"github.com/brainlet/brainkit/test/suite/bus"
	"github.com/brainlet/brainkit/test/suite/deploy"
	"github.com/brainlet/brainkit/test/suite/fs"
	"github.com/brainlet/brainkit/test/suite/gateway"
	"github.com/brainlet/brainkit/test/suite/health"
	"github.com/brainlet/brainkit/test/suite/mcp"
	"github.com/brainlet/brainkit/test/suite/persistence"
	"github.com/brainlet/brainkit/test/suite/registry"
	"github.com/brainlet/brainkit/test/suite/scheduling"
	"github.com/brainlet/brainkit/test/suite/secrets"
	"github.com/brainlet/brainkit/test/suite/tools"
	"github.com/brainlet/brainkit/test/suite/tracing"
	"github.com/brainlet/brainkit/test/suite/workflows"
)

type Domain string

const (
	DomainBus         Domain = "bus"
	DomainDeploy      Domain = "deploy"
	DomainTools       Domain = "tools"
	DomainAgents      Domain = "agents"
	DomainScheduling  Domain = "scheduling"
	DomainHealth      Domain = "health"
	DomainSecrets     Domain = "secrets"
	DomainRegistry    Domain = "registry"
	DomainMCP         Domain = "mcp"
	DomainWorkflows   Domain = "workflows"
	DomainTracing     Domain = "tracing"
	DomainFS          Domain = "fs"
	DomainPersistence Domain = "persistence"
	DomainGateway     Domain = "gateway"
)

type Shard struct {
	Name    string
	Domains []Domain
}

var (
	CoreShard = Shard{
		Name: "core",
		Domains: []Domain{
			DomainBus,
			DomainTools,
			DomainAgents,
			DomainHealth,
			DomainSecrets,
			DomainRegistry,
		},
	}

	RuntimeShard = Shard{
		Name: "runtime",
		Domains: []Domain{
			DomainDeploy,
			DomainScheduling,
			DomainWorkflows,
			DomainFS,
			DomainPersistence,
		},
	}

	IntegrationsShard = Shard{
		Name: "integrations",
		Domains: []Domain{
			DomainMCP,
			DomainTracing,
			DomainGateway,
		},
	}
)

// RunBackendShard runs one transport campaign shard. Backend and shard
// packages stay separate so each combination gets its own go test package
// timeout budget.
func RunBackendShard(t *testing.T, backend string, shard Shard) {
	t.Helper()

	if backend != "embedded" {
		campaigns.RequirePodman(t)
	}
	RunShard(t, backend, shard)
}

// RunAll runs the full transport-sensitive suite matrix against one backend.
// Prefer RunBackendShard for package-level timeout isolation.
func RunAll(t *testing.T, backend string) {
	t.Helper()

	RunShard(t, backend, CoreShard)
	RunShard(t, backend, RuntimeShard)
	RunShard(t, backend, IntegrationsShard)
}

func RunShard(t *testing.T, backend string, shard Shard) {
	t.Helper()

	if shard.Name == "" || len(shard.Domains) == 0 {
		t.Fatalf("empty transport campaign shard")
	}

	opts := []campaigns.InfraOption{campaigns.Transport(backend)}
	if shard.Contains(DomainMCP) {
		opts = append(opts, campaigns.MCP())
	}

	infra := campaigns.NewInfra(t, opts...)
	env := infra.Env(t)

	for _, domain := range shard.Domains {
		domain := domain
		t.Run(string(domain), func(t *testing.T) {
			runDomain(t, env, domain)
		})
	}
}

func (s Shard) Contains(want Domain) bool {
	for _, domain := range s.Domains {
		if domain == want {
			return true
		}
	}
	return false
}

func runDomain(t *testing.T, env *suite.TestEnv, domain Domain) {
	t.Helper()

	switch domain {
	case DomainBus:
		bus.Run(t, env)
	case DomainDeploy:
		deploy.Run(t, env)
	case DomainTools:
		tools.Run(t, env)
	case DomainAgents:
		agents.Run(t, env)
	case DomainScheduling:
		scheduling.Run(t, env)
	case DomainHealth:
		health.Run(t, env)
	case DomainSecrets:
		secrets.Run(t, env)
	case DomainRegistry:
		registry.Run(t, env)
	case DomainMCP:
		mcp.Run(t, env)
	case DomainWorkflows:
		workflows.Run(t, env)
	case DomainTracing:
		tracing.Run(t, env)
	case DomainFS:
		fs.Run(t, env)
	case DomainPersistence:
		persistence.Run(t, env)
	case DomainGateway:
		gateway.Run(t, env)
	default:
		t.Fatalf("unknown transport campaign domain %q", domain)
	}
}
