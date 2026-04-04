// Package campaigns provides the Infra builder for campaign-level integration tests.
// Campaigns test brainkit across backend combinations: transport x storage x vector x auth.
// The Infra builder manages containers, creates TestEnvs, and integrates with the fixture runner.
package campaigns

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/testcontainers/testcontainers-go/wait"
)

// InfraOption configures an Infra builder.
type InfraOption func(*infraConfig)

type infraConfig struct {
	transport   string
	storages    map[string]string // backend name → placeholder (resolved at Env() time)
	vectors     map[string]string // backend name → placeholder (resolved at Env() time)
	persistence bool
	rbac        bool
	tracing     bool
	ai          bool
	mcp         bool
	plugins     []brainkit.PluginConfig
	nodeCount   int
}

// Infra manages containers and builds TestEnvs for campaign tests.
// Containers are started lazily on first use and shared across Env() calls.
type Infra struct {
	t   *testing.T
	cfg infraConfig

	postgresAddr string
	postgresOnce sync.Once
	mongoAddr    string
	mongoOnce    sync.Once
	libsqlAddr   string
	libsqlOnce   sync.Once
}

// NewInfra creates a campaign Infra builder. All containers are started lazily.
// Callers that need Podman-backed containers should call RequirePodman(t) before NewInfra.
func NewInfra(t *testing.T, opts ...InfraOption) *Infra {
	t.Helper()

	cfg := infraConfig{
		storages: make(map[string]string),
		vectors:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Infra{t: t, cfg: cfg}
}

// RequirePodman skips the test if Podman is not available.
func RequirePodman(t *testing.T) {
	t.Helper()
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
}

// ── Option functions ────────────────────────────────────────────────────────

// Transport sets the transport backend: "nats", "amqp", "redis", "sql-sqlite", "sql-postgres".
func Transport(backend string) InfraOption {
	return func(c *infraConfig) { c.transport = backend }
}

// Storage adds a storage backend: "postgres", "mongodb", "libsql".
func Storage(backend string) InfraOption {
	return func(c *infraConfig) {
		if c.storages == nil {
			c.storages = make(map[string]string)
		}
		c.storages[backend] = backend
	}
}

// Vector adds a vector store backend: "pgvector", "libsql", "mongodb".
func Vector(backend string) InfraOption {
	return func(c *infraConfig) {
		if c.vectors == nil {
			c.vectors = make(map[string]string)
		}
		c.vectors[backend] = backend
	}
}

// Persistence enables SQLite KitStore persistence.
func Persistence() InfraOption {
	return func(c *infraConfig) { c.persistence = true }
}

// RBAC enables RBAC with default test roles.
func RBAC() InfraOption {
	return func(c *infraConfig) { c.rbac = true }
}

// Tracing enables in-memory trace store.
func Tracing() InfraOption {
	return func(c *infraConfig) { c.tracing = true }
}

// AI enables AI provider auto-detection from .env.
func AI() InfraOption {
	return func(c *infraConfig) { c.ai = true }
}

// MCP enables MCP server configuration.
func MCP() InfraOption {
	return func(c *infraConfig) { c.mcp = true }
}

// Plugins adds plugin configurations.
func Plugins(configs ...brainkit.PluginConfig) InfraOption {
	return func(c *infraConfig) { c.plugins = append(c.plugins, configs...) }
}

// Nodes sets the node count for cross-kit tests.
func Nodes(count int) InfraOption {
	return func(c *infraConfig) { c.nodeCount = count }
}

// ── Container management ────────────────────────────────────────────────────
// Containers are started lazily via sync.Once. The same container is reused
// when both transport and storage need postgres, or when both storage and
// vector need mongodb.

func (inf *Infra) ensurePostgres() string {
	inf.postgresOnce.Do(func() {
		inf.postgresAddr = testutil.StartContainer(inf.t,
			"pgvector/pgvector:pg16", "5432/tcp", nil,
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
			"POSTGRES_USER=test", "POSTGRES_PASSWORD=test", "POSTGRES_DB=brainkit")
	})
	return inf.postgresAddr
}

func (inf *Infra) ensureMongoDB() string {
	inf.mongoOnce.Do(func() {
		inf.mongoAddr = testutil.StartContainer(inf.t,
			"mongo:7", "27017/tcp", nil,
			wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
			"MONGO_INITDB_ROOT_USERNAME=test", "MONGO_INITDB_ROOT_PASSWORD=test")
	})
	return inf.mongoAddr
}

func (inf *Infra) ensureLibSQL() string {
	inf.libsqlOnce.Do(func() {
		inf.libsqlAddr = testutil.StartContainer(inf.t,
			"ghcr.io/tursodatabase/libsql-server:latest",
			"8080/tcp",
			[]string{"sqld", "--http-listen-addr", "0.0.0.0:8080"},
			wait.ForHTTP("/health").WithStartupTimeout(30*time.Second))
	})
	return inf.libsqlAddr
}

// ── Env construction ────────────────────────────────────────────────────────

// Env creates a TestEnv with the configured infrastructure.
// Extra options layer on top of the infra config.
func (inf *Infra) Env(t *testing.T, extra ...suite.EnvOption) *suite.TestEnv {
	t.Helper()

	var opts []suite.EnvOption

	// Transport — delegates container management to suite.NewEnv via testutil.
	if inf.cfg.transport != "" {
		opts = append(opts, suite.WithTransport(inf.cfg.transport))
	}

	// Storages — start containers and resolve to real configs.
	for backend := range inf.cfg.storages {
		switch backend {
		case "postgres":
			addr := inf.ensurePostgres()
			connStr := fmt.Sprintf("postgresql://test:test@%s/brainkit", addr)
			opts = append(opts, suite.WithStorage("postgres", brainkit.PostgresStorage(connStr)))
		case "mongodb":
			addr := inf.ensureMongoDB()
			uri := fmt.Sprintf("mongodb://test:test@%s", addr)
			opts = append(opts, suite.WithStorage("mongodb", brainkit.MongoDBStorage(uri, "brainkit")))
		case "libsql":
			addr := inf.ensureLibSQL()
			url := "http://" + addr
			opts = append(opts, suite.WithStorage("libsql", brainkit.StorageConfig{
				Type: "sqlite",
				Path: url,
			}))
		}
	}

	// Vectors — start containers and resolve to real configs.
	for backend := range inf.cfg.vectors {
		switch backend {
		case "pgvector":
			addr := inf.ensurePostgres()
			connStr := fmt.Sprintf("postgresql://test:test@%s/brainkit", addr)
			opts = append(opts, suite.WithVector("pgvector", brainkit.PgVectorStore(connStr)))
		case "mongodb":
			addr := inf.ensureMongoDB()
			uri := fmt.Sprintf("mongodb://test:test@%s", addr)
			opts = append(opts, suite.WithVector("mongodb", brainkit.MongoDBVectorStore(uri, "brainkit")))
		case "libsql":
			addr := inf.ensureLibSQL()
			url := "http://" + addr
			opts = append(opts, suite.WithVector("libsql", brainkit.VectorConfig{
				Type: "sqlite",
				Path: url,
			}))
		}
	}

	// Feature flags
	if inf.cfg.persistence {
		opts = append(opts, suite.WithPersistence())
	}
	if inf.cfg.rbac {
		opts = append(opts, suite.WithRBAC(defaultTestRoles(), "service"), suite.WithPersistence())
	}
	if inf.cfg.tracing {
		opts = append(opts, suite.WithTracing())
	}
	if inf.cfg.ai {
		opts = append(opts, suite.WithAI())
	}
	if inf.cfg.nodeCount > 0 {
		opts = append(opts, suite.WithNodes(inf.cfg.nodeCount))
	}
	if len(inf.cfg.plugins) > 0 {
		opts = append(opts, suite.WithPlugins(inf.cfg.plugins...))
	}

	// Layer caller-provided options on top
	opts = append(opts, extra...)

	return suite.Full(t, opts...)
}

// RunFixtures creates a fixture Runner and runs fixtures matching the given patterns.
func (inf *Infra) RunFixtures(t *testing.T, patterns ...string) {
	t.Helper()

	root := fixturesRoot(t)
	runner := fixtures.NewRunner(root)

	// Wire up a kernel factory that uses our infra containers.
	runner.WithKernelFactory(func(t *testing.T, needs fixtures.FixtureNeeds) *brainkit.Kernel {
		t.Helper()
		env := inf.Env(t)
		return env.Kernel
	})

	runner.RunMatching(t, patterns...)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// defaultTestRoles returns the standard RBAC role set used by campaign tests.
func defaultTestRoles() map[string]rbac.Role {
	return map[string]rbac.Role{
		"admin":    rbac.RoleAdmin,
		"service":  rbac.RoleService,
		"gateway":  rbac.RoleGateway,
		"observer": rbac.RoleObserver,
	}
}

// fixturesRoot returns the absolute path to the fixtures/ directory.
func fixturesRoot(t *testing.T) string {
	t.Helper()
	root := projectRoot(t)
	return filepath.Join(root, "fixtures")
}

// projectRoot walks up from cwd to find the directory containing go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("project root (go.mod) not found")
		}
		dir = parent
	}
}
