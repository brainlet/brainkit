package brainkit_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestGeneratedWrappersDoNotExposeLowLevelReplyTopicHelpers(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`func\s+Publish[A-Z][A-Za-z0-9_]*\s*(?:\[|\()`),
		regexp.MustCompile(`func\s+Subscribe[A-Z][A-Za-z0-9_]*Resp\s*(?:\[|\()`),
	}

	var checked int
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", ".brainkit":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) != "typed_gen.go" {
			return nil
		}
		checked++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, pattern := range patterns {
			if match := pattern.Find(data); match != nil {
				t.Fatalf("%s exposes low-level reply-topic helper %q", path, match)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk typed_gen.go: %v", err)
	}
	if checked == 0 {
		t.Fatal("no generated typed wrappers were checked")
	}
}
