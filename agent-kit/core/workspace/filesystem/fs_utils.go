// Ported from: packages/core/src/workspace/filesystem/fs-utils.ts
package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// Path Utilities
// =============================================================================

// ExpandTilde expands a leading "~" in a path to the user's home directory.
// Returns the path unchanged if it doesn't start with "~".
func ExpandTilde(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// ResolveWorkspacePath resolves a workspace-relative path against a base path.
// If the path is already absolute, it is returned as-is.
func ResolveWorkspacePath(basePath, workspacePath string) string {
	if filepath.IsAbs(workspacePath) {
		return workspacePath
	}
	return filepath.Join(basePath, workspacePath)
}

// =============================================================================
// File Existence & Stat
// =============================================================================

// FsExists checks if a path exists on the local filesystem.
func FsExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FsStat returns a FileStat for a local filesystem path.
// The displayPath parameter controls the Path field in the result.
func FsStat(path string, displayPath string) (*FileStat, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &fileNotFoundError{path: displayPath}
		}
		return nil, err
	}

	entryType := "file"
	if info.IsDir() {
		entryType = "directory"
	}

	mimeType := ""
	if !info.IsDir() {
		mimeType = GetMimeType(filepath.Base(path))
	}

	return &FileStat{
		Name:       info.Name(),
		Path:       displayPath,
		Type:       entryType,
		Size:       info.Size(),
		CreatedAt:  info.ModTime(), // Go doesn't expose creation time portably
		ModifiedAt: info.ModTime(),
		MimeType:   mimeType,
	}, nil
}

// fileNotFoundError is a simple error for file-not-found.
type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "File not found: " + e.path
}

// =============================================================================
// Text File Detection
// =============================================================================

// textExtensions is the set of file extensions considered to be text files.
var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".mdown": true,
	".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".mjs": true, ".cjs": true,
	".py": true, ".rb": true, ".go": true, ".rs": true, ".java": true,
	".c": true, ".cpp": true, ".cc": true, ".h": true, ".hpp": true,
	".cs": true, ".swift": true, ".kt": true, ".kts": true,
	".php": true, ".pl": true, ".pm": true, ".r": true,
	".scala": true, ".clj": true, ".cljs": true, ".ex": true, ".exs": true,
	".erl": true, ".hrl": true, ".hs": true, ".lua": true, ".tcl": true,
	".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".ps1": true, ".bat": true, ".cmd": true,
	".html": true, ".htm": true, ".xhtml": true,
	".css": true, ".scss": true, ".sass": true, ".less": true, ".styl": true,
	".json": true, ".jsonl": true, ".json5": true, ".jsonc": true,
	".xml": true, ".svg": true, ".xsl": true, ".xslt": true,
	".yaml": true, ".yml": true,
	".toml": true, ".ini": true, ".cfg": true, ".conf": true, ".properties": true,
	".env": true,
	".csv": true, ".tsv": true,
	".sql": true,
	".graphql": true, ".gql": true,
	".proto": true,
	".dockerfile": true,
	".gitignore": true, ".gitattributes": true, ".gitmodules": true,
	".editorconfig": true,
	".eslintrc": true, ".prettierrc": true, ".babelrc": true, ".stylelintrc": true,
	".npmrc": true, ".yarnrc": true,
	".tf": true, ".tfvars": true,
	".vue": true, ".svelte": true, ".astro": true,
	".mdx": true, ".rst": true, ".adoc": true, ".tex": true, ".latex": true,
	".lock": true,
	".log": true,
	".patch": true, ".diff": true,
	".makefile": true,
	".cmake": true,
	".gradle": true,
	".sbt": true,
	".cabal": true,
	".nix": true,
	".dhall": true,
	".pug": true, ".jade": true, ".ejs": true, ".hbs": true, ".mustache": true,
	".twig": true, ".jinja": true, ".j2": true,
	".prisma": true,
	".sol": true,
	".v": true, ".sv": true, ".vhd": true, ".vhdl": true,
	".zig": true, ".nim": true, ".d": true, ".dart": true,
	".ml": true, ".mli": true, ".fs": true, ".fsi": true, ".fsx": true,
	".lisp": true, ".cl": true, ".el": true, ".scm": true, ".rkt": true,
	".m": true, ".mm": true,
	".asm": true, ".s": true,
	".wasm": true, ".wat": true,
	".ipynb": true,
	".snap": true,
}

// IsTextFile checks if a file is likely a text file based on its extension.
// Files without an extension are also treated as text.
func IsTextFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return true // Files without extensions are treated as text
	}

	// Check known text extensions
	if textExtensions[ext] {
		return true
	}

	// Check well-known filenames without standard extensions
	base := strings.ToLower(filepath.Base(filename))
	switch base {
	case "makefile", "dockerfile", "vagrantfile", "gemfile", "rakefile",
		"procfile", "brewfile", "justfile", "taskfile",
		"license", "licence", "readme", "changelog", "contributing",
		"authors", "contributors", "copying", "notice",
		"codeowners", "cname":
		return true
	}

	return false
}

// =============================================================================
// MIME Type Detection
// =============================================================================

// mimeTypes maps file extensions to MIME types.
var mimeTypes = map[string]string{
	// Text
	".txt":  "text/plain",
	".md":   "text/markdown",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".csv":  "text/csv",
	".xml":  "text/xml",
	".svg":  "image/svg+xml",
	".yaml": "text/yaml",
	".yml":  "text/yaml",
	".json": "application/json",
	".toml": "application/toml",

	// JavaScript/TypeScript
	".js":  "text/javascript",
	".jsx": "text/javascript",
	".ts":  "text/typescript",
	".tsx": "text/typescript",
	".mjs": "text/javascript",
	".cjs": "text/javascript",

	// Images
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".avif": "image/avif",

	// Audio
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".m4a":  "audio/mp4",
	".wma":  "audio/x-ms-wma",

	// Video
	".mp4":  "video/mp4",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".webm": "video/webm",
	".mkv":  "video/x-matroska",

	// Archives
	".zip":  "application/zip",
	".tar":  "application/x-tar",
	".gz":   "application/gzip",
	".bz2":  "application/x-bzip2",
	".xz":   "application/x-xz",
	".7z":   "application/x-7z-compressed",
	".rar":  "application/vnd.rar",

	// Documents
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

	// Fonts
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".eot":   "application/vnd.ms-fontobject",

	// Other
	".wasm": "application/wasm",
}

// GetMimeType returns the MIME type for a filename based on its extension.
// Returns "application/octet-stream" if the type is unknown.
func GetMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// =============================================================================
// ModTime Cache (for FileReadTracker)
// =============================================================================

// GetModTime returns the modification time of a file.
// Returns zero time if the file doesn't exist.
func GetModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
