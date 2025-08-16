# Load Tests

This directory contains load and performance tests for SecAuto.

## Test Types

- **Concurrent Playbook Execution**: Test system under concurrent load
- **API Endpoint Load**: Test individual endpoint performance
- **Redis Performance**: Test Redis operations under load
- **Memory Usage**: Test memory consumption patterns

## Running Load Tests

```bash
# Run load tests
go test ./tests/load/... -timeout=30m

# Run with custom parameters
go test ./tests/load/playbook_load_test.go -concurrent=100 -duration=5m

# Generate performance report
./scripts/run-load-tests.sh --report
```

## Performance Targets

- **Response Time**: <100ms average for API calls
- **Throughput**: >1000 requests/second
- **Memory Usage**: <512MB under normal load
- **Concurrent Jobs**: Support 100+ concurrent playbook executions