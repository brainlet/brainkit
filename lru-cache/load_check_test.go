package lrucache

// Tests ported from node-lru-cache test/load-check.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/load-check.ts
//
// This is an intensive load/stress verification test. It fills the cache with
// random data, performs many get/set cycles, and periodically verifies that the
// internal keyMap is consistent with keyList and valList.
//
// Uses helpers from helpers_test.go: assertEqual, exposeKeyMap, exposeValList

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

// randomHex returns a random hex string of 24 characters (12 random bytes).
// TS source: test/load-check.ts line 10 — crypto.randomBytes(12).toString('hex')
func randomHex() string {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// getVal returns a slice of 4 random hex strings.
// TS source: test/load-check.ts lines 10-15
// const getVal = () => [
//
//	crypto.randomBytes(12).toString('hex'),
//	crypto.randomBytes(12).toString('hex'),
//	crypto.randomBytes(12).toString('hex'),
//	crypto.randomBytes(12).toString('hex'),
//
// ]
func getVal() []string {
	return []string{randomHex(), randomHex(), randomHex(), randomHex()}
}

// cryptoRandIntn returns a random integer in [0, n) using crypto/rand.
func cryptoRandIntn(n int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(fmt.Sprintf("crypto/rand.Int failed: %v", err))
	}
	return int(nBig.Int64())
}

// ---------------------------------------------------------------------------
// test/load-check.ts — Intensive load verification (lines 1-59)
// ---------------------------------------------------------------------------

func TestLoadCheck(t *testing.T) {
	// TS source: test/load-check.ts lines 1-59
	// process.env.TAP_BAIL = '1'
	// This test bails on first failure (equivalent to t.Fatal in Go).

	// TS source: line 7 — const max = 10000
	const maxItems = 10000

	// TS source: line 8 — const cache = new LRU<string, number[]>({ max })
	// In Go, we use []string as the value type (since getVal returns []string).
	cache := New[string, []string](Options[string, []string]{Max: maxItems})

	// TS source: lines 17-22
	// const seeds = new Array(max * 3)
	// for (let i = 0; i < max * 3; i++) {
	//   const v = getVal()
	//   seeds[i] = [v.join(':'), v]
	// }
	type seed struct {
		key string
		val []string
	}
	seedCount := maxItems * 3
	seeds := make([]seed, seedCount)
	for i := 0; i < seedCount; i++ {
		v := getVal()
		seeds[i] = seed{
			key: strings.Join(v, ":"),
			val: v,
		}
	}
	// TS source: line 23 — t.pass('generated seed data')
	t.Log("generated seed data")

	// TS source: lines 25-38
	// const verifyCache = () => {
	//   const e = expose(cache)
	//   for (const [k, i] of e.keyMap.entries()) {
	//     const v = e.valList[i] as number[]
	//     const key = e.keyList[i]
	//     if (k !== key) { t.equal(k, key, ...) }
	//     if (v.join(':') !== k) { t.equal(k, v.join(':'), ...) }
	//   }
	// }
	verifyCache := func() {
		keyMap := exposeKeyMap(cache)
		valList := exposeValList(cache)
		for k, i := range keyMap {
			// Verify keyList[i] == k
			if cache.keyList[i] == nil {
				t.Fatalf("verifyCache: keyList[%d] is nil for key %q", i, k)
			}
			actualKey := *cache.keyList[i]
			if k != actualKey {
				t.Fatalf("verifyCache: keyMap key %q != keyList[%d] key %q", k, i, actualKey)
			}
			// Verify valList[i].join(':') == k
			if valList[i] == nil {
				t.Fatalf("verifyCache: valList[%d] is nil for key %q", i, k)
			}
			joinedVal := strings.Join(*valList[i], ":")
			if joinedVal != k {
				t.Fatalf("verifyCache: key %q != joined value %q at index %d", k, joinedVal, i)
			}
		}
	}

	// TS source: lines 41-58
	// let cycles = 0
	// const cycleLength = Math.floor(max / 100)
	// while (cycles < max * 5) {
	//   const r = Math.floor(Math.random() * seeds.length)
	//   const seed = seeds[r]
	//   const v = cache.get(seed[0])
	//   if (v === undefined) {
	//     cache.set(seed[0], seed[1])
	//   } else {
	//     t.equal(v.join(':'), seed[0], 'correct get ' + cycles, { seed, v })
	//   }
	//   if (++cycles % cycleLength === 0) {
	//     verifyCache()
	//     t.pass('cycle check ' + cycles)
	//   }
	// }
	cycles := 0
	cycleLength := maxItems / 100 // 100

	totalCycles := maxItems * 5 // 50000
	for cycles < totalCycles {
		r := cryptoRandIntn(len(seeds))
		s := seeds[r]

		v, ok := cache.Get(s.key)
		if !ok {
			// TS source: line 48 — cache.set(seed[0], seed[1])
			cache.Set(s.key, s.val)
		} else {
			// TS source: lines 50-53 — verify retrieved value matches key
			joinedV := strings.Join(v, ":")
			if joinedV != s.key {
				t.Fatalf("cycle %d: expected get(%q) to return value joining to key, got %q",
					cycles, s.key, joinedV)
			}
		}

		cycles++
		if cycles%cycleLength == 0 {
			verifyCache()
			// TS source: line 57 — t.pass('cycle check ' + cycles)
			t.Logf("cycle check %d", cycles)
		}
	}

	t.Logf("completed %d cycles successfully", totalCycles)
}
