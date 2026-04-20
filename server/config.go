package server

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
	"gopkg.in/yaml.v3"
)

// FileConfig is the YAML shape for a server config file. LoadConfig
// unmarshals into this, substitutes environment variables, then
// projects onto the runtime Config.
//
// Module sections (audit, tracing, probes, schedules, mcp, discovery,
// topology, workflow) are opt-in — presence of the key enables the
// module, absence leaves it off. The single exception is `gateway`,
// which remains required because server mode exists to serve HTTP
// traffic.
type FileConfig struct {
	Namespace    string                 `yaml:"namespace"`
	Transport    TransportYAML          `yaml:"transport"`
	FSRoot       string                 `yaml:"fs_root"`
	KitStorePath string                 `yaml:"kit_store_path"`
	SecretKey    string                 `yaml:"secret_key"`
	Gateway      GatewayYAML            `yaml:"gateway"`
	Providers    []ProviderYAML         `yaml:"providers"`
	Storages     map[string]StorageYAML `yaml:"storages"`
	Vectors      map[string]VectorYAML  `yaml:"vectors"`
	Plugins      []PluginYAML           `yaml:"plugins"`
	Packages     []PackageYAML          `yaml:"packages"`

	// Optional module sections — presence = module enabled.
	Audit     *AuditYAML     `yaml:"audit,omitempty"`
	Tracing   *TracingYAML   `yaml:"tracing,omitempty"`
	Probes    *ProbesYAML    `yaml:"probes,omitempty"`
	Schedules *SchedulesYAML `yaml:"schedules,omitempty"`
	MCP       *MCPYAML       `yaml:"mcp,omitempty"`
	Discovery *DiscoveryYAML `yaml:"discovery,omitempty"`
	Topology  *TopologyYAML  `yaml:"topology,omitempty"`
	Workflow  *WorkflowYAML  `yaml:"workflow,omitempty"`
}

// TransportYAML selects a transport backend from config.
type TransportYAML struct {
	Type     string `yaml:"type"` // memory, embedded, nats, amqp, redis
	URL      string `yaml:"url"`
	NATSName string `yaml:"nats_name"`
}

// GatewayYAML configures the HTTP gateway.
type GatewayYAML struct {
	Listen  string        `yaml:"listen"`
	Timeout time.Duration `yaml:"timeout"`
}

// ProviderYAML configures a single AI provider.
type ProviderYAML struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"` // openai, anthropic, ...
	APIKey string `yaml:"api_key"`
}

// StorageYAML configures a storage backend.
type StorageYAML struct {
	Type             string `yaml:"type"`
	Path             string `yaml:"path"`
	ConnectionString string `yaml:"connection_string"`
	URI              string `yaml:"uri"`
	DBName           string `yaml:"db_name"`
	URL              string `yaml:"url"`
	Token            string `yaml:"token"`
}

// VectorYAML configures a vector store.
type VectorYAML struct {
	Type             string `yaml:"type"`
	Path             string `yaml:"path"`
	ConnectionString string `yaml:"connection_string"`
	URI              string `yaml:"uri"`
	DBName           string `yaml:"db_name"`
}

// PluginYAML configures a subprocess plugin.
type PluginYAML struct {
	Name   string            `yaml:"name"`
	Binary string            `yaml:"binary"`
	Env    map[string]string `yaml:"env"`
}

// AuditYAML configures the audit module. Presence enables the
// module; empty path falls back to `<FSRoot>/audit.db`.
type AuditYAML struct {
	Path    string `yaml:"path"`
	Verbose bool   `yaml:"verbose"`
}

// TracingYAML configures the tracing module. Presence enables the
// module; empty path falls back to `<FSRoot>/tracing.db`.
type TracingYAML struct {
	Path      string        `yaml:"path"`
	Retention time.Duration `yaml:"retention"`
}

// ProbesYAML configures periodic health probing. Presence enables
// the module; zero Interval means "run probes only on register".
type ProbesYAML struct {
	Interval        time.Duration `yaml:"interval"`
	ProbeOnRegister *bool         `yaml:"probe_on_register"`
}

// SchedulesYAML configures the scheduling module. Presence enables
// the module; empty path = in-memory only (no persistence).
type SchedulesYAML struct {
	Path string `yaml:"path"`
}

// MCPYAML configures the MCP-client module. The `servers` map keys
// name each server; at least one server entry is required.
type MCPYAML struct {
	Servers map[string]MCPServerYAML `yaml:"servers"`
}

// MCPServerYAML configures a single MCP server connection — either
// a local subprocess (Command/Args/Env) or a remote HTTP endpoint
// (URL). Exactly one mode must be set.
type MCPServerYAML struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
	URL     string            `yaml:"url"`
}

