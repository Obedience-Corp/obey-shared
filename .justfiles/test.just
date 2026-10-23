# Test recipes for obey-shared
# Most recipes delegate to github.com/Obedience-Corp/build-util via
# cmd/buildutil (library mode: test/lint/coverage/clean/all).

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

# Run container_fs-tagged tests in a Linux container with no network. The host
# only vets and cross-compiles; every file the tests create lives in the container.
[no-cd]
container-fs image="alpine:3.22":
    #!/usr/bin/env bash
    set -euo pipefail
    pkgs=$(grep -rl --include='*_test.go' '^//go:build container_fs' . | xargs -n1 dirname | sort -u)
    if [ -z "$pkgs" ]; then echo "no container_fs tests"; exit 0; fi
    go vet -tags container_fs $pkgs
    case "$(docker info --format '{{{{.Architecture}}')" in
        aarch64|arm64) goarch=arm64 ;;
        x86_64|amd64) goarch=amd64 ;;
        *) echo "unsupported docker architecture" >&2; exit 1 ;;
    esac
    out=$(mktemp -d)
    cid=""
    cleanup() { if [ -n "$cid" ]; then docker rm -f "$cid" >/dev/null 2>&1 || true; fi; rm -rf "$out"; }
    trap cleanup EXIT
    for pkg in $pkgs; do
        name=$(printf '%s' "${pkg#./}" | tr '/' '_')
        GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 go test -c -tags container_fs -o "$out/$name.test" "$pkg"
    done
    cid=$(docker create --network none {{image}} sh -c 'set -e; for t in /t/*.test; do echo "== $t"; "$t" -test.v -test.count=1; done')
    docker cp "$out/." "$cid:/t"
    status=0
    docker start -a "$cid" || status=$?
    state=$(docker inspect --format '{{{{.State.Status}}' "$cid")
    if [ "$state" != "exited" ]; then
        echo "container-fs: container never ran (state=$state, start status=$status)" >&2
        exit 1
    fi
    exit "$status"
