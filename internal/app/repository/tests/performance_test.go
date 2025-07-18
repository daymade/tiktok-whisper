package tests

import (
	"database/sql"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/pg"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/testutil"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// PerformanceTestSuite provides comprehensive performance testing for database operations
type PerformanceTestSuite struct {
	name string
	dao  repository.TranscriptionDAO
	db   *sql.DB
}

// BenchmarkRepositoryPerformance runs comprehensive performance tests across all database implementations
func BenchmarkRepositoryPerformance(b *testing.B) {
	benchmarks := []struct {
		name      string
		setupFunc func(b *testing.B) PerformanceTestSuite
		available func() bool
	}{
		{
			name:      "SQLite",
			setupFunc: setupSQLiteBenchmark,
			available: func() bool { return true },
		},
		{
			name:      "PostgreSQL",
			setupFunc: setupPostgresBenchmark,
			available: isPostgresAvailable,
		},
	}

	for _, bm := range benchmarks {
		if !bm.available() {
			b.Logf("Skipping %s benchmarks - not available", bm.name)
			continue
		}

		b.Run(bm.name, func(b *testing.B) {
			suite := bm.setupFunc(b)
			runPerformanceBenchmarks(b, suite)
		})
	}
}

// runPerformanceBenchmarks executes all performance benchmark tests
func runPerformanceBenchmarks(b *testing.B, suite PerformanceTestSuite) {
	// Single operation benchmarks
	b.Run("RecordToDB_Single", func(b *testing.B) {
		benchmarkRecordToDB(b, suite, 1)
	})

	b.Run("CheckIfFileProcessed_Single", func(b *testing.B) {
		benchmarkCheckIfFileProcessed(b, suite, 1)
	})

	// Batch operation benchmarks
	b.Run("RecordToDB_Batch", func(b *testing.B) {
		benchmarkRecordToDB(b, suite, 100)
	})

	b.Run("CheckIfFileProcessed_Batch", func(b *testing.B) {
		benchmarkCheckIfFileProcessed(b, suite, 100)
	})

	// GetAllByUser benchmarks (if implemented)
	b.Run("GetAllByUser", func(b *testing.B) {
		benchmarkGetAllByUser(b, suite)
	})

	// Concurrent operation benchmarks
	b.Run("RecordToDB_Concurrent", func(b *testing.B) {
		benchmarkConcurrentRecordToDB(b, suite)
	})

	b.Run("CheckIfFileProcessed_Concurrent", func(b *testing.B) {
		benchmarkConcurrentCheckIfFileProcessed(b, suite)
	})

	// Large data benchmarks
	b.Run("RecordToDB_LargeData", func(b *testing.B) {
		benchmarkLargeDataRecordToDB(b, suite)
	})

	// Memory usage benchmarks
	b.Run("MemoryUsage", func(b *testing.B) {
		benchmarkMemoryUsage(b, suite)
	})
}

// benchmarkRecordToDB benchmarks the RecordToDB operation
func benchmarkRecordToDB(b *testing.B, suite PerformanceTestSuite, batchSize int) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	// Pre-generate test data to avoid including generation time in benchmark
	testData := make([]struct {
		user          string
		fileName      string
		transcription string
		duration      int
	}, batchSize)

	for i := 0; i < batchSize; i++ {
		testData[i] = struct {
			user          string
			fileName      string
			transcription string
			duration      int
		}{
			user:          fmt.Sprintf("bench_user_%d", i%10), // Cycle through 10 users
			fileName:      fmt.Sprintf("bench_file_%d.mp3", i),
			transcription: generateBenchmarkTranscription(500), // 500 char transcription
			duration:      120 + rand.Intn(3480),              // 2 min to 1 hour
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := 0; j < batchSize; j++ {
			data := testData[j]
			suite.dao.RecordToDB(
				data.user,
				"/benchmark/input",
				data.fileName,
				data.fileName,
				data.duration,
				data.transcription,
				time.Now(),
				0,
				"",
			)
		}
	}

	// Report custom metrics
	totalOps := int64(b.N * batchSize)
	if b.Elapsed() > 0 {
		opsPerSecond := float64(totalOps) / b.Elapsed().Seconds()
		b.ReportMetric(opsPerSecond, "ops/sec")
	}
}

// benchmarkCheckIfFileProcessed benchmarks the CheckIfFileProcessed operation
func benchmarkCheckIfFileProcessed(b *testing.B, suite PerformanceTestSuite, batchSize int) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	// Seed the database with test data
	for i := 0; i < batchSize; i++ {
		suite.dao.RecordToDB(
			"bench_user",
			"/benchmark/input",
			fmt.Sprintf("bench_file_%d.mp3", i),
			fmt.Sprintf("bench_file_%d.mp3", i),
			120,
			"Benchmark transcription",
			time.Now(),
			0,
			"",
		)
	}

	// Pre-generate file names to check
	fileNames := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		fileNames[i] = fmt.Sprintf("bench_file_%d.mp3", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := 0; j < batchSize; j++ {
			_, _ = suite.dao.CheckIfFileProcessed(fileNames[j])
		}
	}

	// Report custom metrics
	totalOps := int64(b.N * batchSize)
	if b.Elapsed() > 0 {
		opsPerSecond := float64(totalOps) / b.Elapsed().Seconds()
		b.ReportMetric(opsPerSecond, "ops/sec")
	}
}

