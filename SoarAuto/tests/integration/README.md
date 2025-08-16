# Integration Tests

This directory contains integration tests that test the interaction between multiple components of SecAuto.

## Test Categories

- **Redis Integration**: Tests Redis connectivity and operations
- **API Integration**: Tests HTTP endpoints with real dependencies
- **TLS Integration**: Tests HTTPS functionality and certificate handling
- **Client Integration**: Tests multi-tenant client functionality

## Running Integration Tests

```bash
# Run all integration tests
go test ./tests/integration/...

# Run specific integration test
go test ./tests/integration/redis_test.go

# Run with verbose output
go test -v ./tests/integration/...
```

## Test Environment

Integration tests require:
- Redis server running on localhost:6379
- Test configuration files
- Valid test certificates for TLS tests