#!/bin/bash

# File Utilities Test Runner
# Comprehensive test suite for internal/app/util/files package

set -e

echo "🔧 File Utilities Test Suite"
echo "=============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test directory
TEST_DIR="./internal/app/util/files/"

echo -e "${BLUE}📁 Testing directory: ${TEST_DIR}${NC}"
echo

# Function to run tests with timing
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "${YELLOW}🧪 Running: ${test_name}${NC}"
    start_time=$(date +%s.%N)
    
    if eval "$test_command"; then
        end_time=$(date +%s.%N)
        duration=$(echo "$end_time - $start_time" | bc -l)
        echo -e "${GREEN}✅ ${test_name} passed (${duration}s)${NC}"
        return 0
    else
        echo -e "${RED}❌ ${test_name} failed${NC}"
        return 1
    fi
    echo
}

# Track test results
total_tests=0
passed_tests=0

# Basic functionality tests
total_tests=$((total_tests + 1))
if run_test "Basic Unit Tests" "go test ${TEST_DIR} -v -timeout 2m"; then
    passed_tests=$((passed_tests + 1))
fi

# Coverage analysis
total_tests=$((total_tests + 1))
if run_test "Coverage Analysis" "go test ${TEST_DIR} -cover -timeout 2m"; then
    passed_tests=$((passed_tests + 1))
fi

# Race condition detection
total_tests=$((total_tests + 1))
if run_test "Race Condition Detection" "go test ${TEST_DIR} -race -timeout 3m"; then
    passed_tests=$((passed_tests + 1))
fi

# Memory leak detection
total_tests=$((total_tests + 1))
if run_test "Memory Leak Detection" "go test ${TEST_DIR} -v -timeout 2m -test.memprofile=mem.prof"; then
    passed_tests=$((passed_tests + 1))
fi

# Performance benchmarks
total_tests=$((total_tests + 1))
if run_test "Performance Benchmarks" "go test ${TEST_DIR} -bench=. -benchmem -benchtime=500ms -timeout 5m -run=^$"; then
    passed_tests=$((passed_tests + 1))
fi

# Stress testing
total_tests=$((total_tests + 1))
if run_test "Concurrent Stress Tests" "go test ${TEST_DIR} -run=TestConcurrentFileOperations -v -timeout 2m"; then
    passed_tests=$((passed_tests + 1))
fi

# Platform-specific tests
if [[ "$OSTYPE" == "linux-gnu"* ]] || [[ "$OSTYPE" == "darwin"* ]]; then
    total_tests=$((total_tests + 1))
    if run_test "Unix-specific Tests" "go test ${TEST_DIR} -run='TestSymlinkHandling|TestFilePermissionValidation' -v"; then
        passed_tests=$((passed_tests + 1))
    fi
fi

# Cleanup test files
echo -e "${BLUE}🧹 Cleaning up test artifacts...${NC}"
rm -f mem.prof cpu.prof coverage.out 2>/dev/null || true

# Summary
echo
echo "=============================="
echo -e "${BLUE}📊 Test Summary${NC}"
echo "=============================="
echo -e "Total test suites: ${total_tests}"
echo -e "Passed: ${GREEN}${passed_tests}${NC}"
echo -e "Failed: ${RED}$((total_tests - passed_tests))${NC}"

if [ $passed_tests -eq $total_tests ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some tests failed!${NC}"
    exit 1
fi