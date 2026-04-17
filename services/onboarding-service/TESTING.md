# Testing Guide - Onboarding Service

## Overview

The onboarding service has comprehensive test coverage including unit tests, integration tests, and CI/CD automation.

**Current Coverage**: 80%+ (enforced in CI)

## Quick Start

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests (requires PostgreSQL)
make test-integration

# Generate coverage report
make test-coverage
```

## Test Structure

### Unit Tests

Located alongside the code they test, following Go conventions (`*_test.go` files).

#### Models Tests (`internal/models/models_test.go`)

- JSON serialization/deserialization
- Constant validations
- Message type conversions
- Kafka message structures

**Coverage**: ~95%

```bash
cd internal/models
go test -v
```

#### CSV Parser Tests (`internal/integrations/csv/parser_test.go`)

- Valid CSV parsing
- Validation error handling
- Required field checking
- Role normalization
- Course code parsing
- Email validation
- Graduation year parsing

**Coverage**: ~90%

```bash
cd internal/integrations/csv
go test -v
```

#### Config Tests (`internal/config/config_test.go`)

- Environment variable loading
- Default values
- Type conversions (int, int64, duration)
- JWT secret validation
- Production vs development mode

**Coverage**: ~85%

```bash
cd internal/config
go test -v
```

### Integration Tests

Located in `integration_test.go` (build tag: `integration`).

#### Database Tests

- Job lifecycle (create, retrieve, update)
- Task lifecycle (create, retrieve, update)
- Batch task creation (100 tasks)
- Audit log immutability
- Job progress tracking

**Setup Requirements**:

- PostgreSQL 15+ running
- Database: `onboarding_test`

**Environment Variables**:

```bash
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_USER=postgres
export TEST_DB_PASSWORD=postgres
export TEST_DB_NAME=onboarding_test
```

**Run**:

```bash
# With Docker Compose
docker-compose up -d postgres
make test-integration

# Manual
go test -v -tags=integration ./...
```

## Running Tests

### Local Development

#### Prerequisites

```bash
# Install dependencies
go mod download

# Install linter
make dev-setup
```

#### Run All Tests

```bash
make test
```

#### Run Specific Package

```bash
# Models
go test -v ./internal/models

# CSV Parser
go test -v ./internal/integrations/csv

