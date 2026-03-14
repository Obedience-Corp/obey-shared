#!/usr/bin/env just --justfile
# obey-shared — shared packages for Obedience Corp projects

# Modules
[doc('Testing (unit, coverage)')]
mod test '.justfiles/test.just'

[doc('Code quality (lint, vet, fmt)')]
mod lint '.justfiles/lint.just'

[private]
default:
    #!/usr/bin/env bash
    echo "obey-shared — shared packages"
    echo ""
    just --list --unsorted

# Build all packages
build:
    go build ./...

# Format code
fmt:
    go fmt ./...
    gofmt -s -w .

# Update and tidy dependencies
deps:
    go get -u ./...
    go mod tidy

# Run go vet
vet:
    go vet ./...
