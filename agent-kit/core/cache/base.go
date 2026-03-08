// Ported from: packages/core/src/cache/base.ts
package cache

// MastraServerCache defines the interface for server-side caching.
// Ported from the abstract class MastraServerCache in base.ts.
type MastraServerCache interface {
	Get(key string) (any, error)
	Set(key string, value any) error
	Delete(key string) error
	Clear() error
	ListPush(key string, value any) error
	ListLength(key string) (int, error)
	ListFromTo(key string, from int, to int) ([]any, error)
}
