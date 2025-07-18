package testutil

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkHelper provides utilities for performance testing and benchmarking
type BenchmarkHelper struct {
	startTime    time.Time
	endTime      time.Time
	memStart     runtime.MemStats
	memEnd       runtime.MemStats
	measurements []Measurement
	mutex        sync.RWMutex
	name         string
}

// Measurement represents a single performance measurement
type Measurement struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	MemBefore runtime.MemStats
	MemAfter  runtime.MemStats
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(name string) *BenchmarkHelper {
	return &BenchmarkHelper{
		name:         name,
		measurements: make([]Measurement, 0),
	}
}

// Start begins timing and memory tracking
func (bh *BenchmarkHelper) Start() {
	bh.mutex.Lock()
	defer bh.mutex.Unlock()

	runtime.GC() // Force garbage collection before measurement
	runtime.ReadMemStats(&bh.memStart)
	bh.startTime = time.Now()
}

// Stop ends timing and memory tracking
func (bh *BenchmarkHelper) Stop() {
	bh.mutex.Lock()
	defer bh.mutex.Unlock()

	bh.endTime = time.Now()
	runtime.ReadMemStats(&bh.memEnd)
}

// Duration returns the total duration of the benchmark
func (bh *BenchmarkHelper) Duration() time.Duration {
	bh.mutex.RLock()
	defer bh.mutex.RUnlock()

	if bh.endTime.IsZero() {
		return time.Since(bh.startTime)
	}
	return bh.endTime.Sub(bh.startTime)
}

// MemoryUsage returns the memory usage statistics
func (bh *BenchmarkHelper) MemoryUsage() MemoryStats {
	bh.mutex.RLock()
	defer bh.mutex.RUnlock()

	return MemoryStats{
		AllocatedStart:  bh.memStart.Alloc,
		AllocatedEnd:    bh.memEnd.Alloc,
		AllocatedDelta:  int64(bh.memEnd.Alloc) - int64(bh.memStart.Alloc),
		TotalAllocStart: bh.memStart.TotalAlloc,
		TotalAllocEnd:   bh.memEnd.TotalAlloc,
		TotalAllocDelta: int64(bh.memEnd.TotalAlloc) - int64(bh.memStart.TotalAlloc),
		SysMemoryStart:  bh.memStart.Sys,
		SysMemoryEnd:    bh.memEnd.Sys,
		SysMemoryDelta:  int64(bh.memEnd.Sys) - int64(bh.memStart.Sys),
		GCCyclesStart:   bh.memStart.NumGC,
		GCCyclesEnd:     bh.memEnd.NumGC,
		GCCyclesDelta:   bh.memEnd.NumGC - bh.memStart.NumGC,
		PauseTotalStart: bh.memStart.PauseTotalNs,
		PauseTotalEnd:   bh.memEnd.PauseTotalNs,
		PauseTotalDelta: bh.memEnd.PauseTotalNs - bh.memStart.PauseTotalNs,
	}
}

// MemoryStats holds memory usage statistics
type MemoryStats struct {
	AllocatedStart  uint64
	AllocatedEnd    uint64
	AllocatedDelta  int64
	TotalAllocStart uint64
	TotalAllocEnd   uint64
	TotalAllocDelta int64
	SysMemoryStart  uint64
	SysMemoryEnd    uint64
	SysMemoryDelta  int64
	GCCyclesStart   uint32
	GCCyclesEnd     uint32
	GCCyclesDelta   uint32
	PauseTotalStart uint64
	PauseTotalEnd   uint64
	PauseTotalDelta uint64
}

// Measure executes a function and measures its performance
func (bh *BenchmarkHelper) Measure(name string, fn func()) {
	bh.mutex.Lock()
	defer bh.mutex.Unlock()

	var memBefore, memAfter runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	startTime := time.Now()

	fn()

	endTime := time.Now()
	runtime.ReadMemStats(&memAfter)

	measurement := Measurement{
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		MemBefore: memBefore,
		MemAfter:  memAfter,
	}

	bh.measurements = append(bh.measurements, measurement)
}

