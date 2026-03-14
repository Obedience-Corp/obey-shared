# Lint and code quality recipes for obey-shared

# Show available lint commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Run all quality checks
[no-cd]
all: vet
    @echo "All quality checks passed"

# Run go vet
[no-cd]
vet:
    go vet ./...

# Run golangci-lint (if installed)
[no-cd]
golangci:
    golangci-lint run ./...
