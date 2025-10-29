#!/bin/bash

echo "=== Password Vault CLI - Comprehensive Test Suite ==="
cd /workspace

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

run_test_suite() {
    local suite_name="$1"
    local test_path="$2"
    local coverage_file="$3"
    
    echo -e "${YELLOW}=== Running $suite_name ===${NC}"
    
    if go test -v -race -coverprofile="$coverage_file" "$test_path"; then
        echo -e "${GREEN}✅ $suite_name PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ $suite_name FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
}

# Create test results directory
mkdir -p test-results

# Run all test suites
echo "Starting comprehensive test execution..."
echo ""

# 1. Crypto Engine Tests
run_test_suite "Crypto Engine Unit Tests" "./internal/vault/" "test-results/crypto_coverage.out"

# 2. Storage Layer Tests  
run_test_suite "Storage Layer Tests" "./internal/store/" "test-results/storage_coverage.out"

# 3. CLI Command Tests
run_test_suite "CLI Command Tests" "./internal/cli/" "test-results/cli_coverage.out"

# 4. Domain Model Tests
run_test_suite "Domain Model Tests" "./internal/domain/" "test-results/domain_coverage.out"

# 5. Configuration Tests
run_test_suite "Configuration Tests" "./internal/config/" "test-results/config_coverage.out"

# 6. Integration Tests
echo -e "${YELLOW}=== Running End-to-End Integration Tests ===${NC}"
if go test -v -race -tags=integration ./tests/integration/...; then
    echo -e "${GREEN}✅ Integration Tests PASSED${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ Integration Tests FAILED${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo ""

# Generate combined coverage report
echo -e "${YELLOW}=== Generating Coverage Reports ===${NC}"

# Combine coverage profiles
echo "mode: set" > test-results/combined_coverage.out
for coverage_file in test-results/*_coverage.out; do
    if [ -f "$coverage_file" ]; then
        tail -n +2 "$coverage_file" >> test-results/combined_coverage.out
    fi
done

# Generate HTML coverage report
go tool cover -html=test-results/combined_coverage.out -o test-results/coverage_report.html

# Calculate coverage percentage
COVERAGE_PERCENT=$(go tool cover -func=test-results/combined_coverage.out | grep total | awk '{print $3}')

echo "Coverage report generated: test-results/coverage_report.html"
echo "Overall coverage: $COVERAGE_PERCENT"
echo ""

# Run benchmarks
echo -e "${YELLOW}=== Running Performance Benchmarks ===${NC}"
go test -bench=. -benchmem -cpuprofile=test-results/cpu.prof -memprofile=test-results/mem.prof ./internal/vault/ > test-results/benchmark_results.txt
echo "Benchmark results saved to: test-results/benchmark_results.txt"
echo ""

# Security tests
echo -e "${YELLOW}=== Running Security Tests ===${NC}"
if command -v gosec &> /dev/null; then
    gosec -fmt json -out test-results/security_report.json ./...
    echo "Security scan completed: test-results/security_report.json"
else
    echo "gosec not installed, skipping security scan"
fi
echo ""

# Test summary
echo -e "${YELLOW}=== Test Summary ===${NC}"
echo "Total test suites: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo "Coverage: $COVERAGE_PERCENT"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some tests failed!${NC}"
    exit 1
fi
