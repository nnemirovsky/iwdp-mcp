# iwdp-mcp

MCP server + CLI for debugging iOS Safari via [ios-webkit-debug-proxy](https://github.com/google/ios-webkit-debug-proxy).

Speaks **WebKit Inspector Protocol** natively — full access to all 27 WebKit Inspector domains, including httpOnly cookies, network interception, heap snapshots, and more.

## Installation

### Claude Code Plugin (recommended)

Inside Claude Code, run:

```
/plugin marketplace add nnemirovsky/iwdp-mcp
/plugin install iwdp-mcp
```

### Go Install

```bash
# Install both binaries
go install github.com/nnemirovsky/iwdp-mcp/cmd/...@latest
```

### Pre-built Binaries

```bash
# Or download a pre-built binary from GitHub Releases
# https://github.com/nnemirovsky/iwdp-mcp/releases
```

### Build from Source

```bash
git clone https://github.com/nnemirovsky/iwdp-mcp.git
cd iwdp-mcp
make build
```

### Prerequisites

```bash
# Install ios-webkit-debug-proxy (macOS)
brew install ios-webkit-debug-proxy

# Connect an iOS device via USB and enable Web Inspector:
# Settings → Safari → Advanced → Web Inspector → ON
```

For Linux and other platforms, see the [ios-webkit-debug-proxy installation guide](https://github.com/google/ios-webkit-debug-proxy#installation).

## Quick Start

### CLI

```bash
# Start the proxy
ios_webkit_debug_proxy --no-frontend &

# List connected devices (port 9221)
iwdp-cli devices

# List Safari tabs on the first device (port 9222)
iwdp-cli pages

# Evaluate JavaScript
iwdp-cli eval "document.title"

# Take a screenshot
iwdp-cli screenshot -o page.png

# Show all cookies (including httpOnly)
iwdp-cli cookies
```

### MCP Server

Add to your Claude Code config (`.mcp.json`):

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```

## How It Works

```
┌────────────┐    USB     ┌──────────┐   HTTP/WS    ┌──────────┐
│ iOS Device │◄──────────►│   iwdp   │◄────────────►│ iwdp-mcp │
│  (Safari)  │            │ :9221-N  │              │  or CLI  │
└────────────┘            └──────────┘              └──────────┘
```

`ios-webkit-debug-proxy` exposes:
- **Port 9221** — lists all connected devices
- **Port 9222+** — each device gets an incremented port listing its Safari tabs
- Each tab provides a **WebSocket URL** for the WebKit Inspector Protocol

`iwdp-mcp` connects to those WebSocket endpoints and exposes 100+ tools.

## Tools

### Core
| Tool | Description |
|------|-------------|
| `list_devices` | List connected iOS devices (HTTP GET :9221) |
| `list_pages` | List Safari tabs (HTTP GET :9222+) |
| `select_page` | Connect to a specific tab |
| `navigate` | Go to URL |
| `take_screenshot` | Capture page as PNG |
| `evaluate_script` | Run JavaScript |
| `get_document` | Get DOM tree |
| `query_selector` | Find elements by CSS selector |

### DOM & CSS
`get_outer_html`, `get_attributes`, `get_event_listeners`, `highlight_node`, `get_matched_styles`, `get_computed_style`, `set_style_text`, `force_pseudo_state`, and more.

### Network
`network_enable`, `list_network_requests`, `get_response_body`, `set_request_interception`, `intercept_continue`, `intercept_with_response`, `set_emulated_conditions`, `set_resource_caching_disabled`.

### Storage
`get_cookies` (httpOnly + secure), `set_cookie`, `delete_cookie`, `get_local_storage`, `get_session_storage`, `list_indexed_databases`, `get_indexed_db_data`.

### Debugging
`debugger_enable`, `set_breakpoint`, `pause`, `resume`, `step_over`, `step_into`, `step_out`, `get_script_source`, `evaluate_on_call_frame`, `set_pause_on_exceptions`.

### Performance
`timeline_start/stop`, `memory_start/stop_tracking`, `heap_snapshot`, `cpu_start/stop_profiling`, `script_start/stop_profiling`.

### More
Animation, Canvas, LayerTree, Workers, Audit, Security (TLS certificates), and element interaction (`click`, `fill`, `type_text`).

## Development

```bash
make build              # Build both binaries
make test               # Run all tests
make test-coverage      # Tests with coverage report
make lint               # golangci-lint
make fmt                # gofumpt formatting
```

### Testing

Unit tests use a mock WebSocket server (`internal/webkit/testutil/`) that simulates the WebKit Inspector Protocol.

```bash
# Unit tests (no device needed)
make test

# E2E tests (builds binaries, tests CLI + MCP server JSON-RPC)
make test-e2e

# Integration tests (requires iwdp binary installed)
make test-integration

# Simulator tests — boots iOS Simulator + iwdp, tests ALL tools against real Safari
make test-simulator
```

#### iOS Simulator Tests

Simulator tests (`-tags=simulator`) boot an iOS Simulator, start `ios_webkit_debug_proxy` with the simulator's web inspector socket, and run every tool against a real Safari page. No physical device needed.

```bash
# One-command: setup → test → teardown
make test-simulator

# Or manually for debugging:
make sim-setup          # Prints IWDP_SIM_WS_URL
export IWDP_SIM_WS_URL=ws://localhost:9222/devtools/page/1
go test -tags=simulator ./... -v -run TestSim_Navigate
make sim-teardown
```

> **Note:** Requires macOS with Xcode and `ios-webkit-debug-proxy` installed. GitHub Actions `macos-latest` runners have Xcode and iOS Simulator runtimes pre-installed.

### Project Structure

```
iwdp-mcp/
├── cmd/
│   ├── iwdp-mcp/          # MCP server (stdio transport)
│   └── iwdp-cli/          # CLI tool
├── internal/
│   ├── webkit/            # WebKit Inspector Protocol client
│   │   ├── client.go      # WebSocket connection + message routing
│   │   ├── types.go       # Protocol type definitions
│   │   ├── domains.go     # Domain enable/disable helpers
│   │   └── testutil/      # Mock WebSocket server
│   ├── tools/             # Tool implementations (shared by both binaries)
│   └── proxy/             # iwdp process detection + management
├── skills/                # Claude Code skill definition
├── .claude-plugin/        # Claude Code plugin manifest
└── .mcp.json              # MCP server configuration
```

## License

MIT — see [LICENSE](LICENSE).

`ios-webkit-debug-proxy` is a separate project licensed under BSD-3-Clause. This tool connects to it over HTTP/WebSocket at runtime without bundling its code.
