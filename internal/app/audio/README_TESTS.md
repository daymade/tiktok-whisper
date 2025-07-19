# Audio Processing Unit Tests

This directory contains comprehensive unit tests for the audio processing utilities in the `tiktok-whisper` application.

## Test Files Overview

### Core Test Files

- **`audio_test.go`** - Core unit tests for audio processing logic
- **`audio_simple_test.go`** - Additional utility and validation tests
- **`audio_integration_test.go`** - Integration tests requiring FFmpeg (build tag: `integration`)
- **`benchmark_simple_test.go`** - Performance benchmarks

## Test Categories

### 1. Unit Tests (`audio_test.go` & `audio_simple_test.go`)

These tests focus on the **logic and algorithms** without requiring external dependencies:

#### Duration Parsing Tests
- Tests for `parseDurationOutput()` function
- Validates parsing of FFprobe output strings
- Handles edge cases (invalid formats, whitespace, etc.)
- Tests mathematical rounding logic

#### Format Detection Tests  
- Tests for `parseProbeOutput()` function
- Validates FFprobe JSON parsing
- Tests 16kHz WAV file detection logic
- Handles malformed JSON and missing fields

#### Path Handling Tests
- File extension validation
- Output path generation
- Special characters and edge cases
- Cross-platform path handling

#### Command Argument Construction Tests
- FFmpeg command line argument validation
- Different conversion scenarios
- Error message formatting

### 2. Integration Tests (`audio_integration_test.go`)

These tests require FFmpeg to be available and use actual audio files:

```bash
# Run integration tests
go test ./internal/app/audio/ -tags=integration -v
```

#### Features Tested:
- Actual audio duration extraction
- Real file format detection
- Audio conversion workflows
- Error handling with real FFmpeg commands
- Performance measurements
- Concurrent operations

#### Test File Requirements:
The integration tests expect audio files in `test/data/audio/`:
- `short_sine_16khz.wav`
- `medium_sine_16khz.wav` 
- `short_sine_44khz.wav`
- `short_sine_22khz.mp3`
- `short_sine_48khz.m4a`
- `silence_5s.wav`

### 3. Benchmark Tests (`benchmark_simple_test.go`)

Performance tests for critical operations:

```bash
# Run benchmarks
go test ./internal/app/audio/ -bench=. -benchmem
```

#### Benchmarked Operations:
- Path manipulation performance
- String operations efficiency  
- Format validation speed
- Memory allocation patterns
- Concurrent vs sequential processing

## Running Tests

### Basic Unit Tests
```bash
# Run all unit tests
go test ./internal/app/audio/ -v

# Run with coverage
go test ./internal/app/audio/ -cover

# Run specific test
go test ./internal/app/audio/ -run TestGetAudioDuration -v
```

### Integration Tests
```bash
# Run integration tests (requires FFmpeg)
go test ./internal/app/audio/ -tags=integration -v

# Skip integration tests
go test ./internal/app/audio/ -short -v
```

### Benchmarks
```bash
# Run all benchmarks
go test ./internal/app/audio/ -bench=. -benchmem

# Run specific benchmark
go test ./internal/app/audio/ -bench=BenchmarkPathOperations

# Run benchmarks with CPU profiling
go test ./internal/app/audio/ -bench=. -cpuprofile=cpu.prof
```

## Test Design Philosophy

### Unit Test Approach

Rather than complex mocking of `exec.Command`, these tests focus on **testing the core logic** by extracting testable functions:

- `parseDurationOutput()` - Tests duration parsing logic
- `parseProbeOutput()` - Tests FFprobe JSON parsing logic
- Path manipulation logic - Tests file extension and path handling

This approach provides:
- ✅ **Reliability** - Tests don't depend on complex mocking infrastructure
- ✅ **Maintainability** - Simple test logic that's easy to understand
- ✅ **Fast execution** - No subprocess overhead
- ✅ **Comprehensive coverage** - Tests all critical logic paths

### Integration Test Approach

Integration tests handle the full workflow:
- Real FFmpeg command execution
- Actual file processing
- End-to-end validation
- Performance measurement

## Test Coverage

The test suite covers:

### Audio Format Handling ✅
- MP3, WAV, M4A format support
- Format validation logic
- Extension handling (case-insensitive)
- Unsupported format detection

### Path Processing ✅
- Output path generation
- Special characters in paths
- Cross-platform compatibility
- Extension manipulation

### FFmpeg Integration ✅
- Command argument construction
- Error handling and reporting
- Output parsing
- Process execution

### Error Scenarios ✅
- Invalid audio files
- Missing FFmpeg binary
- Corrupted file handling
- Network/permission issues

### Performance ✅
- Path operation benchmarks
- String manipulation performance
- Memory allocation patterns
- Concurrent processing efficiency

## Test Data

### Synthetic Test Data
The unit tests use synthetic data to ensure predictable, fast execution:
- Predefined FFprobe JSON outputs
- Mock duration strings
- Test file paths

### Real Test Data (Integration)
Integration tests can use real audio files when available:
- Various sample rates (16kHz, 44kHz, 48kHz)
- Different formats (WAV, MP3, M4A)
- Different durations (short, medium, long)

## Continuous Integration

### Test Commands for CI
```bash
# Unit tests only (no FFmpeg required)
go test ./internal/app/audio/ -short

# Full test suite (requires FFmpeg)
go test ./internal/app/audio/ -tags=integration

# Benchmarks for performance regression detection
go test ./internal/app/audio/ -bench=. -benchtime=1s
```

### Test Requirements
- **Unit tests**: No external dependencies
- **Integration tests**: FFmpeg binary in PATH
- **Benchmarks**: Stable environment for consistent results

## Contributing

When adding new tests:

1. **Unit tests** for new logic/algorithms
2. **Integration tests** for new FFmpeg workflows  
3. **Benchmarks** for performance-critical code
4. **Error handling tests** for new failure modes

### Test Naming Conventions
- `TestFunctionName` - Basic functionality
- `TestFunctionNameEdgeCases` - Edge cases and error conditions
- `TestFunctionNameIntegration` - Integration tests
- `BenchmarkFunctionName` - Performance benchmarks

### Test File Organization
- Keep unit tests focused and fast
- Use build tags for integration tests
- Include helpful test descriptions
- Add benchmark comparisons for performance changes

## Example Test Output

```bash
$ go test ./internal/app/audio/ -v
=== RUN   TestGetAudioDuration
=== RUN   TestGetAudioDuration/valid_duration_-_integer_seconds
=== RUN   TestGetAudioDuration/valid_duration_-_decimal_seconds
--- PASS: TestGetAudioDuration (0.00s)
=== RUN   TestAudioFileExtensions  
--- PASS: TestAudioFileExtensions (0.00s)
=== RUN   TestPathHandling
--- PASS: TestPathHandling (0.00s)
PASS
ok      tiktok-whisper/internal/app/audio    0.428s
```

```bash
$ go test ./internal/app/audio/ -bench=. -benchmem
BenchmarkPathOperations/TrimSuffix-8           43758820    26.02 ns/op       0 B/op       0 allocs/op
BenchmarkPathOperations/CombineOperations-8     7821340   146.7 ns/op     288 B/op       4 allocs/op
BenchmarkDurationParsing-8                       3284004   360.6 ns/op       0 B/op       0 allocs/op
PASS
ok      tiktok-whisper/internal/app/audio    18.769s
```

This comprehensive test suite ensures the audio processing utilities are robust, performant, and reliable across different environments and use cases.