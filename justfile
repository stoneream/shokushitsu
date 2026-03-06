set shell := ["bash", "-eu", "-o", "pipefail", "-c"]
export GOCACHE := justfile_directory() + "/.gocache"

default: check

# Apply gofmt formatting.
fmt:
    go fmt ./...

# Fail if any file is not gofmt-formatted.
fmt-check:
    unformatted=$(gofmt -l .); \
    if [[ -n "$unformatted" ]]; then \
      echo "Unformatted files:"; \
      echo "$unformatted"; \
      exit 1; \
    fi

# Normalize module dependencies.
deps:
    go mod tidy

# Fail if go.mod/go.sum are not tidy.
deps-check:
    go mod tidy -diff

# Run go vet checks.
vet:
    go vet ./...

# Run all tests.
test:
    go test ./...

# Build the CLI binary.
build:
    mkdir -p bin
    go build -o bin/shoku ./cmd/shoku

# Run full quality gate.
check: fmt-check deps-check vet test
