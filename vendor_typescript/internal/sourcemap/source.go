package sourcemap

import "github.com/brainlet/brainkit/vendor_typescript/internal/core"

type Source interface {
	Text() string
	FileName() string
	ECMALineMap() []core.TextPos
}
