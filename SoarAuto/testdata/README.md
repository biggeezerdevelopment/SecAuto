# Test Data

This directory contains test data files used by the test suites.

## Directory Structure

```
testdata/
├── configs/          # Test configuration files
├── playbooks/        # Test playbook definitions
├── certificates/     # Test TLS certificates
├── integrations/     # Test integration configurations
├── clients/          # Test client data
└── fixtures/         # Test fixtures and mock data
```

## Usage

Test data files are loaded by tests using relative paths from the test files.

Example:
```go
configPath := filepath.Join("../../testdata/configs/test-config.yaml")
config, err := config.LoadConfig(configPath)
```

## Test Data Guidelines

- Use realistic but anonymized data
- Keep test data minimal but comprehensive
- Document any special test data requirements
- Use consistent naming conventions