// Ported from: packages/ai/src/util/split-array.ts
package util

import "fmt"

// SplitArray splits a slice into chunks of a specified size.
func SplitArray[T any](array []T, chunkSize int) ([][]T, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunkSize must be greater than 0")
	}

	var result [][]T
	for i := 0; i < len(array); i += chunkSize {
		end := i + chunkSize
		if end > len(array) {
			end = len(array)
		}
		result = append(result, array[i:end])
	}

	if result == nil {
		result = [][]T{}
	}

	return result, nil
}