// DiscoveryYAML configures peer discovery. `type` selects the
// backend: "static" (hand-rolled peer list), "bus" (broadcast
// presence on the kit's transport), or "" (disabled; skip including
// the discovery: key to disable).
type DiscoveryYAML struct {
	Type      string            `yaml:"type"`
	Name      string            `yaml:"name"`
	Heartbeat time.Duration     `yaml:"heartbeat"`
	TTL       time.Duration     `yaml:"ttl"`
	Peers     []DiscoveryPeerYAML `yaml:"peers"`
}

// DiscoveryPeerYAML is a static peer entry used when type == "static"
// or when supplementing the dynamic list in bus mode.
type DiscoveryPeerYAML struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Address   string            `yaml:"address"`
	Meta      map[string]string `yaml:"meta"`
}

// TopologyYAML configures cross-kit routing. `peers` is the static
// fallback list; `use_discovery: true` wires the discovery module
// as a dynamic ProviderSource (requires a `discovery:` section).
type TopologyYAML struct {
	Peers        []DiscoveryPeerYAML `yaml:"peers"`
	UseDiscovery bool                `yaml:"use_discovery"`
}

// WorkflowYAML configures the workflow module. Today it's a pure
// toggle — the module takes no options — but the type exists so
// future options can be added without breaking presence-based
// enablement.
type WorkflowYAML struct{}

// PackageYAML configures a package to auto-deploy at startup.
type PackageYAML struct {
	Path string `yaml:"path"`
}

// LoadConfig reads a YAML file, substitutes `$VAR` and `${VAR}`
// references against the process environment, and validates the
// projected runtime Config.
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("server: read config %q: %w", path, err)
	}
	expanded := expandEnv(string(raw))

	var fc FileConfig
	if err := yaml.Unmarshal([]byte(expanded), &fc); err != nil {
		return Config{}, fmt.Errorf("server: parse config %q: %w", path, err)
	}
	return fc.toConfig()
}

// envVarPattern matches $VAR and ${VAR} forms.
var envVarPattern = regexp.MustCompile(`\$\{?([A-Z_][A-Z0-9_]*)\}?`)

// expandEnv replaces $VAR and ${VAR} with os.Getenv lookups. Missing
// variables expand to empty strings, matching envsubst semantics.
func expandEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match
		name = envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(name)
	})
}

func (fc FileConfig) toConfig() (Config, error) {
	cfg := Config{
		Namespace:    fc.Namespace,
		FSRoot:       fc.FSRoot,
		KitStorePath: fc.KitStorePath,
		SecretKey:    fc.SecretKey,
		Gateway: gateway.Config{
			Listen:  fc.Gateway.Listen,
			Timeout: fc.Gateway.Timeout,
		},
	}

	transport, err := fc.Transport.build()
	if err != nil {
		return Config{}, err
	}
	cfg.Transport = transport

	for _, p := range fc.Providers {
		prov, err := buildProvider(p)
		if err != nil {
			return Config{}, err
		}
		cfg.Providers = append(cfg.Providers, prov)
	}

	if len(fc.Storages) > 0 {
		cfg.Storages = make(map[string]brainkit.StorageConfig, len(fc.Storages))
		for name, s := range fc.Storages {
			cfg.Storages[name] = buildStorage(s)
		}
	}
	if len(fc.Vectors) > 0 {
		cfg.Vectors = make(map[string]brainkit.VectorConfig, len(fc.Vectors))
		for name, v := range fc.Vectors {
			cfg.Vectors[name] = buildVector(v)
		}
	}

	for _, p := range fc.Plugins {
		cfg.Plugins = append(cfg.Plugins, brainkit.PluginConfig{
			Name:   p.Name,
			Binary: p.Binary,
			Env:    p.Env,
		})
	}

	if fc.Audit != nil {
		cfg.Audit = &AuditConfig{Path: fc.Audit.Path, Verbose: fc.Audit.Verbose}
	}
	if fc.Tracing != nil {
		cfg.Tracing = &TracingConfig{
			Path:      fc.Tracing.Path,
			Retention: fc.Tracing.Retention,
		}
	}
	if fc.Probes != nil {
		probeOnRegister := true
		if fc.Probes.ProbeOnRegister != nil {
			probeOnRegister = *fc.Probes.ProbeOnRegister
		}
		cfg.Probes = &ProbesConfig{
			Interval:        fc.Probes.Interval,
			ProbeOnRegister: probeOnRegister,
		}
	}
	if fc.Schedules != nil {
		cfg.Schedules = &SchedulesConfig{Path: fc.Schedules.Path}
	}
	if fc.MCP != nil {
		if len(fc.MCP.Servers) == 0 {
			return Config{}, fmt.Errorf("server: mcp: at least one server entry required")
		}
		servers := make(map[string]brainkit.MCPServerConfig, len(fc.MCP.Servers))
		for name, s := range fc.MCP.Servers {
			servers[name] = brainkit.MCPServerConfig{
				Command: s.Command,
				Args:    s.Args,
				Env:     s.Env,
				URL:     s.URL,
			}
		}
		cfg.MCP = &MCPConfig{Servers: servers}
	}
	if fc.Discovery != nil {
		dcfg := &DiscoveryConfig{
			Type:      fc.Discovery.Type,
			Name:      fc.Discovery.Name,
			Heartbeat: fc.Discovery.Heartbeat,
			TTL:       fc.Discovery.TTL,
		}
		for _, p := range fc.Discovery.Peers {
			dcfg.Peers = append(dcfg.Peers, DiscoveryPeer{
				Name:      p.Name,
				Namespace: p.Namespace,
				Address:   p.Address,
				Meta:      p.Meta,
			})
		}
		cfg.Discovery = dcfg
	}
	if fc.Topology != nil {
		tcfg := &TopologyConfig{UseDiscovery: fc.Topology.UseDiscovery}
		for _, p := range fc.Topology.Peers {
			tcfg.Peers = append(tcfg.Peers, DiscoveryPeer{
				Name:      p.Name,
				Namespace: p.Namespace,
				Address:   p.Address,
				Meta:      p.Meta,
			})
		}
		cfg.Topology = tcfg
	}
	if fc.Workflow != nil {
		cfg.Workflow = &WorkflowConfig{}
	}

	for _, pkg := range fc.Packages {
		p, err := brainkit.PackageFromDir(pkg.Path)
		if err != nil {
			return Config{}, fmt.Errorf("server: load package %q: %w", pkg.Path, err)
		}
		cfg.Packages = append(cfg.Packages, p)
	}

	return cfg, nil
}

