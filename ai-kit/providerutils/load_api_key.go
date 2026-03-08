// Ported from: packages/provider-utils/src/load-api-key.ts
package providerutils

import (
	"fmt"
	"os"
)

// LoadApiKeyOptions are the options for LoadApiKey.
type LoadApiKeyOptions struct {
	// ApiKey is the API key value, if provided directly.
	ApiKey *string
	// EnvironmentVariableName is the name of the environment variable to check.
	EnvironmentVariableName string
	// ApiKeyParameterName is the name of the parameter for error messages. Default: "apiKey".
	ApiKeyParameterName string
	// Description is the description of the provider for error messages.
	Description string
}

// LoadApiKey loads an API key from a direct parameter or an environment variable.
// Returns an error if the key cannot be found.
func LoadApiKey(opts LoadApiKeyOptions) (string, error) {
	paramName := opts.ApiKeyParameterName
	if paramName == "" {
		paramName = "apiKey"
	}

	if opts.ApiKey != nil {
		return *opts.ApiKey, nil
	}

	apiKey := os.Getenv(opts.EnvironmentVariableName)
	if apiKey == "" {
		return "", fmt.Errorf(
			"%s API key is missing. Pass it using the '%s' parameter or the %s environment variable.",
			opts.Description, paramName, opts.EnvironmentVariableName,
		)
	}

	return apiKey, nil
}
