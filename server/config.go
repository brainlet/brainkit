package server

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/brainlet/brainkit"
	"gopkg.in/yaml.v3"
)

// FileConfig is the YAML shape for a server config file. LoadConfig
// unmarshals into this, substitutes environment variables, then
// projects onto the runtime Config by walking the module registry.
//
// Every module is driven via the `modules:` map. Keys not registered
// at binary-link time produce an error at load — typos surface
// loudly instead of silently disabling the module.
type FileConfig struct {
	Namespace    string                    `yaml:"namespace"`
	Transport    TransportYAML             `yaml:"transport"`
	FSRoot       string                    `yaml:"fs_root"`
	KitStorePath string                    `yaml:"kit_store_path"`
	SecretKey    string                    `yaml:"secret_key"`
	Providers    []ProviderYAML            `yaml:"providers"`
	Storages     map[string]StorageYAML    `yaml:"storages"`
	Vectors      map[string]VectorYAML     `yaml:"vectors"`
	Packages     []PackageYAML             `yaml:"packages"`
	Modules      map[string]yaml.Node      `yaml:"modules"`
}

// TransportYAML selects a transport backend from config.
type TransportYAML struct {
	Type     string `yaml:"type"` // memory, embedded, nats, amqp, redis
	URL      string `yaml:"url"`
	NATSName string `yaml:"nats_name"`
}

// ProviderYAML configures a single AI provider.
type ProviderYAML struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
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

// PackageYAML configures a package to auto-deploy at startup.
type PackageYAML struct {
	Path string `yaml:"path"`
}

// LoadConfig reads a YAML file, substitutes `$VAR` and `${VAR}`
// references against the process environment, and projects it onto
// the runtime Config. Unknown module keys return an error with a
// "did you mean" hint listing registered modules.
//
// Strict decoding is enabled: unknown top-level keys (including the
// old pre-registry shape where modules lived at the root — `gateway:`,
// `audit:`, `plugins:`, etc.) fail the load with a pointer to the
// `modules:` map they now belong under. This is the second half of
// the unknown-key guard: the modules map catches typos under
// `modules.<name>`, strict decode catches mis-nested keys at the root.
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("server: read config %q: %w", path, err)
	}
	expanded := expandEnv(string(raw))

	dec := yaml.NewDecoder(strings.NewReader(expanded))
	dec.KnownFields(true)
	var fc FileConfig
	if err := dec.Decode(&fc); err != nil {
		return Config{}, translateTopLevelYAMLError(err, path)
	}
	return fc.toConfig()
}

// topLevelModuleKeys is the set of YAML keys that used to live at
// the root of the FileConfig and now belong under `modules:`. When
// a user's old config trips strict decoding on one of these, we add
// the concrete fix hint instead of just "unknown field".
var topLevelModuleKeys = map[string]bool{
	"gateway":   true,
	"audit":     true,
	"tracing":   true,
	"probes":    true,
	"schedules": true,
	"mcp":       true,
	"discovery": true,
	"topology":  true,
	"workflow":  true,
	"plugins":   true,
	"harness":   true,
}

// translateTopLevelYAMLError wraps a strict-decode error with a
// pointer to the modules: map when the offending key is a legacy
// top-level module section. Falls back to the raw parse error
// otherwise.
func translateTopLevelYAMLError(err error, path string) error {
	msg := err.Error()
	for key := range topLevelModuleKeys {
		needle := "field " + key + " not found"
		if strings.Contains(msg, needle) {
			return fmt.Errorf("server: parse config %q: top-level %q is no longer accepted — move it under `modules:` as `modules.%s`", path, key, key)
		}
	}
	return fmt.Errorf("server: parse config %q: %w", path, err)
}

var envVarPattern = regexp.MustCompile(`\$\{?([A-Z_][A-Z0-9_]*)\}?`)

// expandEnv replaces $VAR and ${VAR} with os.Getenv lookups. Missing
// variables expand to empty strings, matching envsubst semantics.
func expandEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(name)
	})
}

func (fc FileConfig) toConfig() (Config, error) {
	cfg := Config{
		Namespace:    fc.Namespace,
		FSRoot:       fc.FSRoot,
		KitStorePath: fc.KitStorePath,
		SecretKey:    fc.SecretKey,
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

	// Walk `modules:` in a stable order (map iteration is randomized
	// otherwise) so startup logs and failure paths stay deterministic
	// across runs.
	names := make([]string, 0, len(fc.Modules))
	for k := range fc.Modules {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		factory, ok := brainkit.LookupModuleFactory(name)
		if !ok {
			return Config{}, unknownModuleError(name)
		}
		node := fc.Modules[name]
		decode := func(v any) error {
			if node.Kind == 0 {
				// Empty section (`modules.audit:` with no body) is
				// legal — the factory gets a zero decode and uses
				// its defaults.
				return nil
			}
			return node.Decode(v)
		}
		mod, err := factory.Build(brainkit.ModuleContext{
			FSRoot: cfg.FSRoot,
			Decode: decode,
		})
		if err != nil {
			return Config{}, fmt.Errorf("server: module %q: %w", name, err)
		}
		if mod != nil {
			cfg.Modules = append(cfg.Modules, mod)
		}
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

// unknownModuleError builds a "did you mean" error for a module key
// that isn't in the registry. Lists every registered name so users
// can see what's available in their binary.
func unknownModuleError(name string) error {
	registered := brainkit.RegisteredModuleNames()
	if len(registered) == 0 {
		return fmt.Errorf("server: unknown module %q (no modules registered — did the binary import them?)", name)
	}
	return fmt.Errorf("server: unknown module %q (registered: %v)", name, registered)
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
