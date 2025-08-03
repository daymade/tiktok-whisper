#!/bin/bash

# Test script for whisper-server HTTP provider integration
# This script tests the whisper-server provider with real instances

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REMOTE_HOST="daymade@mac-mini-m4-1.local"
WHISPER_DIR="/Users/daymade/Workspace/cpp/whisper.cpp"
WHISPER_SERVER_PORT="8080"
LOCAL_PORT="8080"

# Test audio file (will be created if it doesn't exist)
TEST_AUDIO_DIR="./test/data"
TEST_AUDIO_FILE="$TEST_AUDIO_DIR/test_whisper_server.wav"

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE} Whisper-Server HTTP Provider Integration Test${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo
}

print_step() {
    echo -e "${YELLOW}[STEP] $1${NC}"
}

print_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

print_error() {
    echo -e "${RED}[ERROR] $1${NC}"
}

print_info() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

# Function to create test audio file if it doesn't exist
create_test_audio() {
    if [[ ! -f "$TEST_AUDIO_FILE" ]]; then
        print_step "Creating test audio file..."
        mkdir -p "$TEST_AUDIO_DIR"
        
        # Create a simple WAV file using ffmpeg (if available)
        if command -v ffmpeg >/dev/null 2>&1; then
            # Generate 5 seconds of sine wave at 440Hz
            ffmpeg -f lavfi -i "sine=frequency=440:duration=5" -ar 16000 -ac 1 "$TEST_AUDIO_FILE" -y >/dev/null 2>&1
            print_success "Created test audio file with ffmpeg"
        else
            # Create a minimal WAV header + some data
            echo -e "\x52\x49\x46\x46\x24\x00\x00\x00\x57\x41\x56\x45\x66\x6d\x74\x20\x10\x00\x00\x00\x01\x00\x01\x00\x40\x1f\x00\x00\x80\x3e\x00\x00\x02\x00\x10\x00\x64\x61\x74\x61\x00\x00\x00\x00" > "$TEST_AUDIO_FILE"
            print_success "Created minimal test audio file"
        fi
    else
        print_info "Test audio file already exists: $TEST_AUDIO_FILE"
    fi
}

# Function to check if remote whisper-server is accessible
check_remote_whisper_server() {
    print_step "Checking if whisper-server is running on $REMOTE_HOST..."
    
    # First, test SSH connectivity
    if ! ssh -o ConnectTimeout=5 -q "$REMOTE_HOST" exit; then
        print_error "Cannot connect to $REMOTE_HOST via SSH"
        return 1
    fi
    
    # Check if whisper.cpp directory exists
    if ! ssh "$REMOTE_HOST" "test -d $WHISPER_DIR"; then
        print_error "Whisper.cpp directory not found on remote host: $WHISPER_DIR"
        return 1
    fi
    
    # Check if whisper-server binary exists
    if ! ssh "$REMOTE_HOST" "test -f $WHISPER_DIR/build/bin/whisper-server"; then
        print_error "whisper-server binary not found: $WHISPER_DIR/build/bin/whisper-server"
        print_info "You may need to build whisper.cpp with server support:"
        print_info "  ssh $REMOTE_HOST"
        print_info "  cd $WHISPER_DIR"
        print_info "  cmake -B build -DWHISPER_BUILD_SERVER=ON"
        print_info "  cmake --build build -j"
        return 1
    fi
    
    # Check if server is already running
    if ssh "$REMOTE_HOST" "pgrep -f whisper-server" >/dev/null 2>&1; then
        print_success "whisper-server is already running on remote host"
        
        # Test if it's accessible via HTTP
        if curl -s -m 5 "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" >/dev/null 2>&1; then
            print_success "whisper-server is accessible via HTTP"
            return 0
        else
            print_error "whisper-server is running but not accessible via HTTP"
            return 1
        fi
    else
        print_info "whisper-server is not running, attempting to start it..."
        return 2
    fi
}

# Function to start whisper-server on remote host
start_remote_whisper_server() {
    print_step "Starting whisper-server on $REMOTE_HOST..."
    
    # Check for model files
    ssh "$REMOTE_HOST" "ls $WHISPER_DIR/models/ggml-*.bin" >/dev/null 2>&1 || {
        print_error "No whisper models found on remote host"
        print_info "Please download models first:"
        print_info "  ssh $REMOTE_HOST"
        print_info "  cd $WHISPER_DIR"
        print_info "  bash ./models/download-ggml-model.sh base.en"
        return 1
    }
    
    # Find the first available model
    MODEL_FILE=$(ssh "$REMOTE_HOST" "ls $WHISPER_DIR/models/ggml-base*.bin | head -1" 2>/dev/null || echo "")
    if [[ -z "$MODEL_FILE" ]]; then
        MODEL_FILE=$(ssh "$REMOTE_HOST" "ls $WHISPER_DIR/models/ggml-*.bin | head -1" 2>/dev/null || echo "")
    fi
    
    if [[ -z "$MODEL_FILE" ]]; then
        print_error "No suitable model file found"
        return 1
    fi
    
    print_info "Using model: $(basename "$MODEL_FILE")"
    
    # Start whisper-server in background
    ssh "$REMOTE_HOST" "cd $WHISPER_DIR && nohup ./build/bin/whisper-server --host 0.0.0.0 --port $WHISPER_SERVER_PORT --model $MODEL_FILE > whisper-server.log 2>&1 &"
    
    # Wait for server to start
    print_info "Waiting for server to start..."
    for i in {1..30}; do
        if curl -s -m 5 "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" >/dev/null 2>&1; then
            print_success "whisper-server started successfully"
            return 0
        fi
        sleep 1
        echo -n "."
    done
    
    echo
    print_error "Failed to start whisper-server or server is not responding"
    
    # Show server logs for debugging
    print_info "Server logs:"
    ssh "$REMOTE_HOST" "cd $WHISPER_DIR && tail -20 whisper-server.log" || true
    
    return 1
}

