# Test recipes for obey-shared
# Most recipes delegate to the shared buildutil dashboard running in
# library mode (see cmd/buildutil).

# Show available test commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Run the test dashboard
[no-cd]
all:
    go run ./cmd/buildutil test

# Run all tests with verbose output (raw go test)
[no-cd]
verbose:
    go test -v ./...

# Run the coverage dashboard (profile + HTML report)
[no-cd]
coverage:
    go run ./cmd/buildutil coverage

# Alias retained for muscle memory — same as `coverage`
[no-cd]
coverage-html: coverage

# Run the lint dashboard (go vet across all packages)
[no-cd]
lint:
    go run ./cmd/buildutil lint

# Run clean + lint + test + coverage via the dashboard
[no-cd]
ci:
    go run ./cmd/buildutil all

# Run all tests with race detection (raw go test)
[no-cd]
race:
    go test -count=1 -race ./...

# Run tests for a specific package (raw go test)
[no-cd]
pkg package:
    go test -count=1 -v ./{{package}}/...
