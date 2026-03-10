// Port of: assemblyscript/tests/tokenizer.js
// Tokenizes source files and verifies the tokenizer completes without panicking.
//
// The original tokenizer.js is a diagnostic script that tokenizes a single file
// and prints token info. This Go port runs the tokenizer against all std library
// files as a smoke test.
package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

func stdRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "std", "assembly")
}

// collectTokenizerSources walks std/assembly/ for all .ts files.
func collectTokenizerSources(t *testing.T) []string {
	t.Helper()
	root := stdRoot()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".ts" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Skipf("std sources not found at %s: %v", root, err)
	}
	sort.Strings(files)
	return files
}

// maxTokensPerFile is a safety limit to detect infinite loops in the tokenizer.
// No single std library file should produce more than this many tokens.
const maxTokensPerFile = 500_000

// TestTokenizer_NoPanic verifies the tokenizer can process all std library files without panicking.
// Ported from: assemblyscript/tests/tokenizer.js
func TestTokenizer_NoPanic(t *testing.T) {
	files := collectTokenizerSources(t)
	if len(files) == 0 {
		t.Skip("no std source files found")
	}

	passed := 0
	failed := 0

	for _, file := range files {
		rel, _ := filepath.Rel(stdRoot(), file)
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}

			panicked := false
			timedOut := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
						buf := make([]byte, 4096)
						n := runtime.Stack(buf, false)
						t.Logf("PANIC: %v\n%s", r, buf[:n])
					}
				}()

				source := ast.NewSource(ast.SourceKindUserEntry, rel, string(data))
				tn := tokenizer.NewTokenizer(source, nil)
				count := 0
				for {
					token := tn.Next(tokenizer.IdentifierHandlingPrefer)
					if token == tokenizer.TokenEndOfFile {
						break
					}
					// Must consume token values to advance position past the token.
					// Mirrors: assemblyscript/tests/tokenizer.js (lines 17-24)
					switch token {
					case tokenizer.TokenIdentifier:
						tn.ReadIdentifier()
					case tokenizer.TokenStringLiteral:
						tn.ReadString(int32('"'), false)
					case tokenizer.TokenIntegerLiteral:
						tn.ReadInteger()
					case tokenizer.TokenFloatLiteral:
						tn.ReadFloat()
					case tokenizer.TokenTemplateLiteral:
						tn.ReadString(int32('`'), false)
					}
					count++
					if count > maxTokensPerFile {
						timedOut = true
						return
					}
				}
			}()

			if panicked {
				failed++
				t.Errorf("tokenizer panicked on %s", rel)
			} else if timedOut {
				failed++
				t.Errorf("tokenizer exceeded %d tokens on %s (likely infinite loop)", maxTokensPerFile, rel)
			} else {
				passed++
			}
		})
	}

	t.Logf("Results: %d passed, %d failed (of %d total)", passed, failed, len(files))
}

// TestTokenizer_ParserFixtures runs the tokenizer against all parser fixture files.
func TestTokenizer_ParserFixtures(t *testing.T) {
	root := parserFixturesRoot()
	fixtures := discoverParserFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no parser fixtures found")
	}

	passed := 0
	failed := 0

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}

			panicked := false
			timedOut := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
						buf := make([]byte, 4096)
						n := runtime.Stack(buf, false)
						t.Logf("PANIC: %v\n%s", r, buf[:n])
					}
				}()

				source := ast.NewSource(ast.SourceKindUserEntry, name, string(data))
				tn := tokenizer.NewTokenizer(source, nil)
				count := 0
				for {
					token := tn.Next(tokenizer.IdentifierHandlingPrefer)
					if token == tokenizer.TokenEndOfFile {
						break
					}
					switch token {
					case tokenizer.TokenIdentifier:
						tn.ReadIdentifier()
					case tokenizer.TokenStringLiteral:
						tn.ReadString(int32('"'), false)
					case tokenizer.TokenIntegerLiteral:
						tn.ReadInteger()
					case tokenizer.TokenFloatLiteral:
						tn.ReadFloat()
					case tokenizer.TokenTemplateLiteral:
						tn.ReadString(int32('`'), false)
					}
					count++
					if count > maxTokensPerFile {
						timedOut = true
						return
					}
				}
			}()

			if panicked {
				failed++
				t.Errorf("tokenizer panicked on %s", name)
			} else if timedOut {
				failed++
				t.Errorf("tokenizer exceeded %d tokens on %s (likely infinite loop)", maxTokensPerFile, name)
			} else {
				passed++
			}
		})
	}

	t.Logf("Results: %d passed, %d failed (of %d total)", passed, failed, len(fixtures))
}

func init() {
	// suppress unused import
	_ = fmt.Sprintf
}
