#!/bin/bash
# Integration test suite for tiktok-whisper
# This script runs comprehensive tests to verify all components work correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Configuration
TEST_DIR="./test/integration_$(date +%Y%m%d_%H%M%S)"
DB_PATH="./data/transcription.db"
TEST_USER="integration_test_user"
CLEANUP=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-cleanup)
            CLEANUP=false
            shift
            ;;
        --verbose)
            set -x
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--no-cleanup] [--verbose]"
            exit 1
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

test_passed() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

test_failed() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

test_skipped() {
    echo -e "${YELLOW}⊖${NC} $1 (skipped)"
    ((TESTS_SKIPPED++))
}

# Cleanup function
cleanup() {
    if [ "$CLEANUP" = true ] && [ -n "$TEST_DIR" ]; then
        log_info "Cleaning up test artifacts..."
        [ -d "$TEST_DIR" ] && rm -rf "$TEST_DIR"
        # Clean test records from database if it exists
        if [ -f "$DB_PATH" ]; then
            sqlite3 "$DB_PATH" "DELETE FROM transcriptions WHERE user = '$TEST_USER';" 2>/dev/null || true
        fi
    else
        log_info "Cleanup disabled, test artifacts preserved in: $TEST_DIR"
    fi
}

# Set up trap for cleanup only after setup completes
# (Will be set after setup_tests)

# Test setup
setup_tests() {
    log_info "Setting up test environment..."
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    
    # Build the application
    log_info "Building v2t..."
    if CGO_ENABLED=1 go build -o v2t ./cmd/v2t/main.go; then
        test_passed "Build successful"
    else
        test_failed "Build failed"
        exit 1
    fi
    
    # Check database exists
    if [ -f "$DB_PATH" ]; then
        test_passed "Database exists"
    else
        test_failed "Database not found at $DB_PATH"
        exit 1
    fi
}

# Test 1: Database Schema Validation
test_database_schema() {
    echo ""
    log_info "Test 1: Database Schema Validation"
    
    # Check if all expected columns exist
    EXPECTED_COLUMNS="id user input_dir file_name mp3_file_name audio_duration transcription last_conversion_time has_error error_message file_hash file_size provider_type language model_name created_at updated_at deleted_at"
    
    ACTUAL_COLUMNS=$(sqlite3 "$DB_PATH" "PRAGMA table_info(transcriptions);" | awk -F'|' '{print $2}' | tr '\n' ' ')
    
    ALL_COLUMNS_EXIST=true
    for col in $EXPECTED_COLUMNS; do
        if [[ ! " $ACTUAL_COLUMNS " =~ " $col " ]]; then
            test_failed "Missing column: $col"
            ALL_COLUMNS_EXIST=false
        fi
    done
    
    if [ "$ALL_COLUMNS_EXIST" = true ]; then
        test_passed "All expected columns present"
    fi
    
    # Check indexes
    INDEX_COUNT=$(sqlite3 "$DB_PATH" ".indexes transcriptions" | wc -l)
    if [ "$INDEX_COUNT" -ge 7 ]; then
        test_passed "All indexes present ($INDEX_COUNT found)"
    else
        test_failed "Missing indexes (only $INDEX_COUNT found, expected 7+)"
    fi
}

# Test 2: Query Performance
test_query_performance() {
    echo ""
    log_info "Test 2: Query Performance"
    
    # Test indexed query
    START_TIME=$(date +%s.%N)
    sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM transcriptions WHERE user = 'test' AND has_error = 0;" > /dev/null
    END_TIME=$(date +%s.%N)
    
    QUERY_TIME=$(echo "$END_TIME - $START_TIME" | bc)
    if (( $(echo "$QUERY_TIME < 0.1" | bc -l) )); then
        test_passed "Query performance acceptable (${QUERY_TIME}s)"
    else
        test_failed "Query too slow (${QUERY_TIME}s, expected < 0.1s)"
    fi
    
    # Verify index usage
    QUERY_PLAN=$(sqlite3 "$DB_PATH" "EXPLAIN QUERY PLAN SELECT * FROM transcriptions WHERE file_name = 'test.mp3' AND has_error = 0;" 2>&1)
    if [[ "$QUERY_PLAN" =~ "USING INDEX" ]]; then
        test_passed "Query uses index correctly"
    else
        test_failed "Query not using index"
    fi
}

# Test 3: Provider Framework
test_provider_framework() {
    echo ""
    log_info "Test 3: Provider Framework"
    
    # Test provider list command
    if ./v2t providers list > /dev/null 2>&1; then
        test_passed "Provider list command works"
    else
        test_failed "Provider list command failed"
    fi
    
    # Check provider configuration
    if [ -f "$HOME/.tiktok-whisper/providers.yaml" ]; then
        test_passed "Provider configuration exists"
        
        # Check whisper_cpp provider
        if grep -q "whisper_cpp:" "$HOME/.tiktok-whisper/providers.yaml"; then
            test_passed "whisper_cpp provider configured"
        else
            test_failed "whisper_cpp provider not configured"
        fi
    else
        test_skipped "Provider configuration not found"
    fi
}

