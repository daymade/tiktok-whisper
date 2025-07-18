package provider

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Benchmark embedding generation across different dimensions
func BenchmarkEmbeddingGenerationByDimension(b *testing.B) {
	dimensions := []int{128, 256, 512, 768, 1024, 1536, 2048, 4096}
	ctx := context.Background()
	testText := "Benchmark text for dimension testing"

	for _, dim := range dimensions {
		b.Run(fmt.Sprintf("Dimension-%d", dim), func(b *testing.B) {
			provider := NewMockProvider(dim)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Benchmark different text lengths
func BenchmarkEmbeddingGenerationByTextLength(b *testing.B) {
	textLengths := []struct {
		name   string
		length int
	}{
		{"Short-10", 10},
		{"Medium-100", 100},
		{"Long-1000", 1000},
		{"VeryLong-10000", 10000},
		{"Extreme-100000", 100000},
	}

	provider := NewMockProvider(768)
	ctx := context.Background()

	for _, tl := range textLengths {
		b.Run(tl.name, func(b *testing.B) {
			// Create text of specified length
			text := strings.Repeat("a", tl.length)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := provider.GenerateEmbedding(ctx, text)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Benchmark concurrent embedding generation
func BenchmarkConcurrentEmbeddingGeneration(b *testing.B) {
	concurrencyLevels := []int{1, 2, 4, 8, 16, 32}
	provider := NewMockProvider(768)
	ctx := context.Background()
	testText := "Concurrent benchmark text"

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := provider.GenerateEmbedding(ctx, testText)
					if err != nil {
						b.Fatalf("Failed to generate embedding: %v", err)
					}
				}
			})
		})
	}
}

// Benchmark provider creation overhead
func BenchmarkProviderCreation(b *testing.B) {
	b.Run("MockProvider", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewMockProvider(768)
		}
	})

	b.Run("OpenAIProvider", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewOpenAIProvider("test-key")
		}
	})

	b.Run("GeminiProvider", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewGeminiProvider("test-key")
		}
	})
}

// Benchmark GetProviderInfo calls
func BenchmarkGetProviderInfo(b *testing.B) {
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"MockProvider", NewMockProvider(768)},
		{"OpenAIProvider", NewOpenAIProvider("test-key")},
		{"GeminiProvider", NewGeminiProvider("test-key")},
	}

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = p.provider.GetProviderInfo()
			}
		})
	}
}

// Benchmark memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	dimensions := []int{256, 768, 1536, 4096}
	ctx := context.Background()
	testText := "Memory allocation benchmark"

	for _, dim := range dimensions {
		b.Run(fmt.Sprintf("Dim-%d", dim), func(b *testing.B) {
			provider := NewMockProvider(dim)
			
			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)
			
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				embedding, err := provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
				// Simulate using the embedding
				_ = len(embedding)
			}

			b.StopTimer()
			runtime.GC()
			runtime.ReadMemStats(&m2)
			
			b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
		})
	}
}

// Benchmark text preprocessing overhead
func BenchmarkTextPreprocessing(b *testing.B) {
	texts := []struct {
		name string
		text string
	}{
		{"ASCII", "Simple ASCII text for testing"},
		{"Unicode", "Unicode text with émojis 🚀 and accénts"},
		{"Mixed", "Mixed content: ASCII + Unicode 中文 + Numbers 123 + Symbols @#$%"},
		{"LongUnicode", strings.Repeat("Unicode émoji 🚀 text ", 100)},
	}

	provider := NewMockProvider(768)
	ctx := context.Background()

	for _, text := range texts {
		b.Run(text.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := provider.GenerateEmbedding(ctx, text.text)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Benchmark provider switching overhead
func BenchmarkProviderSwitching(b *testing.B) {
	providers := []EmbeddingProvider{
		NewMockProvider(768),
		NewGeminiProvider("test-key"),
	}

	ctx := context.Background()
	testText := "Provider switching benchmark"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate switching between providers
		provider := providers[i%len(providers)]
		_, err := provider.GenerateEmbedding(ctx, testText)
		if err != nil {
			b.Fatalf("Failed to generate embedding: %v", err)
		}
	}
}

// Benchmark hash computation (for mock provider)
func BenchmarkHashComputation(b *testing.B) {
	texts := []struct {
		name   string
		length int
	}{
		{"Short", 10},
		{"Medium", 100},
		{"Long", 1000},
		{"VeryLong", 10000},
	}

	provider := NewMockProvider(768)
	ctx := context.Background()

	for _, text := range texts {
		b.Run(text.name, func(b *testing.B) {
			testText := strings.Repeat("a", text.length)
			b.ReportAllocs()
			b.SetBytes(int64(text.length))

			for i := 0; i < b.N; i++ {
				_, err := provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Benchmark embedding normalization
func BenchmarkEmbeddingNormalization(b *testing.B) {
	// This benchmarks the float32 conversion and normalization in mock provider
	dimensions := []int{128, 256, 512, 768, 1024, 1536, 2048, 4096}

	for _, dim := range dimensions {
		b.Run(fmt.Sprintf("Norm-%d", dim), func(b *testing.B) {
			provider := NewMockProvider(dim)
			ctx := context.Background()
			testText := "Normalization benchmark"
			
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				embedding, err := provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
				// Simulate accessing all values (triggers normalization)
				var sum float32
				for _, val := range embedding {
					sum += val
				}
				_ = sum
			}
		})
	}
}

// Benchmark real OpenAI API calls (if available)
func BenchmarkOpenAIRealAPI(b *testing.B) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_API_KEY not set, skipping real API benchmark")
	}

	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()
	testText := "Real OpenAI API benchmark test"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateEmbedding(ctx, testText)
		if err != nil {
			b.Fatalf("OpenAI API call failed: %v", err)
		}
		
		// Add small delay to respect rate limits
		if b.N > 1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Benchmark throughput under load
func BenchmarkThroughputUnderLoad(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping throughput test in short mode")
	}

	provider := NewMockProvider(768)
	ctx := context.Background()
	testText := "Throughput benchmark"

	// Test different load patterns
	loadPatterns := []struct {
		name        string
		workers     int
		requestsPerWorker int
	}{
		{"Light", 5, 10},
		{"Medium", 10, 20},
		{"Heavy", 20, 50},
		{"Extreme", 50, 100},
	}

	for _, pattern := range loadPatterns {
		b.Run(pattern.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				start := time.Now()

				for w := 0; w < pattern.workers; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for r := 0; r < pattern.requestsPerWorker; r++ {
							_, err := provider.GenerateEmbedding(ctx, testText)
							if err != nil {
								b.Errorf("Request failed: %v", err)
								return
							}
						}
					}()
				}

				wg.Wait()
				duration := time.Since(start)
				totalRequests := pattern.workers * pattern.requestsPerWorker
				
				b.ReportMetric(float64(totalRequests)/duration.Seconds(), "requests/sec")
			}
		})
	}
}

