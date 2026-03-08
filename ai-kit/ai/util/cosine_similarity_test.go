// Ported from: packages/ai/src/util/cosine-similarity.test.ts
package util

import (
	"math"
	"testing"
)

func closeTo(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestCosineSimilarity_Basic(t *testing.T) {
	vector1 := []float64{1, 2, 3}
	vector2 := []float64{4, 5, 6}

	result, err := CosineSimilarity(vector1, vector2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closeTo(result, 0.9746318461970762, 1e-5) {
		t.Fatalf("expected ~0.9746, got %f", result)
	}
}

func TestCosineSimilarity_Negative(t *testing.T) {
	vector1 := []float64{1, 0}
	vector2 := []float64{-1, 0}

	result, err := CosineSimilarity(vector1, vector2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closeTo(result, -1, 1e-5) {
		t.Fatalf("expected -1, got %f", result)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	vector1 := []float64{1, 2, 3}
	vector2 := []float64{4, 5}

	_, err := CosineSimilarity(vector1, vector2)
	if err == nil {
		t.Fatal("expected error for different lengths")
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	vector1 := []float64{0, 1, 2}
	vector2 := []float64{0, 0, 0}

	result, err := CosineSimilarity(vector1, vector2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Fatalf("expected 0, got %f", result)
	}

	result2, err := CosineSimilarity(vector2, vector1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != 0 {
		t.Fatalf("expected 0, got %f", result2)
	}
}

func TestCosineSimilarity_SmallMagnitudes(t *testing.T) {
	vector1 := []float64{1e-10, 0, 0}
	vector2 := []float64{2e-10, 0, 0}

	result, err := CosineSimilarity(vector1, vector2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %f", result)
	}

	vector3 := []float64{1e-10, 0, 0}
	vector4 := []float64{-1e-10, 0, 0}

	result2, err := CosineSimilarity(vector3, vector4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != -1 {
		t.Fatalf("expected -1, got %f", result2)
	}
}
