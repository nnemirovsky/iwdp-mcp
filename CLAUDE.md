# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Dev Commands

```bash
go build ./...                    # Compile (quick check)
make build                        # Compile to bin/iwdp-mcp + bin/iwdp-cli
make install                      # Install both binaries via go install
make test                         # Run all tests (go test ./... -v -count=1)
make test-e2e                     # Run e2e tests (builds binaries, tests CLI + MCP server)
make test-coverage                # Run tests with coverage → coverage.html
make test-integration             # Run integration tests (requires iwdp binary installed)
make test-simulator               # Run simulator tests (boots iOS Sim + iwdp, tests all tools)
make sim-setup                    # Boot iOS Simulator + iwdp (prints IWDP_SIM_WS_URL)
make sim-teardown                 # Shut down simulator + iwdp
make fmt                          # gofumpt -w . (format all Go files)
make lint                         # golangci-lint run ./...
make tidy                         # go mod tidy
make clean                        # Remove bin/ and coverage files
```

## Pre-commit

Before committing, always run `make fmt` and `make lint` and fix any issues.

## How ios-webkit-debug-proxy Works

- **Port 9221** — listing port: `http://localhost:9221/json` returns connected devices
- **Port 9222+** — each device gets an incremented port starting at 9222
  - `http://localhost:9222/json` → pages for first device
  - `http://localhost:9223/json` → pages for second device, etc.
- Each page has a `webSocketDebuggerUrl` for WebSocket connection
- The WebSocket speaks **WebKit Inspector Protocol**

### Target-Based Message Routing

iwdp uses **Target routing** — you cannot send domain commands directly on the WebSocket. On connect, iwdp sends a `Target.targetCreated` event with a `targetId`. All subsequent commands must be wrapped in `Target.sendMessageToTarget`:

```
Client → Target.sendMessageToTarget { targetId, message: JSON-stringified inner command }
iwdp → WebKit
WebKit → Target.dispatchMessageFromTarget { targetId, message: JSON-stringified response }
iwdp → Client
```

- The outer `sendMessageToTarget` gets an empty ack `{"result":{},"id":N}` — this is ignored
- The inner command has its own ID used for response matching (dual-ID routing)
- `webkit.Client` handles this transparently: it waits up to 100ms for `Target.targetCreated` on connect; if received, all `Send()` calls auto-wrap in Target routing
- Mock test servers don't send `Target.targetCreated`, so the client falls back to direct mode (no wrapping)

### Domain Enable/Disable Through Target Routing

Many `<Domain>.enable`/`<Domain>.disable` methods (DOM.enable, CSS.enable) return "not found" through iwdp Target routing. However, the actual domain methods (DOM.getDocument, CSS.getMatchedStylesForNode, Runtime.evaluate) work without explicit enabling. Don't require `.enable` calls as a prerequisite.

### Known Limitations

- `CSS.getAllStyleSheets` — exists in the WebKit protocol spec but requires `CSS.enable` first, which doesn't work through iwdp Target routing. Skipped in tests.
- `Page.snapshotRect` requires explicit pixel dimensions — compute them first via `Runtime.evaluate` (see `TakeScreenshot` in page.go).
- Only **one WebSocket debugger connection per page** — simulator tests use a `sync.Once` shared connection pattern.
- iwdp sends error `data` as a JSON array `[{"code":...,"message":...}]` — `ErrorData.Data` is `json.RawMessage` to handle this.

## Architecture

Two binaries, one shared `internal/` package tree:

- `cmd/iwdp-mcp/` — MCP server binary (stdio transport, 100+ tools registered via `mcp.AddTool`)
- `cmd/iwdp-cli/` — CLI binary (devices, pages, eval, navigate, screenshot, cookies, dom, console, network)

### Package Layout

- `internal/webkit/` — WebKit Inspector Protocol client
  - `client.go` — WebSocket connection with Target-based message routing for iwdp, concurrent-safe writes (`writeMu`), dual-ID request/response routing, event dispatch
  - `types.go` — Protocol type definitions for all domains (DOM, CSS, Network, Runtime, Debugger, etc.)
  - `domains.go` — Domain enable/disable helpers
  - `testutil/mock.go` — Mock WebSocket server for unit tests
- `internal/tools/` — Tool implementations shared by MCP server and CLI
  - One file per domain: page.go, runtime.go, dom.go, css.go, network.go, storage.go, console.go, debugger.go, timeline.go, memory.go, profiler.go, animation.go, canvas.go, layertree.go, worker.go, audit.go, security.go, interaction.go
- `internal/proxy/` — iwdp process detection and management
  - `IsRunning()` — checks port 9221 (listing port). Never use a device port for detection.
  - `ListDevices()` — HTTP GET on port 9221
  - `ListPages(port)` — HTTP GET on a device-specific port
  - `ListAllPages()` — discovers all devices, queries each for pages
  - `DevicePort(entry)` — extracts port from device entry URL (e.g., `localhost:9222` → 9222)

### Key Patterns

- All WebKit protocol communication goes through `webkit.Client`
- Tools return structured results; formatting is done by the caller (MCP or CLI)
- Unit tests use mock WebSocket server from `internal/webkit/testutil/`
- E2E tests in `e2e/` build actual binaries and test CLI help/error paths + MCP server JSON-RPC initialization
- Integration tests (`-tags=integration`) use the real `ios_webkit_debug_proxy` binary
- Simulator tests (`-tags=simulator`) boot iOS Simulator + iwdp and test all tools against real Safari
- Error messages should be actionable (tell the user what to do)
- Network/Console/Timeline use collector patterns: `Start()` registers event handlers, `Get*()` returns collected data
- gorilla/websocket is not concurrent-write-safe — `Client.writeMu` mutex protects `conn.WriteMessage`
- Simulator tests share a single WebSocket connection via `sync.Once` (`getSimClient`) — never create multiple connections to the same page
- Use `simOrigin()` helper to get the page's actual origin for storage tests — never hardcode origins
- Use `t.Skipf` (not `t.Fatalf`) for features that may not be supported in all WebKit versions

## Git Conventions

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add heap snapshot collection
fix: prevent concurrent WebSocket write panic
refactor: extract URL parsing into DevicePort helper
test: add table-driven tests for proxy port parsing
```

### Branch Names

Prefix branches with the change type:

```
feat/heap-snapshots
fix/concurrent-write
refactor/proxy-port-parsing
```

## Key Dependencies

- `github.com/modelcontextprotocol/go-sdk` v1.4.0 — official MCP Go SDK
- `github.com/gorilla/websocket` — WebSocket client

## License

MIT — see LICENSE file. ios-webkit-debug-proxy (BSD-3-Clause) is a runtime dependency only; we connect to it over HTTP/WebSocket without bundling its code.