func (t TransportYAML) build() (brainkit.TransportConfig, error) {
	switch t.Type {
	case "memory":
		return brainkit.Memory(), nil
	case "embedded", "":
		return brainkit.EmbeddedNATS(), nil
	case "nats":
		var opts []brainkit.TransportOption
		if t.NATSName != "" {
			opts = append(opts, brainkit.WithNATSName(t.NATSName))
		}
		return brainkit.NATS(t.URL, opts...), nil
	case "amqp":
		return brainkit.AMQP(t.URL), nil
	case "redis":
		return brainkit.Redis(t.URL), nil
	default:
		return brainkit.TransportConfig{}, fmt.Errorf("server: unknown transport %q", t.Type)
	}
}

func buildProvider(p ProviderYAML) (brainkit.ProviderConfig, error) {
	switch p.Type {
	case "openai":
		return brainkit.OpenAI(p.APIKey), nil
	case "anthropic":
		return brainkit.Anthropic(p.APIKey), nil
	case "google":
		return brainkit.Google(p.APIKey), nil
	case "mistral":
		return brainkit.Mistral(p.APIKey), nil
	case "groq":
		return brainkit.Groq(p.APIKey), nil
	case "deepseek":
		return brainkit.DeepSeek(p.APIKey), nil
	case "xai":
		return brainkit.XAI(p.APIKey), nil
	case "cohere":
		return brainkit.Cohere(p.APIKey), nil
	case "perplexity":
		return brainkit.Perplexity(p.APIKey), nil
	case "togetherai":
		return brainkit.TogetherAI(p.APIKey), nil
	case "fireworks":
		return brainkit.Fireworks(p.APIKey), nil
	case "cerebras":
		return brainkit.Cerebras(p.APIKey), nil
	default:
		return brainkit.ProviderConfig{}, fmt.Errorf("server: unknown provider type %q", p.Type)
	}
}

func buildStorage(s StorageYAML) brainkit.StorageConfig {
	switch s.Type {
	case "sqlite":
		return brainkit.SQLiteStorage(s.Path)
	case "postgres":
		return brainkit.PostgresStorage(s.ConnectionString)
	case "mongodb":
		return brainkit.MongoDBStorage(s.URI, s.DBName)
	case "upstash":
		return brainkit.UpstashStorage(s.URL, s.Token)
	case "memory":
		return brainkit.InMemoryStorage()
	default:
		return brainkit.InMemoryStorage()
	}
}

func buildVector(v VectorYAML) brainkit.VectorConfig {
	switch v.Type {
	case "sqlite":
		return brainkit.SQLiteVector(v.Path)
	case "pgvector":
		return brainkit.PgVectorStore(v.ConnectionString)
	case "mongodb":
		return brainkit.MongoDBVectorStore(v.URI, v.DBName)
	default:
		return brainkit.SQLiteVector(v.Path)
	}
}
