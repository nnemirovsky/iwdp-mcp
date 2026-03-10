# iOS WebKit Debug Proxy MCP Server + CLI

[![CI](https://github.com/nnemirovsky/iwdp-mcp/actions/workflows/test.yml/badge.svg)](https://github.com/nnemirovsky/iwdp-mcp/actions/workflows/test.yml)
[![Lint](https://github.com/nnemirovsky/iwdp-mcp/actions/workflows/lint.yml/badge.svg)](https://github.com/nnemirovsky/iwdp-mcp/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nnemirovsky/iwdp-mcp)](https://goreportcard.com/report/github.com/nnemirovsky/iwdp-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[![Install in Claude Code](https://img.shields.io/badge/Claude_Code-Install_Plugin-F97316?style=flat-square&logo=claude&logoColor=white)](#claude-code)
[![Install in VS Code](https://img.shields.io/badge/VS_Code-Install_Server-0098FF?style=flat-square&logo=visualstudiocode&logoColor=white)](https://vscode.dev/redirect/mcp/install?name=iwdp-mcp&config=%7B%22command%22%3A%22iwdp-mcp%22%7D)
[![Install in VS Code Insiders](https://img.shields.io/badge/VS_Code_Insiders-Install_Server-24bfa5?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=iwdp-mcp&config=%7B%22command%22%3A%22iwdp-mcp%22%7D&quality=insiders)
[![Install in Cursor](https://img.shields.io/badge/Cursor-Install_Server-7B3FE4?style=flat-square&logo=cursor&logoColor=white)](https://cursor.com/en/install-mcp?name=iwdp-mcp&config=eyJjb21tYW5kIjoiaXdkcC1tY3AifQ==)
[![Install in Windsurf](https://img.shields.io/badge/Windsurf-Install_Server-5C2D91?style=flat-square&logo=codeium&logoColor=white)](#windsurf)
[![Install in Codex](https://img.shields.io/badge/Codex_CLI-Install_Server-74AA9C?style=flat-square&logo=openai&logoColor=white)](#codex-cli)
[![Install in Antigravity](https://img.shields.io/badge/Antigravity-Install_Server-4285F4?style=flat-square&logo=google&logoColor=white)](#antigravity)
[![Install in JetBrains](https://img.shields.io/badge/JetBrains-Install_Server-000000?style=flat-square&logo=jetbrains&logoColor=white)](#jetbrains-ides)

**Debug iOS Safari from your AI agent.** No existing tool lets AI assistants interact with Safari on a real iPhone. Chrome DevTools MCP speaks CDP, which is incompatible with WebKit. This project bridges that gap.

<video src="https://github.com/user-attachments/assets/dcc0dc05-3a1f-4454-88a8-51ee28da9fd2" autoplay loop muted playsinline></video>

100+ tools across all 27 WebKit Inspector Protocol domains — navigation, screenshots, DOM inspection, CSS, network interception, httpOnly cookies, JS debugging, heap snapshots, profiling, and more.

## Why?

- **Chrome DevTools MCP** exists for Chrome/Chromium. **Nothing existed for Safari** — until now.
- **httpOnly cookies** are invisible to `document.cookie`. This tool reads them via the protocol.
- **Real device testing** catches iOS-specific bugs that simulators miss.
- Works with **Claude Code, VS Code, Cursor, Windsurf, Codex CLI, Antigravity, JetBrains**, and any MCP-compatible client.

## Installation

### Prerequisites

```bash
# Install ios-webkit-debug-proxy (macOS)
brew install ios-webkit-debug-proxy

# Connect an iOS device via USB and enable Web Inspector:
# Settings → Safari → Advanced → Web Inspector → ON
```

For Linux and other platforms, see the [ios-webkit-debug-proxy installation guide](https://github.com/google/ios-webkit-debug-proxy#installation).

### Claude Code Plugin (recommended)

Inside Claude Code, run:

```
/plugin marketplace add nnemirovsky/iwdp-mcp
/plugin install iwdp-mcp
```

### Go Install

```bash
go install github.com/nnemirovsky/iwdp-mcp/cmd/...@latest
```

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/nnemirovsky/iwdp-mcp/releases).

### Build from Source

```bash
git clone https://github.com/nnemirovsky/iwdp-mcp.git
cd iwdp-mcp
make build
```

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

<details id="claude-code">
<summary><strong>Claude Code</strong></summary>

Install as a plugin (recommended):

```
/plugin marketplace add nnemirovsky/iwdp-mcp
/plugin install iwdp-mcp
```

Or add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```

Or via CLI:

```sh
claude mcp add iwdp-mcp -- iwdp-mcp
```
</details>

<details>
<summary><strong>Claude Desktop</strong></summary>

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```
</details>

<details>
<summary><strong>VS Code / VS Code Insiders</strong></summary>

Click the install badges at the top of this README, or add to `.vscode/mcp.json`:

```json
{
  "servers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```

Or via CLI:

```sh
code --add-mcp '{"name":"iwdp-mcp","command":"iwdp-mcp"}'
```
</details>

<details>
<summary><strong>Cursor</strong></summary>

Click the Cursor install badge at the top of this README, or add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```
</details>

<details id="windsurf">
<summary><strong>Windsurf</strong></summary>

Add to `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```
</details>

<details id="codex-cli">
<summary><strong>Codex CLI</strong></summary>

```sh
codex mcp add iwdp-mcp -- iwdp-mcp
```

Or add to `~/.codex/config.toml`:

```toml
[mcp_servers.iwdp-mcp]
command = "iwdp-mcp"
```
</details>

<details id="antigravity">
<summary><strong>Antigravity</strong></summary>

Add to `~/.gemini/antigravity/mcp_config.json`:

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```
</details>

<details id="jetbrains-ides">
<summary><strong>JetBrains IDEs</strong></summary>

Go to **Settings → Tools → AI Assistant → Model Context Protocol (MCP) → Add**, or add the following JSON config:

```json
{
  "mcpServers": {
    "iwdp-mcp": {
      "command": "iwdp-mcp"
    }
  }
}
```
</details>

### Example Prompts

Once the MCP server is configured, you can ask your AI assistant things like:

**Getting started**
```
Connect to my iPhone's Safari and show me what tabs are open
Take a screenshot of the current page
What's the page title and URL?
```

**JavaScript & DOM**
```
Run document.querySelectorAll('a') and list all links on the page
Find all elements with class "error" and show their text content
Get the full DOM tree of the page
What event listeners are attached to the submit button?
Highlight the navigation bar so I can see its bounds
```

**CSS & Styles**
```
What CSS rules are applied to the header element?
Show me the computed styles for the main content area
Force the :hover state on the dropdown menu
Change the background color of .hero to #f0f0f0
List all stylesheets loaded on this page
```

**Network**
```
Monitor network requests while I interact with the page
Show me the response body of that failed API call
Intercept all requests to api.example.com and log them
Block all requests to analytics.google.com
Set a custom Authorization header for all requests
Throttle the network to simulate a slow 3G connection
Disable the browser cache and reload the page
```

**Storage & Cookies**
```
Show me all cookies including httpOnly and secure ones
Set a session cookie named "debug" with value "true"
Delete the authentication cookie
What's in localStorage for this site?
Set a localStorage item "theme" to "dark"
List all IndexedDB databases and their object stores
Show me the data in the "users" object store
```

**Debugging**
```
Set a breakpoint in main.js at line 42
Pause execution and show me the call stack
Step through the code and show variable values
What's the source of the script at bundle.js?
Search for "addEventListener" across all loaded scripts
Break on any uncaught exceptions
Set a DOM breakpoint — break when the #content div is modified
```

**Performance & Profiling**
```
Start a timeline recording while I navigate through the app
Take a heap snapshot and find potential memory leaks
Profile the CPU usage while this animation runs
Track memory usage while I scroll through the feed
Force a garbage collection
```

**Advanced**
```
List all compositing layers and why they were composited
Show me all active CSS animations on the page
Capture the canvas content as an image
Get the TLS certificate details for this page
Check if there are any web workers running
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
| `iwdp_status` | Check/auto-start ios-webkit-debug-proxy |
| `restart_iwdp` | Restart iwdp after a crash (e.g., heap snapshot) |
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
