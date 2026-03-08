// Ported from: packages/core/src/workspace/lsp/language.ts
package lsp

import (
	"path/filepath"
	"strings"
)

// =============================================================================
// Language Detection
// =============================================================================

// LanguageExtensions maps file extensions (including the dot) to LSP language identifiers.
var LanguageExtensions = map[string]string{
	// TypeScript/JavaScript
	".ts":  "typescript",
	".tsx": "typescriptreact",
	".js":  "javascript",
	".jsx": "javascriptreact",
	".mjs": "javascript",
	".cjs": "javascript",

	// Python
	".py":  "python",
	".pyi": "python",

	// Go
	".go": "go",

	// Rust
	".rs": "rust",

	// C/C++
	".c":   "c",
	".cpp": "cpp",
	".cc":  "cpp",
	".cxx": "cpp",
	".h":   "c",
	".hpp": "cpp",

	// Java
	".java": "java",

	// JSON
	".json":  "json",
	".jsonc": "jsonc",

	// YAML
	".yaml": "yaml",
	".yml":  "yaml",

	// Markdown
	".md": "markdown",

	// HTML/CSS
	".html": "html",
	".css":  "css",
	".scss": "scss",
	".sass": "sass",
	".less": "less",
}

// GetLanguageId returns the LSP language ID for a file path based on its extension.
// Returns empty string if the extension is not recognized.
func GetLanguageId(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return ""
	}
	ext = strings.ToLower(ext)
	if lang, ok := LanguageExtensions[ext]; ok {
		return lang
	}
	return ""
}
