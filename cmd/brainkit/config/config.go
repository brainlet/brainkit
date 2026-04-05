package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit"
	"github.com/spf13/viper"
)

type StorageEntry struct {
	Type             string `mapstructure:"type"`
	Path             string `mapstructure:"path"`
	ConnectionString string `mapstructure:"connection_string"`
	URI              string `mapstructure:"uri"`
	DBName           string `mapstructure:"db_name"`
	URL              string `mapstructure:"url"`
	Token            string `mapstructure:"token"`
}

type VectorEntry struct {
	Type             string `mapstructure:"type"`
	Path             string `mapstructure:"path"`
	ConnectionString string `mapstructure:"connection_string"`
	URI              string `mapstructure:"uri"`
	DBName           string `mapstructure:"db_name"`
}

type MCPServerEntry struct {
	Command string            `mapstructure:"command"`
	Args    []string          `mapstructure:"args"`
	Env     map[string]string `mapstructure:"env"`
	URL     string            `mapstructure:"url"`
}

type CLIConfig struct {
	Namespace   string                    `mapstructure:"namespace"`
	EnvFile     string                    `mapstructure:"env_file"`
	Transport   string                    `mapstructure:"transport"`
	NATSURL     string                    `mapstructure:"nats_url"`
	NATSName    string                    `mapstructure:"nats_name"`
	AMQPURL     string                    `mapstructure:"amqp_url"`
	RedisURL    string                    `mapstructure:"redis_url"`
	PostgresURL string                    `mapstructure:"postgres_url"`
	SQLitePath  string                    `mapstructure:"sqlite_path"`
	Storage     map[string]StorageEntry   `mapstructure:"storage"`
	Vectors     map[string]VectorEntry    `mapstructure:"vectors"`
	FSRoot      string                    `mapstructure:"fs_root"`
	SecretKey   string                    `mapstructure:"secret_key"`
	StorePath   string                    `mapstructure:"store_path"`
	MCPServers  map[string]MCPServerEntry `mapstructure:"mcp_servers"`
}

func LoadConfig() (*CLIConfig, error) {
	var cfg CLIConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	loadEnvFile(cfg.EnvFile)
	return &cfg, nil
}

// loadEnvFile reads a .env file and sets environment variables.
// If path is empty, tries .env in the current directory (silent if missing).
func loadEnvFile(path string) {
	if path == "" {
		path = ".env"
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Don't override existing env vars
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func BuildNodeConfig(cfg *CLIConfig) (brainkit.NodeConfig, error) {
	nc := brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: cfg.Namespace,
			FSRoot:    cfg.FSRoot,
			SecretKey: cfg.SecretKey,
			LogHandler: func(e brainkit.LogEntry) {
				fmt.Fprintf(os.Stderr, "[%s] [%s] %s\n", e.Source, e.Level, e.Message)
			},
		},
		Messaging: brainkit.MessagingConfig{
			Transport:   cfg.Transport,
			NATSURL:     cfg.NATSURL,
			NATSName:    cfg.NATSName,
			AMQPURL:     cfg.AMQPURL,
			RedisURL:    cfg.RedisURL,
			PostgresURL: cfg.PostgresURL,
			SQLitePath:  cfg.SQLitePath,
		},
	}

	if len(cfg.Storage) > 0 {
		nc.Kernel.Storages = make(map[string]brainkit.StorageConfig)
		for name, entry := range cfg.Storage {
			nc.Kernel.Storages[name] = mapStorage(entry)
		}
	}

	if len(cfg.Vectors) > 0 {
		nc.Kernel.Vectors = make(map[string]brainkit.VectorConfig)
		for name, entry := range cfg.Vectors {
			nc.Kernel.Vectors[name] = mapVector(entry)
		}
	}

	if len(cfg.MCPServers) > 0 {
		nc.Kernel.MCPServers = make(map[string]brainkit.MCPServerConfig)
		for name, entry := range cfg.MCPServers {
			nc.Kernel.MCPServers[name] = brainkit.MCPServerConfig{
				Command: entry.Command,
				Args:    entry.Args,
				Env:     entry.Env,
				URL:     entry.URL,
			}
		}
	}

	if cfg.StorePath != "" {
		store, err := brainkit.NewSQLiteStore(cfg.StorePath)
		if err != nil {
			return nc, fmt.Errorf("open store: %w", err)
		}
		nc.Kernel.Store = store
	}

	return nc, nil
}

func mapStorage(e StorageEntry) brainkit.StorageConfig {
	switch e.Type {
	case "sqlite":
		return brainkit.SQLiteStorage(e.Path)
	case "postgres":
		return brainkit.PostgresStorage(e.ConnectionString)
	case "mongodb":
		return brainkit.MongoDBStorage(e.URI, e.DBName)
	case "upstash":
		return brainkit.UpstashStorage(e.URL, e.Token)
	case "memory":
		return brainkit.InMemoryStorage()
	default:
		return brainkit.InMemoryStorage()
	}
}

func mapVector(e VectorEntry) brainkit.VectorConfig {
	switch e.Type {
	case "sqlite":
		return brainkit.SQLiteVector(e.Path)
	case "pgvector":
		return brainkit.PgVectorStore(e.ConnectionString)
	case "mongodb":
		return brainkit.MongoDBVectorStore(e.URI, e.DBName)
	default:
		return brainkit.SQLiteVector(e.Path)
	}
}

func SanitizeNamespace(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	return strings.ToLower(re.ReplaceAllString(name, "-"))
}

func DefaultNamespace() string {
	dir, err := os.Getwd()
	if err != nil {
		return "brainkit"
	}
	return SanitizeNamespace(filepath.Base(dir))
}
