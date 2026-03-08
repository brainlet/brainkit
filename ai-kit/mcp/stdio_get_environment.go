// Ported from: packages/mcp/src/tool/mcp-stdio/get-environment.ts
package mcp

import (
	"os"
	"runtime"
	"strings"
)

// GetEnvironment constructs the environment variables for the child process.
// It merges custom environment variables with a set of default inherited
// environment variables from the current process.
func GetEnvironment(customEnv map[string]string) map[string]string {
	var defaultInheritedEnvVars []string

	if runtime.GOOS == "windows" {
		defaultInheritedEnvVars = []string{
			"APPDATA",
			"HOMEDRIVE",
			"HOMEPATH",
			"LOCALAPPDATA",
			"PATH",
			"PROCESSOR_ARCHITECTURE",
			"SYSTEMDRIVE",
			"SYSTEMROOT",
			"TEMP",
			"USERNAME",
			"USERPROFILE",
		}
	} else {
		defaultInheritedEnvVars = []string{
			"HOME",
			"LOGNAME",
			"PATH",
			"SHELL",
			"TERM",
			"USER",
		}
	}

	env := make(map[string]string)
	if customEnv != nil {
		for k, v := range customEnv {
			env[k] = v
		}
	}

	for _, key := range defaultInheritedEnvVars {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		// Skip shell functions (values starting with "()")
		if strings.HasPrefix(value, "()") {
			continue
		}

		env[key] = value
	}

	return env
}
