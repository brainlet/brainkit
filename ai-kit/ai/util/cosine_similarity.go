// Ported from: packages/ai/src/util/cosine-similarity.ts
package util

import (
	"fmt"
	"math"
)

// CosineSimilarity calculates the cosine similarity between two vectors.
// This is a useful metric for comparing the similarity of two vectors such as embeddings.
//
// Returns the cosine similarity between vector1 and vector2, or 0 if either vector
// is the zero vector.
//
// Returns an error if the vectors do not have the same length.
func CosineSimilarity(vector1, vector2 []float64) (float64, error) {
	if len(vector1) != len(vector2) {
		return 0, fmt.Errorf(
			"invalid argument for parameter vector1,vector2: Vectors must have the same length (vector1Length: %d, vector2Length: %d)",
			len(vector1), len(vector2),
		)
	}

	n := len(vector1)

	if n == 0 {
		return 0, nil
	}

	var magnitudeSquared1 float64
	var magnitudeSquared2 float64
	var dotProduct float64

	for i := 0; i < n; i++ {
		v1 := vector1[i]
		v2 := vector2[i]

		magnitudeSquared1 += v1 * v1
		magnitudeSquared2 += v2 * v2
		dotProduct += v1 * v2
	}

	if magnitudeSquared1 == 0 || magnitudeSquared2 == 0 {
		return 0, nil
	}

	return dotProduct / (math.Sqrt(magnitudeSquared1) * math.Sqrt(magnitudeSquared2)), nil
}
