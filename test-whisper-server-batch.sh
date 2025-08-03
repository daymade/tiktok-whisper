#!/bin/bash

# Test whisper-server batch conversion
set -e

WHISPER_SERVER_URL="http://mac-mini-m4-1.local:8080"
TEST_FILES=(
    "./test/data/test_16khz.wav"
    "./test/data/output_16khz.wav"
    "./test/data/jfk.wav"
)
OUTPUT_DIR="./data/transcription/whisper-server-test"

echo "Testing whisper-server batch conversion..."
echo "Server URL: $WHISPER_SERVER_URL"
echo "Test files: ${#TEST_FILES[@]}"
echo

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Process each file
for file in "${TEST_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        filename=$(basename "$file")
        output_file="$OUTPUT_DIR/${filename%.wav}.txt"
        
        echo "Processing: $filename"
        
        # Send request to whisper-server
        response=$(curl -s -X POST \
            -F "file=@$file" \
            -F "language=zh" \
            -F "response_format=text" \
            "$WHISPER_SERVER_URL/inference" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            echo "$response" > "$output_file"
            echo "✅ Saved to: $output_file"
            echo "   Transcription: $(echo "$response" | head -c 100)..."
        else
            echo "❌ Failed: $response"
        fi
        echo
    else
        echo "⚠️  File not found: $file"
    fi
done

echo "Batch conversion complete!"
echo "Results saved to: $OUTPUT_DIR"