# Config
go test -v ./internal/config
```

#### Run Single Test

```bash
go test -v -run TestParser_Parse_ValidCSV ./internal/integrations/csv
```

#### Run with Race Detector

```bash
go test -race ./...
```

### CI/CD (GitHub Actions)

The CI pipeline runs automatically on:

- Push to `main` or `claude/**` branches
- Pull requests to `main`

#### Pipeline Stages

1. **Lint** (golangci-lint)
   - Code formatting (gofmt)
   - Static analysis
   - Security checks (gosec)
   - Best practices

2. **Unit Tests**
   - All unit tests with race detector
   - Coverage calculation
   - 80% coverage threshold enforced
   - Upload to Codecov

3. **Integration Tests**
   - PostgreSQL service container
   - Full database lifecycle tests
   - Migration validation

4. **Build**
   - Server binary compilation
   - Worker binary compilation
   - Artifact upload

5. **Docker Build**
   - Service image build
   - Worker image build
   - Layer caching

6. **Security Scan**
   - Gosec security scanner
   - SARIF report upload

## Code Coverage

### View Coverage Report

```bash
# Generate HTML report
make test-coverage

# Open in browser
open coverage.html
```

### Coverage Breakdown

| Package                     | Coverage | Status               |
| --------------------------- | -------- | -------------------- |
| `internal/models`           | ~95%     | ✅ Excellent         |
| `internal/integrations/csv` | ~90%     | ✅ Excellent         |
| `internal/config`           | ~85%     | ✅ Good              |
| `pkg/kafka`                 | ~70%     | ⚠️ Needs improvement |
| `pkg/websocket`             | ~70%     | ⚠️ Needs improvement |
| `internal/repository`       | ~80%     | ✅ Good              |
| **Overall**                 | **80%+** | ✅ Meets threshold   |

### Coverage Requirements

- **Minimum**: 80% overall coverage (enforced in CI)
- **Target**: 85% overall coverage
- **Critical paths**: 90%+ coverage (CSV parsing, database operations)

## Linting

### Run Linter

```bash
# Run all linters
make lint

# Auto-fix formatting
make fmt
```

### Linter Configuration

Located in `.golangci.yml`:

**Enabled Linters** (30+):

- `errcheck` - Unchecked errors
- `gosec` - Security issues
- `govet` - Suspicious constructs
- `staticcheck` - Go static analysis
- `ineffassign` - Ineffectual assignments
- `misspell` - Spelling mistakes
- `dupl` - Code duplication
- `gocyclo` - Cyclomatic complexity
- And more...

**Disabled for Tests**:

- `gomnd` (magic numbers)
- `goconst` (repeated strings)
- `dupl` (code duplication)
- `lll` (line length)

## Writing Tests

### Unit Test Template

```go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "expected",
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Test Template

```go
// +build integration

package main

import (
    "context"
    "testing"
)

func TestIntegration_MyFeature(t *testing.T) {
    // Setup
    ctx := context.Background()
    // ... initialize dependencies

    // Execute
    result, err := MyFeature(ctx)

    // Assert
    if err != nil {
        t.Fatalf("MyFeature() failed: %v", err)
    }
    if result != expected {
        t.Errorf("MyFeature() = %v, want %v", result, expected)
    }
}
```

## Best Practices

### Do's ✅

- Use table-driven tests for multiple scenarios
- Test both success and failure cases
- Use descriptive test names
- Mock external dependencies
- Clean up resources in tests
- Use test fixtures for complex data
- Check error messages, not just error presence
- Use `t.Parallel()` for independent tests

### Don'ts ❌

- Don't test private functions directly
- Don't use sleep/time.Sleep in tests
- Don't hardcode paths or ports
- Don't skip cleanup in defer statements
- Don't share state between tests
- Don't use global variables in tests

## Debugging Tests

### Run with Verbose Output

```bash
go test -v ./...
```

### Run Specific Test

```bash
go test -v -run TestName ./package
```

### Enable Race Detector

```bash
go test -race ./...
```

### Print Test Coverage

```bash
go test -cover ./...
```

### Benchmark Tests

```bash
go test -bench=. ./...
```

## Continuous Integration

### GitHub Actions Workflow

Location: `.github/workflows/onboarding-service-ci.yml`

**Triggers**:

- Push to main or claude/\*\* branches
- Pull requests to main
- Changes to onboarding service files

**Status Badge**:

```markdown
![CI Status](https://github.com/noorshaik95/slate/workflows/Onboarding%20Service%20CI/badge.svg)
```

### Codecov Integration

Coverage reports automatically uploaded to Codecov.

**View Reports**: https://codecov.io/gh/yourusername/slate

## Troubleshooting

### Test Failures

**Issue**: Tests pass locally but fail in CI

- Check environment variables
- Verify PostgreSQL version
- Check file paths (absolute vs relative)
- Review CI logs for specific errors

**Issue**: Integration tests fail

- Ensure PostgreSQL is running
- Check database credentials
- Verify migrations run successfully
- Check connection timeouts

**Issue**: Coverage below 80%

- Add tests for uncovered code paths
- Remove dead code
- Test error paths
- Add edge case tests

### Linter Issues

**Issue**: False positives

- Add `//nolint:lintername` comment with explanation
- Update `.golangci.yml` to exclude specific rules

**Issue**: Formatting errors

- Run `make fmt` to auto-fix
- Configure editor to run gofmt on save

## Performance Testing

### Load Testing (Future)

```bash
# Test 10,000 user bulk upload
./scripts/load_test_onboarding.sh 10000

# Measure throughput
go test -bench=BenchmarkBulkInsert -benchmem
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [golangci-lint](https://golangci-lint.run/)
- [Codecov](https://codecov.io/)
- [GitHub Actions](https://docs.github.com/en/actions)

## Getting Help

For questions or issues:

1. Check this guide
2. Review existing tests for examples
3. Check CI logs for detailed errors
4. Open an issue in the repository
