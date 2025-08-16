# End-to-End Tests

This directory contains end-to-end tests that test complete user workflows and scenarios.

## Test Scenarios

- **Playbook Execution**: Complete playbook execution workflow
- **Client Management**: Multi-tenant client operations
- **Integration Workflows**: External service integration testing
- **Security Workflows**: Authentication and authorization testing

## Running E2E Tests

```bash
# Run all e2e tests
go test ./tests/e2e/...

# Run specific scenario
go test ./tests/e2e/playbook_execution_test.go

# Run with test server
./scripts/run-e2e-tests.sh
```

## Test Environment

E2E tests require:
- Full SecAuto server running
- Redis server
- Test data and configurations
- Mock external services