// Benchmark different data types as input
func BenchmarkInputDataTypes(b *testing.B) {
	provider := NewMockProvider(768)
	ctx := context.Background()

	testCases := []struct {
		name  string
		input string
	}{
		{"PlainText", "This is plain English text."},
		{"JSON", `{"key": "value", "number": 123, "array": [1, 2, 3]}`},
		{"XML", `<root><item id="1">Value</item><item id="2">Another</item></root>`},
		{"Code", `func main() { fmt.Println("Hello, World!") }`},
		{"Numbers", "1234567890 3.14159 -42 1e10"},
		{"Punctuation", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"Base64", "SGVsbG8gV29ybGQhIFRoaXMgaXMgYSB0ZXN0Lg=="},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := provider.GenerateEmbedding(ctx, tc.input)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Benchmark error handling overhead
func BenchmarkErrorHandling(b *testing.B) {
	provider := NewMockProvider(768)
	ctx := context.Background()

	b.Run("SuccessPath", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := provider.GenerateEmbedding(ctx, "valid text")
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})

	b.Run("ErrorPath", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := provider.GenerateEmbedding(ctx, "") // Empty text causes error
			if err == nil {
				b.Fatal("Expected error for empty text")
			}
		}
	})
}

// Benchmark context overhead
func BenchmarkContextOverhead(b *testing.B) {
	provider := NewMockProvider(256)
	testText := "Context overhead test"

	contextTypes := []struct {
		name    string
		ctxFunc func() context.Context
	}{
		{"Background", func() context.Context { return context.Background() }},
		{"TODO", func() context.Context { return context.TODO() }},
		{"WithValue", func() context.Context { 
			return context.WithValue(context.Background(), "key", "value") 
		}},
		{"WithCancel", func() context.Context { 
			ctx, _ := context.WithCancel(context.Background())
			return ctx
		}},
		{"WithTimeout", func() context.Context { 
			ctx, _ := context.WithTimeout(context.Background(), 1*time.Hour)
			return ctx
		}},
	}

	for _, ct := range contextTypes {
		b.Run(ct.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ctx := ct.ctxFunc()
				_, err := provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Comparative benchmark of all providers
func BenchmarkProviderComparison(b *testing.B) {
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"Mock-256", NewMockProvider(256)},
		{"Mock-768", NewMockProvider(768)},
		{"Mock-1536", NewMockProvider(1536)},
		{"Gemini-Mock", NewGeminiProvider("test-key")},
	}

	ctx := context.Background()
	testText := "Provider comparison benchmark text"

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			
			// Warmup
			for i := 0; i < 10; i++ {
				_, _ = p.provider.GenerateEmbedding(ctx, testText)
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				embedding, err := p.provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Provider %s failed: %v", p.name, err)
				}
				
				// Simulate basic usage
				info := p.provider.GetProviderInfo()
				if len(embedding) != info.Dimension {
					b.Fatalf("Dimension mismatch: got %d, expected %d", len(embedding), info.Dimension)
				}
			}
		})
	}
}