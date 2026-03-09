package util

import "math"

func IsPowerOf2(x int32) bool {
	return x != 0 && (x&(x-1)) == 0
}

func AccuratePow64(x, y float64) float64 {
	return math.Pow(x, y)
}
