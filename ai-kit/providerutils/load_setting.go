// Ported from: packages/provider-utils/src/load-setting.ts
package providerutils

import (
	"fmt"
	"os"
)

// LoadSettingOptions are the options for LoadSetting.
type LoadSettingOptions struct {
	// SettingValue is the direct setting value.
	SettingValue *string
	// EnvironmentVariableName is the name of the environment variable.
	EnvironmentVariableName string
	// SettingName is the name of the setting for error messages.
	SettingName string
	// Description is the description of the setting for error messages.
	Description string
}

// LoadSetting loads a required string setting from a parameter or environment variable.
// Returns an error if the setting cannot be found.
func LoadSetting(opts LoadSettingOptions) (string, error) {
	if opts.SettingValue != nil {
		return *opts.SettingValue, nil
	}

	val := os.Getenv(opts.EnvironmentVariableName)
	if val == "" {
		return "", fmt.Errorf(
			"%s setting is missing. Pass it using the '%s' parameter or the %s environment variable.",
			opts.Description, opts.SettingName, opts.EnvironmentVariableName,
		)
	}

	return val, nil
}
