#!/bin/bash

# Script to refactor tests into unit and integration tests

echo "Starting test refactoring..."

# Add build tags to existing integration test files
echo "Adding build tags to integration test files..."

# Find all files with "integration" in the name or containing integration tests
find . -name "*_integration_test.go" -o -name "*integration_test.go" | while read -r file; do
    # Check if file already has build tags
    if ! grep -q "//go:build integration\|// +build integration" "$file"; then
        echo "Adding build tags to $file"
        # Create temp file with build tags
        {
            echo "//go:build integration"
            echo "// +build integration"
            echo ""
            cat "$file"
        } > "$file.tmp"
        mv "$file.tmp" "$file"
    fi
done

# Find test files that use PostgreSQL directly
echo "Finding tests that need PostgreSQL..."
grep -r "postgres://" --include="*_test.go" . | grep -v "integration_test.go" | cut -d: -f1 | sort -u | while read -r file; do
    echo "Found PostgreSQL usage in: $file"
    # These files should either:
    # 1. Be converted to use mocks (unit tests)
    # 2. Be moved to integration tests
    # 3. Already use testing.Short() to skip when running unit tests
done

echo "Test refactoring complete!"
echo ""
echo "Next steps:"
echo "1. Review the files listed above that use PostgreSQL"
echo "2. Convert them to use mocks or move to integration tests"
echo "3. Run 'make test' to verify unit tests pass without external dependencies"
echo "4. Run 'make test-integration-go' to verify integration tests pass with PostgreSQL"