// benchmarkGetAllByUser benchmarks the GetAllByUser operation
func benchmarkGetAllByUser(b *testing.B, suite PerformanceTestSuite) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	// Seed the database with test data for multiple users
	usersData := map[string]int{
		"user_small":  10,   // 10 records
		"user_medium": 50,   // 50 records  
		"user_large":  200,  // 200 records
		"user_xlarge": 1000, // 1000 records
	}

	for user, count := range usersData {
		for i := 0; i < count; i++ {
			suite.dao.RecordToDB(
				user,
				"/benchmark/input",
				fmt.Sprintf("%s_file_%d.mp3", user, i),
				fmt.Sprintf("%s_file_%d.mp3", user, i),
				120,
				generateBenchmarkTranscription(300),
				time.Now().Add(-time.Duration(i)*time.Minute), // Varied timestamps
				0,
				"",
			)
		}
	}

	// Test each user size category
	for user, expectedCount := range usersData {
		b.Run(fmt.Sprintf("Records_%d", expectedCount), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				transcriptions, err := suite.dao.GetAllByUser(user)
				if err != nil && err.Error() != "not implemented" {
					b.Fatalf("Unexpected error: %v", err)
				}
				if err == nil && len(transcriptions) != expectedCount {
					b.Fatalf("Expected %d transcriptions, got %d", expectedCount, len(transcriptions))
				}
			}

			// Report records per second
			if b.Elapsed() > 0 && expectedCount > 0 {
				recordsPerSecond := float64(int64(b.N)*int64(expectedCount)) / b.Elapsed().Seconds()
				b.ReportMetric(recordsPerSecond, "records/sec")
			}
		})
	}
}

// benchmarkConcurrentRecordToDB benchmarks concurrent RecordToDB operations
func benchmarkConcurrentRecordToDB(b *testing.B, suite PerformanceTestSuite) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 50

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					suite.dao.RecordToDB(
						fmt.Sprintf("concurrent_user_%d", goroutineID),
						"/benchmark/concurrent",
						fmt.Sprintf("concurrent_file_%d_%d.mp3", goroutineID, j),
						fmt.Sprintf("concurrent_file_%d_%d.mp3", goroutineID, j),
						120,
						generateBenchmarkTranscription(200),
						time.Now(),
						0,
						"",
					)
				}
			}(g)
		}

		wg.Wait()
	}

	// Report custom metrics
	totalOps := int64(b.N * numGoroutines * operationsPerGoroutine)
	if b.Elapsed() > 0 {
		opsPerSecond := float64(totalOps) / b.Elapsed().Seconds()
		b.ReportMetric(opsPerSecond, "ops/sec")
		b.ReportMetric(float64(numGoroutines), "goroutines")
	}
}

// benchmarkConcurrentCheckIfFileProcessed benchmarks concurrent CheckIfFileProcessed operations
func benchmarkConcurrentCheckIfFileProcessed(b *testing.B, suite PerformanceTestSuite) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	numFiles := 1000
	// Seed database with files to check
	for i := 0; i < numFiles; i++ {
		suite.dao.RecordToDB(
			"concurrent_check_user",
			"/benchmark/concurrent",
			fmt.Sprintf("check_file_%d.mp3", i),
			fmt.Sprintf("check_file_%d.mp3", i),
			120,
			"Concurrent check transcription",
			time.Now(),
			0,
			"",
		)
	}

	numGoroutines := runtime.NumCPU()
	checksPerGoroutine := 100

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < checksPerGoroutine; j++ {
					fileIndex := (goroutineID*checksPerGoroutine + j) % numFiles
					fileName := fmt.Sprintf("check_file_%d.mp3", fileIndex)
					_, _ = suite.dao.CheckIfFileProcessed(fileName)
				}
			}(g)
		}

		wg.Wait()
	}

	// Report custom metrics
	totalOps := int64(b.N * numGoroutines * checksPerGoroutine)
	if b.Elapsed() > 0 {
		opsPerSecond := float64(totalOps) / b.Elapsed().Seconds()
		b.ReportMetric(opsPerSecond, "ops/sec")
		b.ReportMetric(float64(numGoroutines), "goroutines")
	}
}

