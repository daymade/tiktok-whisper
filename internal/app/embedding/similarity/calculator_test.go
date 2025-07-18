package similarity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TDD Cycle 5: RED - Test SimilarityCalculator interface
func TestSimilarityCalculatorInterface(t *testing.T) {
	// Arrange
	var calculator SimilarityCalculator
	calculator = NewCosineSimilarityCalculator()

	// Test identical vectors
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}

	// Act
	similarity, err := calculator.Calculate(a, b)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, float32(1.0), similarity)
}

// Test cosine similarity calculation
func TestCosineSimilarityCalculation(t *testing.T) {
	calculator := NewCosineSimilarityCalculator()

	testCases := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
		epsilon  float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
			epsilon:  0.001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
			epsilon:  0.001,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
			epsilon:  0.001,
		},
		{
			name:     "45 degree vectors",
			a:        []float32{1, 1, 0},
			b:        []float32{1, 0, 0},
			expected: 0.707, // cos(45°) ≈ 0.707
			epsilon:  0.01,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			similarity, err := calculator.Calculate(tc.a, tc.b)

			// Assert
			assert.NoError(t, err)
			assert.InDelta(t, tc.expected, similarity, float64(tc.epsilon))
		})
	}
}

// Test error cases
func TestCosineSimilarityErrors(t *testing.T) {
	calculator := NewCosineSimilarityCalculator()

	testCases := []struct {
		name        string
		a           []float32
		b           []float32
		expectError bool
	}{
		{
			name:        "different dimensions",
			a:           []float32{1, 0, 0},
			b:           []float32{1, 0},
			expectError: true,
		},
		{
			name:        "zero vectors",
			a:           []float32{0, 0, 0},
			b:           []float32{0, 0, 0},
			expectError: false, // Should return 0, not error
		},
		{
			name:        "empty vectors",
			a:           []float32{},
			b:           []float32{},
			expectError: false, // Should return 0, not error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			similarity, err := calculator.Calculate(tc.a, tc.b)

			// Assert
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.name == "zero vectors" || tc.name == "empty vectors" {
					assert.Equal(t, float32(0), similarity)
				}
			}
		})
	}
}

// Test with large vectors (realistic embeddings)
func TestCosineSimilarityLargeVectors(t *testing.T) {
	calculator := NewCosineSimilarityCalculator()

	// Create large vectors (similar to OpenAI embeddings)
	a := make([]float32, 1536)
	b := make([]float32, 1536)

	// Fill with test data
	for i := range a {
		a[i] = float32(i) / 1536.0
		b[i] = float32(i) / 1536.0 * 0.9 // Similar but slightly different
	}

	// Act
	similarity, err := calculator.Calculate(a, b)

	// Assert
	assert.NoError(t, err)
	assert.Greater(t, similarity, float32(0.9)) // Should be highly similar
	assert.LessOrEqual(t, similarity, float32(1.001)) // Allow for floating point precision
}

// Test euclidean distance calculator
func TestEuclideanDistanceCalculator(t *testing.T) {
	calculator := NewEuclideanDistanceCalculator()

	testCases := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
		epsilon  float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
			epsilon:  0.001,
		},
		{
			name:     "unit distance",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
			epsilon:  0.001,
		},
		{
			name:     "3-4-5 triangle",
			a:        []float32{0, 0, 0},
			b:        []float32{3, 4, 0},
			expected: 5.0,
			epsilon:  0.001,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			distance, err := calculator.Calculate(tc.a, tc.b)

			// Assert
			assert.NoError(t, err)
			assert.InDelta(t, tc.expected, distance, float64(tc.epsilon))
		})
	}
}

// Test similarity threshold functionality
func TestSimilarityThreshold(t *testing.T) {
	calculator := NewCosineSimilarityCalculator()
	
	// Test vectors with known similarity
	a := []float32{1, 1, 0}
	b := []float32{1, 0, 0}
	
	similarity, err := calculator.Calculate(a, b)
	assert.NoError(t, err)
	
	// Test threshold checking
	assert.True(t, similarity > 0.7)  // Should pass 70% threshold
	assert.True(t, similarity < 0.8)  // Should fail 80% threshold
}