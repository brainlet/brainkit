// Ported from: packages/core/src/cache/inmemory.test.ts
package cache

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// helper to check deep equality and fail with a clear message.
func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func TestInMemoryServerCache(t *testing.T) {
	// ---------------------------------------------------------------
	// Basic Operations
	// ---------------------------------------------------------------
	t.Run("Basic Operations", func(t *testing.T) {
		t.Run("get/set", func(t *testing.T) {
			t.Run("should store and retrieve a string value", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("key1", "value1")
				result, err := cache.Get("key1")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, "value1")
			})

			t.Run("should store and retrieve a number value", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("key2", 42)
				result, err := cache.Get("key2")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, 42)
			})

			t.Run("should store and retrieve an object value", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				obj := map[string]any{"name": "test", "age": 30}
				_ = cache.Set("key3", obj)
				result, err := cache.Get("key3")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, obj)
			})

			t.Run("should store and retrieve an array value", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				arr := []any{1, 2, 3, "test"}
				_ = cache.Set("key4", arr)
				result, err := cache.Get("key4")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, arr)
			})

			t.Run("should return nil for non-existent keys", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				result, err := cache.Get("nonexistent")
				if err != nil {
					t.Fatal(err)
				}
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			})

			t.Run("should overwrite existing values", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("key5", "original")
				_ = cache.Set("key5", "updated")
				result, err := cache.Get("key5")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, "updated")
			})
		})

		t.Run("delete", func(t *testing.T) {
			t.Run("should delete an existing key", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("deleteMe", "value")
				result, _ := cache.Get("deleteMe")
				assertEqual(t, result, "value")

				_ = cache.Delete("deleteMe")
				result, _ = cache.Get("deleteMe")
				if result != nil {
					t.Errorf("expected nil after delete, got %v", result)
				}
			})

			t.Run("should not error when deleting non-existent key", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				err := cache.Delete("nonexistent")
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			})
		})

		t.Run("clear", func(t *testing.T) {
			t.Run("should clear all cached values", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("key1", "value1")
				_ = cache.Set("key2", "value2")
				_ = cache.Set("key3", []any{1, 2, 3})

				r1, _ := cache.Get("key1")
				assertEqual(t, r1, "value1")
				r2, _ := cache.Get("key2")
				assertEqual(t, r2, "value2")
				r3, _ := cache.Get("key3")
				assertEqual(t, r3, []any{1, 2, 3})

				_ = cache.Clear()

				r1, _ = cache.Get("key1")
				if r1 != nil {
					t.Errorf("key1 should be nil after clear, got %v", r1)
				}
				r2, _ = cache.Get("key2")
				if r2 != nil {
					t.Errorf("key2 should be nil after clear, got %v", r2)
				}
				r3, _ = cache.Get("key3")
				if r3 != nil {
					t.Errorf("key3 should be nil after clear, got %v", r3)
				}
			})
		})
	})

	// ---------------------------------------------------------------
	// List Operations
	// ---------------------------------------------------------------
	t.Run("List Operations", func(t *testing.T) {
		t.Run("listPush", func(t *testing.T) {
			t.Run("should create a new list when key does not exist", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.ListPush("newList", "item1")
				result, _ := cache.Get("newList")
				assertEqual(t, result, []any{"item1"})
			})

			t.Run("should append to existing list", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("existingList", []any{"item1", "item2"})
				_ = cache.ListPush("existingList", "item3")
				result, _ := cache.Get("existingList")
				assertEqual(t, result, []any{"item1", "item2", "item3"})
			})

			t.Run("should handle different data types in list", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.ListPush("mixedList", "string")
				_ = cache.ListPush("mixedList", 42)
				_ = cache.ListPush("mixedList", map[string]any{"key": "value"})
				_ = cache.ListPush("mixedList", []any{1, 2, 3})

				result, _ := cache.Get("mixedList")
				assertEqual(t, result, []any{
					"string",
					42,
					map[string]any{"key": "value"},
					[]any{1, 2, 3},
				})
			})

			t.Run("should create new list when existing value is not an array", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("notAnArray", "string value")
				_ = cache.ListPush("notAnArray", "newItem")
				result, _ := cache.Get("notAnArray")
				assertEqual(t, result, []any{"newItem"})
			})
		})

		t.Run("listLength", func(t *testing.T) {
			t.Run("should return length of existing list", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("testList", []any{"a", "b", "c"})
				length, err := cache.ListLength("testList")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, length, 3)
			})

			t.Run("should return 0 for empty list", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("emptyList", []any{})
				length, err := cache.ListLength("emptyList")
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, length, 0)
			})

			t.Run("should return error when key contains non-array value", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_ = cache.Set("notAnArray", "string value")
				_, err := cache.ListLength("notAnArray")
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "notAnArray is not an array") {
					t.Errorf("expected error containing 'notAnArray is not an array', got %v", err)
				}
			})

			t.Run("should return error when key does not exist", func(t *testing.T) {
				cache := NewInMemoryServerCache()
				_, err := cache.ListLength("nonexistent")
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "nonexistent is not an array") {
					t.Errorf("expected error containing 'nonexistent is not an array', got %v", err)
				}
			})
		})

		t.Run("listFromTo", func(t *testing.T) {
			setup := func() *InMemoryServerCache {
				cache := NewInMemoryServerCache()
				_ = cache.Set("testList", []any{"a", "b", "c", "d", "e"})
				return cache
			}

			t.Run("should return slice from start to end (inclusive)", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("testList", 1, 3)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"b", "c", "d"})
			})

			t.Run("should return slice from start to end of array when to is -1", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("testList", 2, -1)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"c", "d", "e"})
			})

			t.Run("should return slice from start to end of array when to is not provided (use -1)", func(t *testing.T) {
				// In TS, `to` defaults to -1 when not provided.
				// In Go, we pass -1 explicitly.
				cache := setup()
				result, err := cache.ListFromTo("testList", 2, -1)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"c", "d", "e"})
			})

			t.Run("should return full array when from is 0 and to is -1", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("testList", 0, -1)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"a", "b", "c", "d", "e"})
			})

			t.Run("should return empty array when from is greater than array length", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("testList", 10, 15)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{})
			})

			t.Run("should return empty array when key does not exist", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("nonexistent", 0, 2)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{})
			})

			t.Run("should return empty array when key contains non-array value", func(t *testing.T) {
				cache := setup()
				_ = cache.Set("notAnArray", "string value")
				result, err := cache.ListFromTo("notAnArray", 0, 2)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{})
			})

			t.Run("should handle negative from index", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("testList", -2, -1)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"d", "e"})
			})

			t.Run("should return inclusive range when from and to are consecutive", func(t *testing.T) {
				cache := setup()
				result, err := cache.ListFromTo("testList", 1, 2)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"b", "c"})
			})

			t.Run("should behave like Redis LRANGE with inclusive end index", func(t *testing.T) {
				cache := setup()
				// Redis LRANGE includes both start and end indices.
				result, err := cache.ListFromTo("testList", 0, 4)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, []any{"a", "b", "c", "d", "e"})

				singleItem, err := cache.ListFromTo("testList", 2, 2)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, singleItem, []any{"c"})
			})
		})
	})

	// ---------------------------------------------------------------
	// Complex Scenarios
	// ---------------------------------------------------------------
	t.Run("Complex Scenarios", func(t *testing.T) {
		t.Run("should handle multiple concurrent operations", func(t *testing.T) {
			cache := NewInMemoryServerCache()

			// Concurrent sets using goroutines + WaitGroup (mirrors Promise.all).
			var wg sync.WaitGroup
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					_ = cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
				}(i)
			}
			wg.Wait()

			// Verify all values are set.
			for i := 0; i < 10; i++ {
				result, err := cache.Get(fmt.Sprintf("key%d", i))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, fmt.Sprintf("value%d", i))
			}
		})

		t.Run("should handle mixed operations on same key", func(t *testing.T) {
			cache := NewInMemoryServerCache()

			// Start with a regular value.
			_ = cache.Set("mixedKey", "initial")
			r, _ := cache.Get("mixedKey")
			assertEqual(t, r, "initial")

			// Convert to list by pushing.
			_ = cache.ListPush("mixedKey", "listItem")
			r, _ = cache.Get("mixedKey")
			assertEqual(t, r, []any{"listItem"})

			// Add more items.
			_ = cache.ListPush("mixedKey", "anotherItem")
			length, err := cache.ListLength("mixedKey")
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, length, 2)

			// Get slice (Redis-like inclusive).
			slice, err := cache.ListFromTo("mixedKey", 0, 1)
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, slice, []any{"listItem", "anotherItem"})

			// Replace with regular value again.
			_ = cache.Set("mixedKey", "replaced")
			r, _ = cache.Get("mixedKey")
			assertEqual(t, r, "replaced")
		})

		t.Run("should maintain data integrity after operations", func(t *testing.T) {
			cache := NewInMemoryServerCache()

			// Set up initial data.
			_ = cache.Set("string", "test")
			_ = cache.Set("number", 123)
			_ = cache.Set("object", map[string]any{"key": "value"})
			_ = cache.Set("list", []any{"a", "b"})

			// Perform operations.
			_ = cache.ListPush("list", "c")
			_ = cache.Set("string", "updated")

			// Verify integrity.
			r, _ := cache.Get("string")
			assertEqual(t, r, "updated")
			r, _ = cache.Get("number")
			assertEqual(t, r, 123)
			r, _ = cache.Get("object")
			assertEqual(t, r, map[string]any{"key": "value"})
			r, _ = cache.Get("list")
			assertEqual(t, r, []any{"a", "b", "c"})
			length, err := cache.ListLength("list")
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, length, 3)
		})
	})

	// ---------------------------------------------------------------
	// Edge Cases
	// ---------------------------------------------------------------
	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("should handle nil values", func(t *testing.T) {
			// In TS: both null and undefined map to nil in Go.
			cache := NewInMemoryServerCache()
			_ = cache.Set("nilValue", nil)

			result, err := cache.Get("nilValue")
			if err != nil {
				t.Fatal(err)
			}
			// nil stored is distinguishable from "key not found" because
			// the entry exists. Both return nil in Go, matching TS behavior
			// where get returns the stored value.
			if result != nil {
				t.Errorf("expected nil, got %v", result)
			}
		})

		t.Run("should handle empty strings and empty objects", func(t *testing.T) {
			cache := NewInMemoryServerCache()
			_ = cache.Set("emptyString", "")
			_ = cache.Set("emptyObject", map[string]any{})
			_ = cache.Set("emptyArray", []any{})

			r, _ := cache.Get("emptyString")
			assertEqual(t, r, "")
			r, _ = cache.Get("emptyObject")
			assertEqual(t, r, map[string]any{})
			r, _ = cache.Get("emptyArray")
			assertEqual(t, r, []any{})
		})

		t.Run("should handle special characters in keys", func(t *testing.T) {
			cache := NewInMemoryServerCache()
			specialKeys := []string{
				"key with spaces",
				"key-with-dashes",
				"key_with_underscores",
				"key.with.dots",
			}

			for _, key := range specialKeys {
				_ = cache.Set(key, fmt.Sprintf("value for %s", key))
				result, err := cache.Get(key)
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, result, fmt.Sprintf("value for %s", key))
			}
		})

		t.Run("should handle very long keys and values", func(t *testing.T) {
			cache := NewInMemoryServerCache()
			longKey := strings.Repeat("a", 1000)
			longValue := strings.Repeat("b", 10000)

			_ = cache.Set(longKey, longValue)
			result, err := cache.Get(longKey)
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, result, longValue)
		})

		t.Run("should handle complex nested objects", func(t *testing.T) {
			cache := NewInMemoryServerCache()
			complexObject := map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"array":  []any{1, 2, map[string]any{"nested": true}},
							"string": "deep value",
							"number": 42,
						},
					},
				},
			}

			_ = cache.Set("complex", complexObject)
			result, err := cache.Get("complex")
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, result, complexObject)
		})
	})

	// ---------------------------------------------------------------
	// Performance and Limits
	// ---------------------------------------------------------------
	t.Run("Performance and Limits", func(t *testing.T) {
		t.Run("should handle large number of keys", func(t *testing.T) {
			cache := NewInMemoryServerCache()
			keyCount := 100

			// Concurrent sets (mirrors Promise.all).
			var wg sync.WaitGroup
			for i := 0; i < keyCount; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					_ = cache.Set(fmt.Sprintf("bulk%d", i), fmt.Sprintf("value%d", i))
				}(i)
			}
			wg.Wait()

			// Verify a sample of keys.
			r, _ := cache.Get("bulk0")
			assertEqual(t, r, "value0")
			r, _ = cache.Get("bulk50")
			assertEqual(t, r, "value50")
			r, _ = cache.Get("bulk99")
			assertEqual(t, r, "value99")
		})

		t.Run("should handle large lists", func(t *testing.T) {
			cache := NewInMemoryServerCache()
			itemCount := 1000

			for i := 0; i < itemCount; i++ {
				_ = cache.ListPush("largeList", fmt.Sprintf("item%d", i))
			}

			length, err := cache.ListLength("largeList")
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, length, itemCount)

			firstItems, err := cache.ListFromTo("largeList", 0, 5)
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, firstItems, []any{"item0", "item1", "item2", "item3", "item4", "item5"})

			// to=-1 means "to end of array".
			lastItems, err := cache.ListFromTo("largeList", itemCount-5, -1)
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, lastItems, []any{"item995", "item996", "item997", "item998", "item999"})
		})
	})
}
