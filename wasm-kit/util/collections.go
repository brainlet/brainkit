package util

import (
	"fmt"
	"math/bits"
	"strings"
)

func CloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func MergeMaps[K comparable, V any](map1, map2 map[K]V) map[K]V {
	out := make(map[K]V, len(map1)+len(map2))
	for k, v := range map1 {
		out[k] = v
	}
	for k, v := range map2 {
		out[k] = v
	}
	return out
}

type BitSet struct {
	words []uint32
}

func NewBitSet() *BitSet {
	return &BitSet{
		words: make([]uint32, 16),
	}
}

func (bs *BitSet) Size() int {
	count := 0
	for _, word := range bs.words {
		count += bits.OnesCount32(word)
	}
	return count
}

func (bs *BitSet) Add(index int) *BitSet {
	idx := index >> 5
	if idx >= len(bs.words) {
		newWords := make([]uint32, idx+16)
		copy(newWords, bs.words)
		bs.words = newWords
	}
	bs.words[idx] |= 1 << (uint(index) & 31)
	return bs
}

func (bs *BitSet) Delete(index int) {
	idx := index >> 5
	if idx >= len(bs.words) {
		return
	}
	bs.words[idx] &^= 1 << (uint(index) & 31)
}

func (bs *BitSet) Has(index int) bool {
	idx := index >> 5
	if idx >= len(bs.words) {
		return false
	}
	return (bs.words[idx] & (1 << (uint(index) & 31))) != 0
}

func (bs *BitSet) Clear() {
	bs.words = make([]uint32, 16)
}

func (bs *BitSet) ToArray() []int {
	result := make([]int, 0, bs.Size())
	for i, word := range bs.words {
		for word != 0 {
			mask := word & (^word + 1) // word & -word
			result = append(result, (i<<5)+bits.OnesCount32(mask-1))
			word ^= mask
		}
	}
	return result
}

func (bs *BitSet) String() string {
	arr := bs.ToArray()
	var sb strings.Builder
	sb.WriteString("BitSet{")
	for i, v := range arr {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%d", v))
	}
	sb.WriteString("}")
	return sb.String()
}