# Test 4: Basic Conversion
test_basic_conversion() {
    echo ""
    log_info "Test 4: Basic Conversion"
    
    # Create a test audio file
    TEST_AUDIO="$TEST_DIR/test_audio.wav"
    
    # Try to create test audio with ffmpeg
    if command -v ffmpeg &> /dev/null; then
        if ffmpeg -f lavfi -i "sine=frequency=440:duration=2" -ar 16000 "$TEST_AUDIO" -y &> /dev/null; then
            test_passed "Test audio file created"
        else
            test_skipped "Could not create test audio file"
            return
        fi
    else
        test_skipped "ffmpeg not available for audio generation"
        return
    fi
    
    # Test single file conversion
    if ./v2t convert single -a -i "$TEST_AUDIO" -u "$TEST_USER" -o "$TEST_DIR" > /dev/null 2>&1; then
        test_passed "Single file conversion successful"
        
        # Check if record was created in database
        RECORD_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM transcriptions WHERE user = '$TEST_USER';" 2>/dev/null)
        if [ "$RECORD_COUNT" -gt 0 ]; then
            test_passed "Record saved to database"
        else
            test_failed "No record found in database"
        fi
    else
        test_failed "Single file conversion failed"
    fi
}

# Test 5: Database Features
test_database_features() {
    echo ""
    log_info "Test 5: Database Features"
    
    # Test new fields
    RECORD=$(sqlite3 "$DB_PATH" "SELECT provider_type, language, created_at FROM transcriptions WHERE user = '$TEST_USER' LIMIT 1;" 2>/dev/null)
    
    if [ -n "$RECORD" ]; then
        test_passed "New fields populated"
    else
        test_skipped "No test records to verify"
    fi
    
    # Test provider type distribution
    PROVIDER_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(DISTINCT provider_type) FROM transcriptions;" 2>/dev/null)
    if [ "$PROVIDER_COUNT" -ge 1 ]; then
        test_passed "Provider types recorded"
    else
        test_failed "No provider types found"
    fi
}

# Test 6: Embedding System
test_embedding_system() {
    echo ""
    log_info "Test 6: Embedding System"
    
    # Test embed status command
    if ./v2t embed status > /dev/null 2>&1; then
        test_passed "Embedding status command works"
    else
        test_failed "Embedding status command failed"
    fi
    
    # Check if API keys are configured
    if [ -n "$OPENAI_API_KEY" ] || [ -n "$GEMINI_API_KEY" ]; then
        test_passed "Embedding API keys configured"
    else
        test_skipped "No embedding API keys configured"
    fi
}

# Test 7: Web Interface
test_web_interface() {
    echo ""
    log_info "Test 7: Web Interface"
    
    # Start web server in background
    ./v2t web --port :8082 > /dev/null 2>&1 &
    WEB_PID=$!
    
    # Give it time to start
    sleep 2
    
    # Check if server is running
    if kill -0 $WEB_PID 2>/dev/null; then
        test_passed "Web server started"
        
        # Try to access health endpoint
        if command -v curl &> /dev/null; then
            if curl -s -f http://localhost:8082/health > /dev/null 2>&1; then
                test_passed "Health endpoint accessible"
            else
                test_failed "Health endpoint not accessible"
            fi
        else
            test_skipped "curl not available for HTTP test"
        fi
        
        # Stop web server
        kill $WEB_PID 2>/dev/null || true
    else
        test_failed "Web server failed to start"
    fi
}

# Test 8: Export Functionality
test_export_functionality() {
    echo ""
    log_info "Test 8: Export Functionality"
    
    # Test export command
    if ./v2t export list > /dev/null 2>&1; then
        test_passed "Export list command works"
    else
        test_failed "Export list command failed"
    fi
}

# Main test execution
main() {
    echo "=== TikTok-Whisper Integration Test Suite ==="
    echo "Started at: $(date)"
    echo ""
    
    # Setup
    setup_tests
    
    # Now set up the cleanup trap
    trap cleanup EXIT
    
    # Run tests
    test_database_schema
    test_query_performance
    test_provider_framework
    test_basic_conversion
    test_database_features
    test_embedding_system
    test_web_interface
    test_export_functionality
    
    # Summary
    echo ""
    echo "=== Test Summary ==="
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED"
    echo -e "${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
    echo ""
    
    if [ "$TESTS_FAILED" -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Run main
main