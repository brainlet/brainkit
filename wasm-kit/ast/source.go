package ast

import (
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
)

// nativeSource is the singleton native source.
var nativeSource *Source

// NativeSource returns the special native source.
func NativeSource() *Source {
	if nativeSource == nil {
		nativeSource = NewSource(SourceKindLibraryEntry, common.LIBRARY_PREFIX+"native.ts", "[native code]")
	}
	return nativeSource
}

// Source is a top-level source node representing a parsed file.
// It implements both ast.Node and diagnostics.Source.
type Source struct {
	NodeBase
	SourceKind     SourceKind
	NormalizedPath string
	Text           string
	InternalPath   string
	SimplePath     string
	Statements     []Node // Statement elements
	DebugInfoIndex int32
	ExportPaths    []string

	lineCache  []int32
	lineColumn int32
}

// NewSource creates a new source node.
func NewSource(sourceKind SourceKind, normalizedPath string, text string) *Source {
	s := &Source{
		NodeBase:       NodeBase{Kind: NodeKindSource},
		SourceKind:     sourceKind,
		NormalizedPath: normalizedPath,
		Text:           text,
		DebugInfoIndex: -1,
		lineColumn:     1,
	}
	s.Range = diagnostics.Range{
		Start:  0,
		End:    int32(len(text)),
		Source: s,
	}
	internalPath := MangleInternalPath(normalizedPath)
	s.InternalPath = internalPath
	pos := strings.LastIndex(internalPath, common.PATH_DELIMITER)
	if pos >= 0 {
		s.SimplePath = internalPath[pos+1:]
	} else {
		s.SimplePath = internalPath
	}
	return s
}

// --- diagnostics.Source interface ---

// SourceText returns the full source text.
func (s *Source) SourceText() string { return s.Text }

// SourceNormalizedPath returns the normalized path.
func (s *Source) SourceNormalizedPath() string { return s.NormalizedPath }

// LineAt determines the line number at the specified position. Starts at 1.
func (s *Source) LineAt(pos int32) int32 {
	if pos < 0 || pos >= 0x7fffffff {
		panic("assertion failed: invalid position")
	}
	lineCache := s.lineCache
	if lineCache == nil {
		lineCache = []int32{0}
		text := s.Text
		for off := 0; off < len(text); off++ {
			if text[off] == '\n' {
				lineCache = append(lineCache, int32(off+1))
			}
		}
		lineCache = append(lineCache, 0x7fffffff)
		s.lineCache = lineCache
	}
	l := 0
	r := len(lineCache) - 1
	for l < r {
		m := l + ((r - l) >> 1)
		if pos < lineCache[m] {
			r = m
		} else if pos < lineCache[m+1] {
			s.lineColumn = pos - lineCache[m] + 1
			return int32(m + 1)
		} else {
			l = m + 1
		}
	}
	panic("assertion failed: unreachable in LineAt")
}

// ColumnAt gets the column number at the last position queried with LineAt. Starts at 1.
func (s *Source) ColumnAt() int32 {
	return s.lineColumn
}

// --- Source helpers ---

// IsNative checks if this source represents native code.
func (s *Source) IsNative() bool {
	return s == nativeSource
}

// IsLibrary checks if this source is part of the (standard) library.
func (s *Source) IsLibrary() bool {
	return s.SourceKind == SourceKindLibrary || s.SourceKind == SourceKindLibraryEntry
}
