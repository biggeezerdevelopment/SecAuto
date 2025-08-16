#!/bin/bash

# SecAuto Test Runner Script
# This script runs various types of tests for the SecAuto project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT="30m"
COVERAGE_THRESHOLD=70
REDIS_TEST_DB=15

# Functions
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

check_dependencies() {
    print_header "Checking Dependencies"
    
    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    print_success "Go $(go version | cut -d' ' -f3) found"
    
    # Check Redis
    if ! command -v redis-cli &> /dev/null; then
        print_warning "Redis CLI not found - integration tests may fail"
    else
        if redis-cli ping &> /dev/null; then
            print_success "Redis server is running"
        else
            print_warning "Redis server is not running - integration tests will be skipped"
        fi
    fi
    
    # Check test database
    if redis-cli -n $REDIS_TEST_DB ping &> /dev/null; then
        print_success "Redis test database accessible"
        # Clean test database
        redis-cli -n $REDIS_TEST_DB FLUSHDB > /dev/null
        print_success "Test database cleaned"
    fi
}

run_unit_tests() {
    print_header "Running Unit Tests"
    
    echo "Running unit tests with coverage..."
    go test -v -race -coverprofile=coverage.out -timeout=$TEST_TIMEOUT ./pkg/...
    
    if [ $? -eq 0 ]; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        exit 1
    fi
    
    # Generate coverage report
    if [ -f coverage.out ]; then
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "Coverage: ${coverage}%"
        
        if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_success "Coverage threshold met (${coverage}% >= ${COVERAGE_THRESHOLD}%)"
        else
            print_warning "Coverage below threshold (${coverage}% < ${COVERAGE_THRESHOLD}%)"
        fi
        
        # Generate HTML coverage report
        go tool cover -html=coverage.out -o coverage.html
        print_success "Coverage report generated: coverage.html"
    fi
}

run_integration_tests() {
    print_header "Running Integration Tests"
    
    # Check if Redis is available
    if ! redis-cli ping &> /dev/null; then
        print_warning "Skipping integration tests - Redis not available"
        return
    fi
    
    echo "Running integration tests..."
    go test -v -race -tags=integration -timeout=$TEST_TIMEOUT ./tests/integration/...
    
    if [ $? -eq 0 ]; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        exit 1
    fi
}

run_e2e_tests() {
    print_header "Running End-to-End Tests"
    
    echo "Running e2e tests..."
    go test -v -race -tags=e2e -timeout=$TEST_TIMEOUT ./tests/e2e/...
    
    if [ $? -eq 0 ]; then
        print_success "E2E tests passed"
    else
        print_error "E2E tests failed"
        exit 1
    fi
}

run_load_tests() {
    print_header "Running Load Tests"
    
    echo "Running load tests..."
    go test -v -race -tags=load -timeout=$TEST_TIMEOUT ./tests/load/...
    
    if [ $? -eq 0 ]; then
        print_success "Load tests passed"
    else
        print_error "Load tests failed"
        exit 1
    fi
}

run_security_tests() {
    print_header "Running Security Tests"
    
    # Check if gosec is installed
    if ! command -v gosec &> /dev/null; then
        print_warning "gosec not installed - installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    echo "Running security scan..."
    gosec -fmt json -out gosec-report.json -stdout ./...
    
    if [ $? -eq 0 ]; then
        print_success "Security scan passed"
    else
        print_warning "Security issues found - check gosec-report.json"
    fi
}

run_linting() {
    print_header "Running Code Linting"
    
    # Check if golangci-lint is installed
    if ! command -v golangci-lint &> /dev/null; then
        print_warning "golangci-lint not installed - installing..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
    fi
    
    echo "Running linter..."
    golangci-lint run --timeout=$TEST_TIMEOUT
    
    if [ $? -eq 0 ]; then
        print_success "Linting passed"
    else
        print_error "Linting failed"
        exit 1
    fi
}

cleanup() {
    print_header "Cleaning Up"
    
    # Clean test database
    if redis-cli -n $REDIS_TEST_DB ping &> /dev/null; then
        redis-cli -n $REDIS_TEST_DB FLUSHDB > /dev/null
        print_success "Test database cleaned"
    fi
    
    # Remove temporary files
    rm -f coverage.out
    print_success "Temporary files cleaned"
}

show_help() {
    echo "SecAuto Test Runner"
    echo ""
    echo "Usage: $0 [OPTIONS] [TEST_TYPE]"
    echo ""
    echo "TEST_TYPE:"
    echo "  unit        Run unit tests only"
    echo "  integration Run integration tests only"
    echo "  e2e         Run end-to-end tests only"
    echo "  load        Run load tests only"
    echo "  security    Run security tests only"
    echo "  lint        Run linting only"
    echo "  all         Run all tests (default)"
    echo ""
    echo "OPTIONS:"
    echo "  -h, --help     Show this help message"
    echo "  -v, --verbose  Verbose output"
    echo "  -c, --coverage Generate coverage report"
    echo "  --no-cleanup   Skip cleanup"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run all tests"
    echo "  $0 unit              # Run only unit tests"
    echo "  $0 -c unit           # Run unit tests with coverage"
    echo "  $0 integration       # Run only integration tests"
}

# Parse command line arguments
VERBOSE=false
COVERAGE=false
NO_CLEANUP=false
TEST_TYPE="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        --no-cleanup)
            NO_CLEANUP=true
            shift
            ;;
        unit|integration|e2e|load|security|lint|all)
            TEST_TYPE=$1
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Set verbose mode
if [ "$VERBOSE" = true ]; then
    set -x
fi

# Main execution
main() {
    print_header "SecAuto Test Suite"
    echo "Test type: $TEST_TYPE"
    echo "Timeout: $TEST_TIMEOUT"
    echo ""
    
    check_dependencies
    
    case $TEST_TYPE in
        unit)
            run_unit_tests
            ;;
        integration)
            run_integration_tests
            ;;
        e2e)
            run_e2e_tests
            ;;
        load)
            run_load_tests
            ;;
        security)
            run_security_tests
            ;;
        lint)
            run_linting
            ;;
        all)
            run_unit_tests
            run_integration_tests
            run_e2e_tests
            run_security_tests
            run_linting
            ;;
    esac
    
    if [ "$NO_CLEANUP" != true ]; then
        cleanup
    fi
    
    print_header "Test Suite Complete"
    print_success "All tests completed successfully!"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main