package similarity

import (
	"errors"
	"math"
)

// SimilarityCalculator defines the interface for similarity calculations
// Following Single Responsibility Principle - focused on similarity calculation only
type SimilarityCalculator interface {
	Calculate(a, b []float32) (float32, error)
}

// CosineSimilarityCalculator implements cosine similarity calculation
type CosineSimilarityCalculator struct{}

// NewCosineSimilarityCalculator creates a new cosine similarity calculator
func NewCosineSimilarityCalculator() *CosineSimilarityCalculator {
	return &CosineSimilarityCalculator{}
}

// Calculate computes cosine similarity between two vectors
func (c *CosineSimilarityCalculator) Calculate(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, errors.New("vectors must have same dimension")
	}

	if len(a) == 0 {
		return 0, nil
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	// Handle zero vectors
	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) *
		float32(math.Sqrt(float64(normB)))), nil
}

// EuclideanDistanceCalculator implements Euclidean distance calculation
type EuclideanDistanceCalculator struct{}

// NewEuclideanDistanceCalculator creates a new Euclidean distance calculator
func NewEuclideanDistanceCalculator() *EuclideanDistanceCalculator {
	return &EuclideanDistanceCalculator{}
}

// Calculate computes Euclidean distance between two vectors
func (e *EuclideanDistanceCalculator) Calculate(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, errors.New("vectors must have same dimension")
	}

	if len(a) == 0 {
		return 0, nil
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum))), nil
}
