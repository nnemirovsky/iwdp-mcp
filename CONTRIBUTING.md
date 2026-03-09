# Contributing to iwdp-mcp

Thanks for your interest in contributing!

## Development Setup

```bash
# Clone and build
git clone https://github.com/nnemirovsky/iwdp-mcp.git
cd iwdp-mcp
make build

# Install ios-webkit-debug-proxy (macOS)
brew install ios-webkit-debug-proxy
```

## Before Submitting a PR

```bash
make fmt    # gofumpt formatting
make lint   # golangci-lint
make test   # all unit tests
```

## Testing

- **Unit tests** (`make test`) — no device needed, uses mock WebSocket server
- **E2E tests** (`make test-e2e`) — builds binaries, tests CLI + MCP server JSON-RPC
- **Integration tests** (`make test-integration`) — requires `ios_webkit_debug_proxy` binary
- **Simulator tests** (`make test-simulator`) — boots iOS Simulator, tests all tools against real Safari

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add heap snapshot collection
fix: prevent concurrent WebSocket write panic
refactor: extract URL parsing into DevicePort helper
test: add table-driven tests for proxy port parsing
```

## Branch Names

Prefix branches with the change type:

```
feat/heap-snapshots
fix/concurrent-write
refactor/proxy-port-parsing
```

## Architecture

- `internal/webkit/` — WebKit Inspector Protocol client (WebSocket + Target routing)
- `internal/tools/` — Tool implementations shared by MCP server and CLI
- `internal/proxy/` — iwdp process detection and management
- `cmd/iwdp-mcp/` — MCP server binary (stdio transport)
- `cmd/iwdp-cli/` — CLI binary

All WebKit protocol communication goes through `webkit.Client`. Tools return structured results; formatting is done by the caller (MCP or CLI).

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
