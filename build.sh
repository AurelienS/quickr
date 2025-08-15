#!/usr/bin/env bash
set -euo pipefail

# Load .env if present
if [ -f .env ]; then
	set -a
	source .env
	set +a
fi

CMD=${1:-}
case "$CMD" in
	reset-db)
		printf "> Resetting local SQLite database...\n"
		rm -f data/quickr.db
		mkdir -p data
		printf "Done.\n"
		;;
	*)
		echo "Usage: $0 {reset-db}"
		;;
 esac

echo "Building quickr..."
go build -o quickr

echo "Testing embedded resources..."
# Test that the binary exists
if [ ! -f quickr ]; then
    echo "Error: Binary not created"
    exit 1
fi

# Create a test directory
TEST_DIR="test_run"
mkdir -p $TEST_DIR
cd $TEST_DIR

echo "Running binary in test directory..."
../quickr &
PID=$!

# Wait for server to start
sleep 2

# Test homepage (should contain our templates)
if ! curl -s http://localhost:8080 | grep -q "Quickr"; then
    echo "Error: Homepage template not found"
    kill $PID
    exit 1
fi

# Test static files (should serve our JS)
if ! curl -s http://localhost:8080/static/js/theme.js | grep -q "toggleTheme"; then
    echo "Error: Static files not found"
    kill $PID
    exit 1
fi

# Clean up
kill $PID
cd ..
rm -rf $TEST_DIR

echo "All tests passed! Binary contains all embedded resources."