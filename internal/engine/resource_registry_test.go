package engine

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceRegistry_RegisterAndGet(t *testing.T) {
	reg := NewResourceRegistry()

	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo", Source: "agent.ts", CreatedAt: time.Now()})

	entry, ok := reg.Get("tool", "echo")
	require.True(t, ok)
	assert.Equal(t, "tool", entry.Type)
	assert.Equal(t, "echo", entry.ID)
	assert.Equal(t, "agent.ts", entry.Source)

	_, ok = reg.Get("tool", "nonexistent")
	assert.False(t, ok)
}

func TestResourceRegistry_RegisterReplaces(t *testing.T) {
	reg := NewResourceRegistry()

	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo-v1", Source: "v1.ts"})
	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo-v2", Source: "v2.ts"})

	entry, ok := reg.Get("tool", "echo")
	require.True(t, ok)
	assert.Equal(t, "echo-v2", entry.Name)
	assert.Equal(t, "v2.ts", entry.Source)
	assert.Equal(t, 1, reg.Len())
}

func TestResourceRegistry_Unregister(t *testing.T) {
	reg := NewResourceRegistry()
	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo", Source: "a.ts"})

	entry, ok := reg.Unregister("tool", "echo")
	require.True(t, ok)
	assert.Equal(t, "echo", entry.ID)
	assert.Equal(t, 0, reg.Len())

	_, ok = reg.Unregister("tool", "echo")
	assert.False(t, ok)
}

func TestResourceRegistry_List(t *testing.T) {
	reg := NewResourceRegistry()
	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "tool", ID: "add", Name: "add", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "agent", ID: "bot", Name: "bot", Source: "b.ts"})
	reg.Register(ResourceEntry{Type: "workflow", ID: "flow", Name: "flow", Source: "b.ts"})

	all := reg.List("")
	assert.Len(t, all, 4)

	tools := reg.List("tool")
	assert.Len(t, tools, 2)

	agents := reg.List("agent")
	assert.Len(t, agents, 1)
	assert.Equal(t, "bot", agents[0].ID)

	empty := reg.List("nonexistent")
	assert.Len(t, empty, 0)
}

func TestResourceRegistry_ListBySource(t *testing.T) {
	reg := NewResourceRegistry()
	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "agent", ID: "bot", Name: "bot", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "tool", ID: "add", Name: "add", Source: "b.ts"})

	aResources := reg.ListBySource("a.ts")
	assert.Len(t, aResources, 2)

	bResources := reg.ListBySource("b.ts")
	assert.Len(t, bResources, 1)
	assert.Equal(t, "add", bResources[0].ID)

	noResources := reg.ListBySource("c.ts")
	assert.Len(t, noResources, 0)
}

func TestResourceRegistry_RemoveBySource(t *testing.T) {
	reg := NewResourceRegistry()
	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "echo", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "agent", ID: "bot", Name: "bot", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "tool", ID: "add", Name: "add", Source: "b.ts"})

	removed := reg.RemoveBySource("a.ts")
	assert.Len(t, removed, 2)
	assert.Equal(t, 1, reg.Len())

	// b.ts entries untouched
	entry, ok := reg.Get("tool", "add")
	require.True(t, ok)
	assert.Equal(t, "b.ts", entry.Source)

	// a.ts entries gone
	_, ok = reg.Get("tool", "echo")
	assert.False(t, ok)
	_, ok = reg.Get("agent", "bot")
	assert.False(t, ok)

	// Second call returns empty
	removed2 := reg.RemoveBySource("a.ts")
	assert.Len(t, removed2, 0)
}

func TestResourceRegistry_RemoveBySourceAtomicity(t *testing.T) {
	// Verify that concurrent readers don't see partial removal state.
	reg := NewResourceRegistry()
	for i := 0; i < 100; i++ {
		reg.Register(ResourceEntry{Type: "tool", ID: fmt.Sprintf("tool-%d", i), Source: "bulk.ts"})
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Reader goroutines check that source count is either 100 (pre-removal) or 0 (post-removal)
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					entries := reg.ListBySource("bulk.ts")
					count := len(entries)
					if count != 100 && count != 0 {
						t.Errorf("saw partial removal: %d entries (expected 100 or 0)", count)
						return
					}
				}
			}
		}()
	}

	// Writer removes all at once
	removed := reg.RemoveBySource("bulk.ts")
	close(done)
	wg.Wait()

	assert.Len(t, removed, 100)
	assert.Equal(t, 0, reg.Len())
}

func TestResourceRegistry_ConcurrentRegisterUnregister(t *testing.T) {
	reg := NewResourceRegistry()
	var wg sync.WaitGroup

	// 50 goroutines registering, 50 unregistering, on overlapping keys
	for i := 0; i < 50; i++ {
		wg.Add(2)
		id := fmt.Sprintf("tool-%d", i)
		go func() {
			defer wg.Done()
			reg.Register(ResourceEntry{Type: "tool", ID: id, Source: "concurrent.ts"})
		}()
		go func() {
			defer wg.Done()
			reg.Unregister("tool", id)
		}()
	}

	wg.Wait()

	// No panic, no data race (run with -race). Final count is non-deterministic
	// but must be between 0 and 50.
	count := reg.Len()
	assert.True(t, count >= 0 && count <= 50, "count=%d should be 0-50", count)
}

func TestResourceRegistry_KeyIsolation(t *testing.T) {
	// Same ID in different types must not collide.
	reg := NewResourceRegistry()
	reg.Register(ResourceEntry{Type: "tool", ID: "echo", Name: "tool-echo", Source: "a.ts"})
	reg.Register(ResourceEntry{Type: "agent", ID: "echo", Name: "agent-echo", Source: "a.ts"})

	assert.Equal(t, 2, reg.Len())

	tool, ok := reg.Get("tool", "echo")
	require.True(t, ok)
	assert.Equal(t, "tool-echo", tool.Name)

	agent, ok := reg.Get("agent", "echo")
	require.True(t, ok)
	assert.Equal(t, "agent-echo", agent.Name)

	reg.Unregister("tool", "echo")
	assert.Equal(t, 1, reg.Len())
	_, ok = reg.Get("agent", "echo")
	assert.True(t, ok)
}
