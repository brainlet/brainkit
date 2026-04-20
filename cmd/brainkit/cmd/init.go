package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// brainkitYAMLTemplate is the starter config written by `brainkit
// init`. Comments intentionally duplicate server/testdata/example.yaml
// so the file stays self-documenting even without internet access.
const brainkitYAMLTemplate = `# brainkit server config.
# Environment variables referenced as $NAME or ${NAME} are substituted
# at load time; missing variables expand to empty strings.

namespace: %s
fs_root: ./data

transport:
  type: embedded

gateway:
  listen: :8080

# Providers, storages, vectors, plugins, audit, tracing, probes —
# uncomment and populate as needed.
#
# providers:
#   - name: openai
#     type: openai
#     api_key: $OPENAI_API_KEY
#
# storages:
#   default:
#     type: sqlite
#     path: ./data/default.db
#
# vectors:
#   default:
#     type: sqlite
#     path: ./data/default.db
#
# packages:
#   - ./packages/hello
`

const brainkitGitignoreBlock = `# brainkit runtime state
/data
*.db
/nats-data
`

// newInitCmd scaffolds a brainkit runtime in the current working
// directory — writes brainkit.yaml + ./data/ (+ appends to .gitignore
// if one exists). No Go file, no go.mod: this CLI drives the shipped
// brainkit binary, not a user-owned Go project.
func newInitCmd() *cobra.Command {
	var force bool
	var namespace string

	c := &cobra.Command{
		Use:   "init",
		Short: "Initialize a brainkit runtime in the current directory",
		Long: `Init writes a starter brainkit.yaml and a ./data/ directory into
the current working directory. Run "brainkit start" afterwards to
boot the server against that config.

Unlike a Go project scaffolder, init does NOT create main.go or
go.mod — the brainkit binary itself is the runtime. The config file
is all you need.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolve cwd: %w", err)
			}

			if namespace == "" {
				namespace = filepath.Base(cwd)
			}

			yamlPath := filepath.Join(cwd, "brainkit.yaml")
			dataDir := filepath.Join(cwd, "data")

			if _, err := os.Stat(yamlPath); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", yamlPath)
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("stat %s: %w", yamlPath, err)
			}

			yaml := fmt.Sprintf(brainkitYAMLTemplate, namespace)
			if err := os.WriteFile(yamlPath, []byte(yaml), 0644); err != nil {
				return fmt.Errorf("write brainkit.yaml: %w", err)
			}
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("create data directory: %w", err)
			}

			gitignoreTouched, err := ensureGitignore(cwd)
			if err != nil {
				cmd.Printf("warning: .gitignore update skipped: %v\n", err)
			}

			cmd.Printf("Initialized brainkit runtime in %s\n", cwd)
			cmd.Println("  brainkit.yaml")
			cmd.Println("  data/")
			if gitignoreTouched {
				cmd.Println("  .gitignore   (appended brainkit entries)")
			}
			cmd.Println()
			cmd.Println("Next: brainkit start")
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite an existing brainkit.yaml")
	c.Flags().StringVar(&namespace, "namespace", "", "namespace for the runtime (default: current directory name)")
	return c
}

// ensureGitignore appends the brainkit-specific ignore block to .gitignore
// in dir when the block isn't already present. Returns true when the
// file was modified or created. Does nothing when there's no git tree
// detectable above the directory (either CWD isn't under git or the user
// explicitly didn't want a .gitignore).
func ensureGitignore(dir string) (bool, error) {
	path := filepath.Join(dir, ".gitignore")

	// Only manage .gitignore when we can see either an existing one
	// or a .git directory — don't invent one in a bare folder.
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if _, gitErr := os.Stat(filepath.Join(dir, ".git")); errors.Is(gitErr, os.ErrNotExist) {
			return false, nil
		}
		if err := os.WriteFile(path, []byte(brainkitGitignoreBlock), 0644); err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if containsBrainkitBlock(existing) {
		return false, nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}
	if _, err := f.WriteString(prefix + "\n" + brainkitGitignoreBlock); err != nil {
		return false, err
	}
	return true, nil
}

func containsBrainkitBlock(b []byte) bool {
	needle := []byte("# brainkit runtime state")
	for i := 0; i+len(needle) <= len(b); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if b[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
