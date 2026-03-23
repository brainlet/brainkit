package registry

import (
	"fmt"
	"strings"
)

// EnvVarsForRegistration generates BRAINKIT_* env var entries from a registration.
// These are injected into the JS runtime's process.env so Mastra/AI SDK libs
// that read env vars just work.
func EnvVarsForRegistration(category, name string, reg any) map[string]string {
	prefix := "BRAINKIT_" + strings.ToUpper(strings.ReplaceAll(category, " ", "_")) + "_" + strings.ToUpper(name) + "_"
	vars := make(map[string]string)

	switch r := reg.(type) {
	// AI Providers
	case AIProviderRegistration:
		addProviderEnvVars(prefix, r, vars)
	case OpenAIProviderConfig:
		addIfSet(vars, prefix+"API_KEY", r.APIKey)
		addIfSet(vars, prefix+"BASE_URL", r.BaseURL)
	case AnthropicProviderConfig:
		addIfSet(vars, prefix+"API_KEY", r.APIKey)
		addIfSet(vars, prefix+"BASE_URL", r.BaseURL)
	case GoogleProviderConfig:
		addIfSet(vars, prefix+"API_KEY", r.APIKey)
		addIfSet(vars, prefix+"BASE_URL", r.BaseURL)

	// Vector Stores
	case VectorStoreRegistration:
		addVectorStoreEnvVars(prefix, r, vars)
	case LibSQLVectorConfig:
		addIfSet(vars, prefix+"URL", r.URL)
		addIfSet(vars, prefix+"AUTH_TOKEN", r.AuthToken)
	case PgVectorConfig:
		addIfSet(vars, prefix+"CONNECTION_STRING", r.ConnectionString)
		if r.Host != "" {
			addIfSet(vars, prefix+"HOST", r.Host)
			addIfSet(vars, prefix+"PORT", fmt.Sprintf("%d", r.Port))
			addIfSet(vars, prefix+"DATABASE", r.Database)
		}
	case MongoDBVectorConfig:
		addIfSet(vars, prefix+"URI", r.URI)
		addIfSet(vars, prefix+"DB_NAME", r.DBName)

	// Storages
	case StorageRegistration:
		addStorageEnvVars(prefix, r, vars)
	case LibSQLStorageConfig:
		addIfSet(vars, prefix+"URL", r.URL)
		addIfSet(vars, prefix+"AUTH_TOKEN", r.AuthToken)
	case PostgresStorageConfig:
		addIfSet(vars, prefix+"CONNECTION_STRING", r.ConnectionString)
	case MongoDBStorageConfig:
		addIfSet(vars, prefix+"URI", r.URI)
		addIfSet(vars, prefix+"DB_NAME", r.DBName)
	}

	return vars
}

func addProviderEnvVars(prefix string, reg AIProviderRegistration, vars map[string]string) {
	EnvVarsForRegistration("", "", reg.Config) // recurse with the config
	// Copy from inner call
	for k, v := range EnvVarsForRegistration("", "", reg.Config) {
		if k == "_" {
			continue
		}
		vars[prefix+strings.TrimPrefix(k, "BRAINKIT___")] = v
	}
}

func addVectorStoreEnvVars(prefix string, reg VectorStoreRegistration, vars map[string]string) {
	for k, v := range EnvVarsForRegistration("", "", reg.Config) {
		if k == "_" {
			continue
		}
		vars[prefix+strings.TrimPrefix(k, "BRAINKIT___")] = v
	}
}

func addStorageEnvVars(prefix string, reg StorageRegistration, vars map[string]string) {
	for k, v := range EnvVarsForRegistration("", "", reg.Config) {
		if k == "_" {
			continue
		}
		vars[prefix+strings.TrimPrefix(k, "BRAINKIT___")] = v
	}
}

func addIfSet(vars map[string]string, key, value string) {
	if value != "" {
		vars[key] = value
	}
}