// GetMeasurements returns all measurements
func (bh *BenchmarkHelper) GetMeasurements() []Measurement {
	bh.mutex.RLock()
	defer bh.mutex.RUnlock()

	// Return a copy to avoid race conditions
	measurements := make([]Measurement, len(bh.measurements))
	copy(measurements, bh.measurements)
	return measurements
}

// GetMeasurement returns a specific measurement by name
func (bh *BenchmarkHelper) GetMeasurement(name string) (Measurement, bool) {
	bh.mutex.RLock()
	defer bh.mutex.RUnlock()

	for _, m := range bh.measurements {
		if m.Name == name {
			return m, true
		}
	}
	return Measurement{}, false
}

// Report generates a performance report
func (bh *BenchmarkHelper) Report() string {
	bh.mutex.RLock()
	defer bh.mutex.RUnlock()

	report := fmt.Sprintf("Benchmark Report: %s\n", bh.name)
	report += fmt.Sprintf("==========================================\n")
	report += fmt.Sprintf("Total Duration: %v\n", bh.Duration())

	memStats := bh.MemoryUsage()
	report += fmt.Sprintf("Memory Usage:\n")
	report += fmt.Sprintf("  Allocated: %d bytes (delta: %+d)\n", memStats.AllocatedEnd, memStats.AllocatedDelta)
	report += fmt.Sprintf("  Total Allocated: %d bytes (delta: %+d)\n", memStats.TotalAllocEnd, memStats.TotalAllocDelta)
	report += fmt.Sprintf("  System Memory: %d bytes (delta: %+d)\n", memStats.SysMemoryEnd, memStats.SysMemoryDelta)
	report += fmt.Sprintf("  GC Cycles: %d (delta: %+d)\n", memStats.GCCyclesEnd, memStats.GCCyclesDelta)
	report += fmt.Sprintf("  GC Pause Time: %d ns (delta: %+d)\n", memStats.PauseTotalEnd, memStats.PauseTotalDelta)

	if len(bh.measurements) > 0 {
		report += fmt.Sprintf("\nIndividual Measurements:\n")
		for _, m := range bh.measurements {
			report += fmt.Sprintf("  %s: %v\n", m.Name, m.Duration)
			memDelta := int64(m.MemAfter.Alloc) - int64(m.MemBefore.Alloc)
			report += fmt.Sprintf("    Memory Delta: %+d bytes\n", memDelta)
		}
	}

	return report
}

// Reset clears all measurements and resets the benchmark helper
func (bh *BenchmarkHelper) Reset() {
	bh.mutex.Lock()
	defer bh.mutex.Unlock()

	bh.startTime = time.Time{}
	bh.endTime = time.Time{}
	bh.memStart = runtime.MemStats{}
	bh.memEnd = runtime.MemStats{}
	bh.measurements = make([]Measurement, 0)
}

// Performance assertion helpers

// AssertDurationLessThan asserts that the total duration is less than the specified duration
func (bh *BenchmarkHelper) AssertDurationLessThan(t *testing.T, maxDuration time.Duration) {
	t.Helper()

	duration := bh.Duration()
	if duration > maxDuration {
		t.Errorf("Expected duration to be less than %v, got %v", maxDuration, duration)
	}
}

// AssertMemoryUsageLessThan asserts that the memory usage is less than the specified amount
func (bh *BenchmarkHelper) AssertMemoryUsageLessThan(t *testing.T, maxMemory int64) {
	t.Helper()

	memStats := bh.MemoryUsage()
	if memStats.AllocatedDelta > maxMemory {
		t.Errorf("Expected memory usage to be less than %d bytes, got %d bytes", maxMemory, memStats.AllocatedDelta)
	}
}

