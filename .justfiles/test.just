# Test recipes for obey-shared

# Show available test commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Run all tests
[no-cd]
all:
    go test ./...

# Run all tests with verbose output
[no-cd]
verbose:
    go test -v ./...

# Run tests with coverage
[no-cd]
coverage:
    go test -cover ./...

# Generate HTML coverage report
[no-cd]
coverage-html:
    mkdir -p coverage
    go test -coverprofile=coverage/coverage.out ./... && go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Run all tests with race detection
[no-cd]
race:
    go test -count=1 -race ./...

# Run tests for a specific package
[no-cd]
pkg package:
    go test -count=1 -v ./{{package}}/...
