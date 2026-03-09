.PHONY: build test test-e2e test-integration test-simulator test-coverage \
       sim-setup sim-teardown lint fmt tidy install clean release release-snapshot

# Build
build:
	go build -o bin/iwdp-mcp ./cmd/iwdp-mcp
	go build -o bin/iwdp-cli ./cmd/iwdp-cli

# Run
run-mcp:
	go run ./cmd/iwdp-mcp

run-cli:
	go run ./cmd/iwdp-cli $(ARGS)

# Install both binaries
install:
	go install ./cmd/...

# Test
test:
	go test ./... -v -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-e2e:
	go test ./e2e/ -v -count=1 -timeout=120s

test-integration:
	go test -tags=integration ./... -v -count=1 -timeout=120s

# Simulator tests — boots iOS Simulator + iwdp, runs all tools against real Safari.
# Usage: make test-simulator
#   or:  make sim-setup && go test -tags=simulator ./... -v && make sim-teardown
sim-setup:
	@eval "$$(./scripts/sim-setup.sh)" && echo "IWDP_SIM_WS_URL=$$IWDP_SIM_WS_URL"

sim-teardown:
	@./scripts/sim-setup.sh --teardown

test-simulator:
	@echo "==> Setting up iOS Simulator..."
	@eval "$$(./scripts/sim-setup.sh)" && \
		echo "==> Running simulator tests..." && \
		go test -tags=simulator ./... -v -count=1 -timeout=300s && \
		echo "==> Tearing down..." && \
		./scripts/sim-setup.sh --teardown
	@./scripts/sim-setup.sh --teardown 2>/dev/null || true

# Lint
lint:
	golangci-lint run ./...

# Format
fmt:
	gofumpt -w .

# Tidy
tidy:
	go mod tidy

# Release (dry run)
release-snapshot:
	goreleaser release --snapshot --clean

# Release
release:
	goreleaser release --clean

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html dist/
