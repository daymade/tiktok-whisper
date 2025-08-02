package vector

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// BenchmarkVectorOperations benchmarks vector storage operations
func BenchmarkVectorOperations(b *testing.B) {
	// Skip if no PostgreSQL available
	if testing.Short() || os.Getenv("SKIP_PG_TESTS") == "true" {
		b.Skip("Skipping PostgreSQL benchmarks")
	}

	db := setupBenchmarkDB(b)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	// Prepare test data
	openaiEmbedding := generateRandomEmbedding(1536)
	geminiEmbedding := generateRandomEmbedding(768)

	b.Run("StoreEmbedding/OpenAI", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := storage.StoreEmbedding(ctx, i+1, "openai", openaiEmbedding)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("StoreEmbedding/Gemini", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := storage.StoreEmbedding(ctx, i+1, "gemini", geminiEmbedding)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("StoreDualEmbeddings", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := storage.StoreDualEmbeddings(ctx, i+1, openaiEmbedding, geminiEmbedding)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Pre-populate some data for retrieval benchmarks
	for i := 1; i <= 100; i++ {
		_ = storage.StoreDualEmbeddings(ctx, i, openaiEmbedding, geminiEmbedding)
	}

	b.Run("GetEmbedding/OpenAI", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := (i % 100) + 1
			_, err := storage.GetEmbedding(ctx, id, "openai")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetDualEmbeddings", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := (i % 100) + 1
			_, err := storage.GetDualEmbeddings(ctx, id)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetTranscriptionsWithoutEmbeddings", func(b *testing.B) {
		// Clear some embeddings to have data to retrieve
		_, _ = db.Exec("UPDATE transcriptions SET embedding_openai = NULL WHERE id % 2 = 0")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkVectorConversion benchmarks vector string conversion
func BenchmarkVectorConversion(b *testing.B) {
	sizes := []int{128, 768, 1536, 3072}

	for _, size := range sizes {
		vector := generateRandomEmbedding(size)

		b.Run(fmt.Sprintf("VectorToString/size=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = vectorToString(vector)
			}
		})

		vectorStr := vectorToString(vector)

		b.Run(fmt.Sprintf("StringToVector/size=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = stringToVector(vectorStr)
			}
		})
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent access patterns
func BenchmarkConcurrentOperations(b *testing.B) {
	if testing.Short() || os.Getenv("SKIP_PG_TESTS") == "true" {
		b.Skip("Skipping PostgreSQL benchmarks")
	}

	db := setupBenchmarkDB(b)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()
	embedding := generateRandomEmbedding(1536)

	// Pre-populate data
	for i := 1; i <= 1000; i++ {
		_ = storage.StoreEmbedding(ctx, i, "openai", embedding)
	}

	concurrencyLevels := []int{1, 2, 4, 8, 16}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("ConcurrentReads/concurrency=%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					id := rand.Intn(1000) + 1
					_, _ = storage.GetEmbedding(ctx, id, "openai")
				}
			})
		})

		b.Run(fmt.Sprintf("ConcurrentWrites/concurrency=%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			counter := 0
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					counter++
					_ = storage.StoreEmbedding(ctx, 1000+counter, "openai", embedding)
				}
			})
		})
	}
}

// BenchmarkMockVsReal compares mock and real storage performance
func BenchmarkMockVsReal(b *testing.B) {
	ctx := context.Background()
	embedding := generateRandomEmbedding(1536)

	b.Run("MockStorage/StoreEmbedding", func(b *testing.B) {
		storage := NewMockVectorStorage()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = storage.StoreEmbedding(ctx, i, "openai", embedding)
		}
	})

	b.Run("MockStorage/GetEmbedding", func(b *testing.B) {
		storage := NewMockVectorStorage()
		// Pre-populate
		for i := 0; i < 100; i++ {
			_ = storage.StoreEmbedding(ctx, i, "openai", embedding)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = storage.GetEmbedding(ctx, i%100, "openai")
		}
	})

	// Compare with real storage if available
	if !testing.Short() && os.Getenv("SKIP_PG_TESTS") != "true" {
		db := setupBenchmarkDB(b)
		defer db.Close()
		storage := NewPgVectorStorage(db)
		defer storage.Close()

		b.Run("PgVectorStorage/StoreEmbedding", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = storage.StoreEmbedding(ctx, i+10000, "openai", embedding)
			}
		})

		b.Run("PgVectorStorage/GetEmbedding", func(b *testing.B) {
			// Pre-populate
			for i := 0; i < 100; i++ {
				_ = storage.StoreEmbedding(ctx, i+20000, "openai", embedding)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = storage.GetEmbedding(ctx, (i%100)+20000, "openai")
			}
		})
	}
}

// BenchmarkLargeScaleOperations benchmarks operations at scale
func BenchmarkLargeScaleOperations(b *testing.B) {
	if testing.Short() || os.Getenv("SKIP_PG_TESTS") == "true" {
		b.Skip("Skipping large scale benchmarks")
	}

	db := setupBenchmarkDB(b)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	// Test with different batch sizes
	batchSizes := []int{10, 50, 100, 500}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("GetTranscriptionsWithoutEmbeddings/batch=%d", batchSize), func(b *testing.B) {
			// Ensure we have enough data
			for i := 1; i <= batchSize*2; i++ {
				_, _ = db.Exec(`
					INSERT INTO transcriptions (user_nickname, mp3_file_name, transcription)
					VALUES ($1, $2, $3)
					ON CONFLICT DO NOTHING
				`, "bench_user", fmt.Sprintf("bench_%d.mp3", i), "Benchmark transcription")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", batchSize)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Helper functions

func setupBenchmarkDB(b *testing.B) *sql.DB {
	b.Helper()

	var db *sql.DB
	var err error

	if pgURL := os.Getenv("POSTGRES_TEST_URL"); pgURL != "" {
		db, err = sql.Open("postgres", pgURL)
	} else {
		db, err = sql.Open("postgres", "user=postgres password=passwd dbname=postgres sslmode=disable host=localhost")
	}

	if err != nil {
		b.Fatal(err)
	}

	// Create extension and tables
	_, _ = db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

	schema := `
	CREATE TABLE IF NOT EXISTS transcriptions (
		id SERIAL PRIMARY KEY,
		user_nickname VARCHAR(255),
		mp3_file_name VARCHAR(255) NOT NULL,
		transcription TEXT NOT NULL,
		last_conversion_time TIMESTAMP DEFAULT NOW(),
		embedding_openai vector(1536),
		embedding_openai_model VARCHAR(50),
		embedding_openai_created_at TIMESTAMP,
		embedding_openai_status VARCHAR(20) DEFAULT 'pending',
		embedding_gemini vector(768),
		embedding_gemini_model VARCHAR(50),
		embedding_gemini_created_at TIMESTAMP,
		embedding_gemini_status VARCHAR(20) DEFAULT 'pending',
		embedding_sync_status VARCHAR(20) DEFAULT 'pending'
	);`

	_, err = db.Exec(schema)
	if err != nil {
		b.Fatal(err)
	}

	// Clean up old benchmark data
	_, _ = db.Exec("TRUNCATE TABLE transcriptions RESTART IDENTITY CASCADE")

	return db
}

func generateRandomEmbedding(size int) []float32 {
	rand.Seed(time.Now().UnixNano())
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = rand.Float32()
	}
	return embedding
}