// AssertGCCyclesLessThan asserts that the number of GC cycles is less than the specified amount
func (bh *BenchmarkHelper) AssertGCCyclesLessThan(t *testing.T, maxCycles uint32) {
	t.Helper()

	memStats := bh.MemoryUsage()
	if memStats.GCCyclesDelta > maxCycles {
		t.Errorf("Expected GC cycles to be less than %d, got %d", maxCycles, memStats.GCCyclesDelta)
	}
}

// BenchmarkRunner provides utilities for running benchmarks
type BenchmarkRunner struct {
	iterations int
	warmup     int
	parallel   bool
}

// NewBenchmarkRunner creates a new benchmark runner
func NewBenchmarkRunner() *BenchmarkRunner {
	return &BenchmarkRunner{
		iterations: 1000,
		warmup:     100,
		parallel:   false,
	}
}

// WithIterations sets the number of iterations for the benchmark
func (br *BenchmarkRunner) WithIterations(iterations int) *BenchmarkRunner {
	br.iterations = iterations
	return br
}

// WithWarmup sets the number of warmup iterations
func (br *BenchmarkRunner) WithWarmup(warmup int) *BenchmarkRunner {
	br.warmup = warmup
	return br
}

// WithParallel enables parallel execution
func (br *BenchmarkRunner) WithParallel(parallel bool) *BenchmarkRunner {
	br.parallel = parallel
	return br
}

// Run executes the benchmark with the specified configuration
func (br *BenchmarkRunner) Run(name string, fn func()) *BenchmarkResults {
	results := &BenchmarkResults{
		Name:       name,
		Iterations: br.iterations,
		Warmup:     br.warmup,
		Parallel:   br.parallel,
		Durations:  make([]time.Duration, 0, br.iterations),
	}

	// Warmup phase
	for i := 0; i < br.warmup; i++ {
		fn()
	}

	// Benchmark phase
	runtime.GC()
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	startTime := time.Now()

	if br.parallel {
		br.runParallel(fn, results)
	} else {
		br.runSequential(fn, results)
	}

	endTime := time.Now()
	runtime.ReadMemStats(&memAfter)

	results.TotalDuration = endTime.Sub(startTime)
	results.MemoryBefore = memBefore
	results.MemoryAfter = memAfter

	return results
}

// runSequential runs the benchmark sequentially
func (br *BenchmarkRunner) runSequential(fn func(), results *BenchmarkResults) {
	for i := 0; i < br.iterations; i++ {
		start := time.Now()
		fn()
		duration := time.Since(start)
		results.Durations = append(results.Durations, duration)
	}
}

// runParallel runs the benchmark in parallel
func (br *BenchmarkRunner) runParallel(fn func(), results *BenchmarkResults) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	numCPU := runtime.NumCPU()
	iterationsPerCPU := br.iterations / numCPU

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			localDurations := make([]time.Duration, 0, iterationsPerCPU)
			for j := 0; j < iterationsPerCPU; j++ {
				start := time.Now()
				fn()
				duration := time.Since(start)
				localDurations = append(localDurations, duration)
			}

			mutex.Lock()
			results.Durations = append(results.Durations, localDurations...)
			mutex.Unlock()
		}()
	}

	wg.Wait()
}

// BenchmarkResults holds the results of a benchmark run
type BenchmarkResults struct {
	Name          string
	Iterations    int
	Warmup        int
	Parallel      bool
	TotalDuration time.Duration
	Durations     []time.Duration
	MemoryBefore  runtime.MemStats
	MemoryAfter   runtime.MemStats
}