// benchmarkLargeDataRecordToDB benchmarks RecordToDB with large transcriptions
func benchmarkLargeDataRecordToDB(b *testing.B, suite PerformanceTestSuite) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	// Test with different transcription sizes
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			transcription := generateBenchmarkTranscription(size.size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				suite.dao.RecordToDB(
					"large_data_user",
					"/benchmark/large",
					fmt.Sprintf("large_file_%d.mp3", i),
					fmt.Sprintf("large_file_%d.mp3", i),
					3600, // 1 hour duration
					transcription,
					time.Now(),
					0,
					"",
				)
			}

			// Report custom metrics
			bytesPerOp := int64(size.size)
			if b.Elapsed() > 0 {
				mbPerSecond := float64(int64(b.N)*bytesPerOp) / (1024 * 1024) / b.Elapsed().Seconds()
				b.ReportMetric(mbPerSecond, "MB/sec")
			}
		})
	}
}

// benchmarkMemoryUsage benchmarks memory usage patterns
func benchmarkMemoryUsage(b *testing.B, suite PerformanceTestSuite) {
	testutil.CleanTestData(&testing.T{}, suite.db)

	benchmark := testutil.NewBenchmarkHelper(fmt.Sprintf("%s_MemoryUsage", suite.name))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchmark.Start()

		// Perform a mixed workload
		benchmark.Measure("mixed_operations", func() {
			// Record operations
			for j := 0; j < 50; j++ {
				suite.dao.RecordToDB(
					"memory_user",
					"/benchmark/memory",
					fmt.Sprintf("memory_file_%d.mp3", j),
					fmt.Sprintf("memory_file_%d.mp3", j),
					120,
					generateBenchmarkTranscription(1000),
					time.Now(),
					0,
					"",
				)
			}

			// Check operations
			for j := 0; j < 50; j++ {
				_, _ = suite.dao.CheckIfFileProcessed(fmt.Sprintf("memory_file_%d.mp3", j))
			}

			// GetAllByUser operation (if implemented)
			_, _ = suite.dao.GetAllByUser("memory_user")
		})

		benchmark.Stop()

		// Force garbage collection to see real memory usage
		runtime.GC()
	}

	// Report memory statistics
	memStats := benchmark.MemoryUsage()
	b.ReportMetric(float64(memStats.AllocatedDelta), "bytes_allocated")
	b.ReportMetric(float64(memStats.GCCyclesDelta), "gc_cycles")

	if testing.Verbose() {
		b.Log(benchmark.Report())
	}
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	// Define performance baselines (adjust based on your requirements)
	baselines := map[string]struct {
		maxDuration time.Duration
		maxMemoryMB int64
	}{
		"RecordToDB_Single":    {maxDuration: 10 * time.Millisecond, maxMemoryMB: 1},
		"CheckFile_Single":     {maxDuration: 5 * time.Millisecond, maxMemoryMB: 1},
		"GetAllByUser_Small":   {maxDuration: 20 * time.Millisecond, maxMemoryMB: 2},
		"Concurrent_100_Ops":   {maxDuration: 500 * time.Millisecond, maxMemoryMB: 10},
		"LargeData_100KB":      {maxDuration: 100 * time.Millisecond, maxMemoryMB: 5},
	}

	databases := []struct {
		name      string
		available func() bool
		setup     func(t *testing.T) PerformanceTestSuite
	}{
		{
			name:      "SQLite",
			available: func() bool { return true },
			setup:     setupSQLiteTest,
		},
		{
			name:      "PostgreSQL",
			available: isPostgresAvailable,
			setup:     setupPostgresTest,
		},
	}

	for _, db := range databases {
		if !db.available() {
			t.Logf("Skipping %s regression tests - not available", db.name)
			continue
		}

		t.Run(db.name, func(t *testing.T) {
			suite := db.setup(t)
			defer suite.dao.Close()

			for testName, baseline := range baselines {
				t.Run(testName, func(t *testing.T) {
					benchmark := testutil.NewBenchmarkHelper(testName)
					benchmark.Start()

					switch testName {
					case "RecordToDB_Single":
						suite.dao.RecordToDB(
							"regression_user",
							"/test/regression",
							"regression_file.mp3",
							"regression_file.mp3",
							120,
							"Regression test transcription",
							time.Now(),
							0,
							"",
						)

					case "CheckFile_Single":
						// First record a file
						suite.dao.RecordToDB(
							"regression_user",
							"/test/regression",
							"check_file.mp3",
							"check_file.mp3",
							120,
							"Check file transcription",
							time.Now(),
							0,
							"",
						)
						_, _ = suite.dao.CheckIfFileProcessed("check_file.mp3")

					case "GetAllByUser_Small":
						// Record 10 files for a user
						for i := 0; i < 10; i++ {
							suite.dao.RecordToDB(
								"small_user",
								"/test/regression",
								fmt.Sprintf("small_file_%d.mp3", i),
								fmt.Sprintf("small_file_%d.mp3", i),
								120,
								"Small user transcription",
								time.Now(),
								0,
								"",
							)
						}
						_, _ = suite.dao.GetAllByUser("small_user")

					case "Concurrent_100_Ops":
						var wg sync.WaitGroup
						numGoroutines := 10
						opsPerGoroutine := 10

						wg.Add(numGoroutines)
						for g := 0; g < numGoroutines; g++ {
							go func(goroutineID int) {
								defer wg.Done()
								for j := 0; j < opsPerGoroutine; j++ {
									suite.dao.RecordToDB(
										fmt.Sprintf("concurrent_user_%d", goroutineID),
										"/test/regression",
										fmt.Sprintf("concurrent_file_%d_%d.mp3", goroutineID, j),
										fmt.Sprintf("concurrent_file_%d_%d.mp3", goroutineID, j),
										120,
										"Concurrent regression transcription",
										time.Now(),
										0,
										"",
									)
								}
							}(g)
						}
						wg.Wait()

					case "LargeData_100KB":
						largeTranscription := generateBenchmarkTranscription(100 * 1024) // 100KB
						suite.dao.RecordToDB(
							"large_data_user",
							"/test/regression",
							"large_data_file.mp3",
							"large_data_file.mp3",
							3600,
							largeTranscription,
							time.Now(),
							0,
							"",
						)
					}

					benchmark.Stop()

					// Check against baselines
					benchmark.AssertDurationLessThan(t, baseline.maxDuration)
					benchmark.AssertMemoryUsageLessThan(t, baseline.maxMemoryMB*1024*1024) // Convert MB to bytes

					if testing.Verbose() {
						t.Log(benchmark.Report())
					}
				})
			}
		})
	}
}

