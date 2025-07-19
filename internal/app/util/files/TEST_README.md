# File Utilities Test Suite

This directory contains comprehensive unit tests for the file utilities in `internal/app/util/files/`. The test suite provides extensive coverage of file system operations, path management, and cross-platform compatibility.

## Test Files Overview

### 1. `FileUtils_test.go` (Enhanced)
Core functionality tests for the main file utilities:
- **GetProjectRoot()** - Project root detection from go.mod
- **GetAbsolutePath()** - Path resolution (relative to absolute)
- **GetUserMp3Dir()** - User-specific MP3 directory paths
- **CheckAndCreateMP3Directory()** - Directory creation with validation
- **GetAllFiles()** - File enumeration with extension filtering
- **ReadOutputFile()** - File content reading with whitespace trimming
- **WriteToFile()** - File writing with directory creation
- **findGoModRoot()** - Go module root detection algorithm

### 2. `path_test.go`
Advanced path handling and normalization tests:
- **Path Normalization** - Handling different path separators and formats
- **Unicode Path Support** - Chinese, Japanese, Korean, Arabic, emoji paths
- **Long Path Handling** - Platform-specific path length limitations
- **Special Characters** - Spaces, symbols, punctuation in paths
- **Relative Path Resolution** - Complex relative path scenarios
- **Cross-Platform Separators** - Windows vs Unix path handling
- **Case Sensitivity** - Platform-dependent file system behaviors
- **Symlink Handling** - Symbolic link resolution (Unix/macOS)

### 3. `directory_test.go`
Directory operations and management tests:
- **Advanced Directory Creation** - Nested, concurrent, permission scenarios
- **Directory Traversal** - Complex directory structure navigation
- **Permission Testing** - Read/write/execute permission validation
- **Directory Cleanup** - Safe removal of test directories
- **Modification Time** - Directory and file timestamp handling
- **Size Calculation** - Directory and file size computation
- **Concurrent Operations** - Thread-safe directory operations

### 4. `validation_test.go`
File validation and edge case handling:
- **Extension Validation** - Case-insensitive extension matching
- **File Size Validation** - Empty to very large file handling
- **Permission Validation** - File accessibility testing
- **Special File Types** - Symlinks, broken links, device files
- **File Name Validation** - Unicode, emoji, special character names
- **Content Validation** - Binary, text, Unicode content handling
- **Concurrent File Operations** - Thread-safe file access

### 5. `benchmark_test.go`
Performance and stress testing:
- **GetAbsolutePath Performance** - Path resolution benchmarks
- **File Enumeration Performance** - Large directory scanning
- **File I/O Performance** - Read/write operations with various sizes
- **Directory Creation Performance** - Bulk directory operations
- **Memory Usage** - Memory efficiency of file operations
- **Concurrent Performance** - Multi-threaded operation benchmarks
- **File System Stress Tests** - Mixed operation workloads

## Running Tests

### Basic Test Execution
```bash
# Run all tests
go test ./internal/app/util/files/

# Run with verbose output
go test -v ./internal/app/util/files/

# Run specific test file
go test -v ./internal/app/util/files/ -run TestGetAbsolutePath
```

### Benchmark Execution
```bash
# Run all benchmarks
go test -bench=. ./internal/app/util/files/

# Run specific benchmark with memory stats
go test -bench=BenchmarkGetAbsolutePath -benchmem ./internal/app/util/files/

# Run benchmarks for specific time
go test -bench=. -benchtime=5s ./internal/app/util/files/
```

### Coverage Analysis
```bash
# Generate coverage report
go test -cover ./internal/app/util/files/

# Generate detailed coverage
go test -coverprofile=coverage.out ./internal/app/util/files/
go tool cover -html=coverage.out
```

## Test Categories

### 1. Functional Tests
- **Core Functionality** - Basic operations work as expected
- **Edge Cases** - Boundary conditions and error scenarios
- **Integration** - Functions work together correctly

### 2. Cross-Platform Tests
- **Windows Compatibility** - Drive letters, backslashes, long paths
- **Unix/Linux Compatibility** - Forward slashes, case sensitivity
- **macOS Compatibility** - Case-insensitive file system, permissions

### 3. Performance Tests
- **Scalability** - Performance with large numbers of files
- **Memory Efficiency** - Memory usage optimization
- **Concurrency** - Thread-safe operations

### 4. Security Tests
- **Path Traversal** - Prevention of directory traversal attacks
- **Permission Validation** - Proper permission checking
- **Input Validation** - Safe handling of malicious input

## Test Data

The tests use several sources of test data:
- **Temporary Directories** - Created and cleaned up automatically
- **Real Audio Files** - From `test/data/audio/` directory
- **Generated Content** - Various file sizes and types
- **Unicode Content** - Multi-language and emoji testing

## Platform-Specific Considerations

### Windows
- Path length limitations (260 characters without long path support)
- Drive letter handling (C:\, D:\, etc.)
- Backslash separators
- Case-insensitive file system
- Different permission model

### Unix/Linux
- Forward slash separators
- Case-sensitive file system
- Symlink support
- Unix permissions (rwx)
- No drive letters

### macOS
- Forward slash separators
- Case-insensitive but case-preserving file system
- Symlink support
- Unix permissions
- HFS+ specific behaviors

## Error Scenarios Tested

### File System Errors
- **Permission Denied** - Insufficient file/directory permissions
- **File Not Found** - Missing files and directories
- **Disk Full** - Out of space conditions (simulated)
- **Path Too Long** - Exceeding platform path limits
- **Invalid Characters** - Platform-specific forbidden characters

### Application Errors
- **Invalid Input** - Null, empty, or malformed paths
- **Race Conditions** - Concurrent access conflicts
- **Resource Leaks** - Proper cleanup of file handles
- **Unicode Issues** - Encoding/decoding problems

## Test Maintenance

### Adding New Tests
1. Follow existing naming conventions
2. Use table-driven tests for multiple scenarios
3. Include both positive and negative test cases
4. Add platform-specific tests when needed
5. Update this documentation

### Test Cleanup
- All tests use temporary directories that are automatically cleaned up
- Manual cleanup is handled in defer statements
- Permission restoration for modified directories

### Continuous Integration
These tests are designed to run in CI environments:
- No external dependencies
- Deterministic results
- Platform-independent where possible
- Reasonable execution time

## Performance Baselines

Typical performance expectations on modern hardware:
- **GetAbsolutePath**: < 2μs for absolute paths, ~2ms for relative paths
- **File Enumeration**: ~1ms per 100 files
- **File I/O**: ~10MB/s for sequential operations
- **Directory Creation**: ~100 directories/ms

These benchmarks help detect performance regressions and optimize critical paths.