// AverageDuration returns the average duration per iteration
func (br *BenchmarkResults) AverageDuration() time.Duration {
	if len(br.Durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range br.Durations {
		total += d
	}

	return total / time.Duration(len(br.Durations))
}

// MinDuration returns the minimum duration
func (br *BenchmarkResults) MinDuration() time.Duration {
	if len(br.Durations) == 0 {
		return 0
	}

	min := br.Durations[0]
	for _, d := range br.Durations[1:] {
		if d < min {
			min = d
		}
	}

	return min
}

// MaxDuration returns the maximum duration
func (br *BenchmarkResults) MaxDuration() time.Duration {
	if len(br.Durations) == 0 {
		return 0
	}

	max := br.Durations[0]
	for _, d := range br.Durations[1:] {
		if d > max {
			max = d
		}
	}

	return max
}

// MemoryDelta returns the memory usage delta
func (br *BenchmarkResults) MemoryDelta() int64 {
	return int64(br.MemoryAfter.Alloc) - int64(br.MemoryBefore.Alloc)
}

// Report generates a detailed report of the benchmark results
func (br *BenchmarkResults) Report() string {
	report := fmt.Sprintf("Benchmark Results: %s\n", br.Name)
	report += fmt.Sprintf("==========================================\n")
	report += fmt.Sprintf("Iterations: %d\n", br.Iterations)
	report += fmt.Sprintf("Warmup: %d\n", br.Warmup)
	report += fmt.Sprintf("Parallel: %t\n", br.Parallel)
	report += fmt.Sprintf("Total Duration: %v\n", br.TotalDuration)
	report += fmt.Sprintf("Average Duration: %v\n", br.AverageDuration())
	report += fmt.Sprintf("Min Duration: %v\n", br.MinDuration())
	report += fmt.Sprintf("Max Duration: %v\n", br.MaxDuration())
	report += fmt.Sprintf("Memory Delta: %+d bytes\n", br.MemoryDelta())
	report += fmt.Sprintf("GC Cycles: %d\n", br.MemoryAfter.NumGC-br.MemoryBefore.NumGC)

	return report
}

// CompareResults compares two benchmark results and returns a comparison report
func CompareResults(baseline, comparison *BenchmarkResults) string {
	report := fmt.Sprintf("Benchmark Comparison\n")
	report += fmt.Sprintf("==========================================\n")
	report += fmt.Sprintf("Baseline: %s\n", baseline.Name)
	report += fmt.Sprintf("Comparison: %s\n", comparison.Name)
	report += fmt.Sprintf("\n")

	// Duration comparison
	baselineAvg := baseline.AverageDuration()
	comparisonAvg := comparison.AverageDuration()

	if baselineAvg > 0 {
		improvement := float64(baselineAvg-comparisonAvg) / float64(baselineAvg) * 100
		report += fmt.Sprintf("Average Duration:\n")
		report += fmt.Sprintf("  Baseline: %v\n", baselineAvg)
		report += fmt.Sprintf("  Comparison: %v\n", comparisonAvg)
		report += fmt.Sprintf("  Improvement: %.2f%%\n", improvement)
	}

	// Memory comparison
	baselineMemory := baseline.MemoryDelta()
	comparisonMemory := comparison.MemoryDelta()

	if baselineMemory != 0 {
		memoryImprovement := float64(baselineMemory-comparisonMemory) / float64(baselineMemory) * 100
		report += fmt.Sprintf("Memory Usage:\n")
		report += fmt.Sprintf("  Baseline: %+d bytes\n", baselineMemory)
		report += fmt.Sprintf("  Comparison: %+d bytes\n", comparisonMemory)
		report += fmt.Sprintf("  Improvement: %.2f%%\n", memoryImprovement)
	}

	return report
}

// BenchmarkFunc is a helper function for running simple benchmarks in tests
func BenchmarkFunc(t *testing.T, name string, fn func()) *BenchmarkResults {
	t.Helper()

	runner := NewBenchmarkRunner().WithIterations(100).WithWarmup(10)
	results := runner.Run(name, fn)

	t.Logf("Benchmark %s completed: avg=%v, min=%v, max=%v, memory=%+d bytes",
		name, results.AverageDuration(), results.MinDuration(), results.MaxDuration(), results.MemoryDelta())

	return results
}