// Setup functions for benchmarks and tests

func setupSQLiteBenchmark(b *testing.B) PerformanceTestSuite {
	db := testutil.SetupTestSQLite(&testing.T{})
	sqliteDAO := &sqlite.SQLiteDB{}
	type sqliteDBInternal struct {
		db *sql.DB
	}
	(*sqliteDBInternal)(sqliteDAO).db = db
	
	return PerformanceTestSuite{
		name: "SQLite",
		dao:  sqliteDAO,
		db:   db,
	}
}

func setupPostgresBenchmark(b *testing.B) PerformanceTestSuite {
	db := testutil.SetupTestPostgres(&testing.T{})
	postgresDAO := &pg.PostgresDB{}
	type postgresDBInternal struct {
		db *sql.DB
	}
	(*postgresDBInternal)(postgresDAO).db = db
	
	return PerformanceTestSuite{
		name: "PostgreSQL",
		dao:  postgresDAO,
		db:   db,
	}
}

func setupSQLiteTest(t *testing.T) PerformanceTestSuite {
	db := testutil.SetupTestSQLite(t)
	sqliteDAO := &sqlite.SQLiteDB{}
	type sqliteDBInternal struct {
		db *sql.DB
	}
	(*sqliteDBInternal)(sqliteDAO).db = db
	
	return PerformanceTestSuite{
		name: "SQLite",
		dao:  sqliteDAO,
		db:   db,
	}
}

func setupPostgresTest(t *testing.T) PerformanceTestSuite {
	db := testutil.SetupTestPostgres(t)
	postgresDAO := &pg.PostgresDB{}
	type postgresDBInternal struct {
		db *sql.DB
	}
	(*postgresDBInternal)(postgresDAO).db = db
	
	return PerformanceTestSuite{
		name: "PostgreSQL",
		dao:  postgresDAO,
		db:   db,
	}
}

// Helper functions

// generateBenchmarkTranscription generates a transcription of specified length for benchmarking
func generateBenchmarkTranscription(length int) string {
	const baseText = "This is a benchmark transcription text that will be repeated to reach the desired length. It contains various characters including spaces, punctuation, and Unicode: 测试 🎵 áéíóú. "
	
	if length <= len(baseText) {
		return baseText[:length]
	}
	
	result := ""
	for len(result) < length {
		remaining := length - len(result)
		if remaining >= len(baseText) {
			result += baseText
		} else {
			result += baseText[:remaining]
		}
	}
	
	return result
}

// isPostgresAvailable checks if PostgreSQL is available for testing
func isPostgresAvailable() bool {
	if os := testutil.DefaultPostgresConfig(); os.Host != "" {
		return true
	}
	
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost/postgres?sslmode=disable")
	if err != nil {
		return false
	}
	defer db.Close()
	
	return db.Ping() == nil
}