# Function to test local whisper-server (if available)
test_local_whisper_server() {
    print_step "Testing local whisper-server (if available)..."
    
    if curl -s -m 5 "http://127.0.0.1:$LOCAL_PORT" >/dev/null 2>&1; then
        print_success "Local whisper-server is accessible"
        
        # Build and run test client
        print_step "Building test client..."
        cd "$(dirname "$0")/.."
        go build -o test-whisper-server-client ./cmd/test-whisper-server/
        
        print_step "Testing local whisper-server with test client..."
        ./test-whisper-server-client -url "http://127.0.0.1:$LOCAL_PORT" -file "$TEST_AUDIO_FILE" -verbose
        
        print_success "Local whisper-server test completed"
    else
        print_info "Local whisper-server not available, skipping local test"
    fi
}

# Function to test remote whisper-server
test_remote_whisper_server() {
    print_step "Testing remote whisper-server..."
    
    # Build and run test client
    print_step "Building test client..."
    cd "$(dirname "$0")/.."
    go build -o test-whisper-server-client ./cmd/test-whisper-server/
    
    print_step "Testing remote whisper-server with test client..."
    ./test-whisper-server-client -url "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" -file "$TEST_AUDIO_FILE" -verbose -timeout 180
    
    print_success "Remote whisper-server test completed"
}

# Function to test provider integration
test_provider_integration() {
    print_step "Testing provider integration..."
    
    # Test basic provider functionality
    print_info "Testing basic transcription..."
    ./test-whisper-server-client -url "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" -file "$TEST_AUDIO_FILE" -format "text"
    
    print_info "Testing JSON response format..."
    ./test-whisper-server-client -url "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" -file "$TEST_AUDIO_FILE" -format "json"
    
    print_info "Testing verbose JSON response format..."
    ./test-whisper-server-client -url "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" -file "$TEST_AUDIO_FILE" -format "verbose_json"
    
    print_info "Testing SRT subtitle format..."
    ./test-whisper-server-client -url "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" -file "$TEST_AUDIO_FILE" -format "srt"
    
    print_success "Provider integration tests completed"
}

# Function to run performance tests
test_performance() {
    print_step "Running performance tests..."
    
    print_info "Testing response times..."
    for i in {1..3}; do
        echo "  Run $i:"
        ./test-whisper-server-client -url "http://$REMOTE_HOST:$WHISPER_SERVER_PORT" -file "$TEST_AUDIO_FILE" -format "json" | grep -E "(Health check|transcription|Processing Time)"
    done
    
    print_success "Performance tests completed"
}

# Function to cleanup
cleanup() {
    print_step "Cleaning up..."
    
    # Remove test client binary
    if [[ -f "./test-whisper-server-client" ]]; then
        rm -f "./test-whisper-server-client"
        print_info "Removed test client binary"
    fi
    
    # Optionally stop remote whisper-server (uncomment if desired)
    # print_info "Stopping remote whisper-server..."
    # ssh "$REMOTE_HOST" "pkill -f whisper-server" || true
    
    print_success "Cleanup completed"
}

# Main execution
main() {
    print_header
    
    # Create test audio file
    create_test_audio
    
    # Check if we can run Go commands
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check remote whisper-server
    check_remote_whisper_server
    server_status=$?
    
    if [[ $server_status -eq 1 ]]; then
        print_error "Cannot proceed with remote server tests"
        exit 1
    elif [[ $server_status -eq 2 ]]; then
        # Try to start the server
        if ! start_remote_whisper_server; then
            print_error "Failed to start remote whisper-server"
            exit 1
        fi
    fi
    
    # Test local server (optional)
    test_local_whisper_server || true
    
    # Test remote server
    test_remote_whisper_server
    
    # Test provider integration
    test_provider_integration
    
    # Performance tests
    test_performance
    
    # Cleanup
    cleanup
    
    print_header
    print_success "All whisper-server HTTP provider tests passed!"
    print_info "The whisper-server HTTP provider is ready for production use."
    echo
}

# Handle script interruption
trap cleanup EXIT

# Run main function
main "$@"