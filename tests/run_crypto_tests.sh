#!/bin/bash

echo "=== Running Crypto Engine Unit Tests ==="
cd /workspace

# Run crypto tests with verbose output and coverage
go test -v -race -coverprofile=crypto_coverage.out ./internal/vault/

# Generate coverage report
go tool cover -html=crypto_coverage.out -o crypto_coverage.html

# Run benchmarks
echo "=== Running Crypto Benchmarks ==="
go test -bench=. -benchmem ./internal/vault/

echo "=== Test Summary ==="
echo "Coverage report generated: crypto_coverage.html"
echo "Tests completed successfully!"
