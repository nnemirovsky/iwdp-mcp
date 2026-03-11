---
name: ios-safari-debug
description: Debug iOS Safari pages via ios-webkit-debug-proxy
---

# iOS Safari Debugging

Use this skill to inspect and debug Safari tabs on a connected iOS device through ios-webkit-debug-proxy (iwdp).

## Prerequisites

1. **Check iwdp status** — call `iwdp_status` first. It checks if ios-webkit-debug-proxy is running and auto-starts it if needed (no manual shell commands required).

2. **An iOS device must be connected** via USB with Safari open and Web Inspector enabled (Settings > Safari > Advanced > Web Inspector).

## Port Layout

- **Port 9221** — listing port: returns all connected devices
- **Port 9222** — first device's pages
- **Port 9223** — second device's pages (and so on)

Each device entry from port 9221 includes a URL field (e.g., `localhost:9222`) indicating which port to query for that device's pages.

## Workflow

1. **Ensure iwdp is running** — call `iwdp_status` to verify (it auto-starts if needed).

2. **List devices** — use `list_devices` to see connected iOS devices and their port assignments.

3. **List pages** — use `list_pages` (optionally with `port` for a specific device) to discover open Safari tabs. Each entry includes a title, URL, and `webSocketDebuggerUrl`.

4. **Select a page** — use `select_page` with the WebSocket URL from the listing to connect to the target tab.

5. **Use debugging tools** for the task at hand. Available tools include:
   - `navigate` — go to a URL
   - `evaluate_script` — run JavaScript in the page
   - `take_screenshot` — capture the page as a PNG file (returns path — use Read to view)
   - `get_document` / `query_selector` / `get_outer_html` — inspect the DOM
   - `click` / `fill` / `type_text` — interact with page elements
   - `get_cookies` / `set_cookie` / `delete_cookie` — manage cookies (incl. httpOnly)
   - `get_local_storage` / `get_session_storage` — read web storage
   - `get_computed_style` / `get_matched_styles` — inspect CSS
   - `network_enable` / `list_network_requests` — monitor network traffic
   - `get_console_messages` — collect console output
   - Debugger, Timeline, Memory, Profiler, and more

## Common Workflows

### Screenshot + Evaluate
```
1. take_screenshot
2. evaluate_script: "document.title"
```

### Network Monitoring
Start monitoring before navigating to capture all requests:
```
1. network_enable
2. navigate to the target URL
3. list_network_requests
4. get_response_body for individual responses
```

### Cookie Inspection
View all cookies including httpOnly ones that JavaScript cannot access:
```
1. get_cookies
2. Examine secure/httpOnly flags, domains, and expiry
```

### DOM Exploration
```
1. get_document (with depth)
2. query_selector to find specific elements
3. get_outer_html to see element markup
4. get_attributes for element attributes
```

### Form Interaction
```
1. query_selector to find the form fields
2. fill each input with a value
3. click the submit button
4. take_screenshot to verify the result
```

## CLI Tool (iwdp-cli)

**Use MCP tools by default.** Only use the CLI when the user explicitly requests it.

The `iwdp-cli` binary provides the same capabilities as the MCP tools but from the command line. Install it with `go install github.com/nnemirovsky/iwdp-mcp/cmd/iwdp-cli@latest`.

### Usage Pattern

```bash
iwdp-cli <command> [args...] [ws-url]
```

The `ws-url` argument (WebSocket URL from `iwdp-cli pages`) is always optional and placed last. If omitted, the CLI auto-connects to the first available page.

### Core Commands

```bash
iwdp-cli devices                    # List connected iOS devices
iwdp-cli pages [port]               # List Safari tabs (default port: 9222)
iwdp-cli status                     # Check if iwdp is running
iwdp-cli eval "document.title"      # Evaluate JavaScript
iwdp-cli navigate "https://..."     # Navigate to URL
iwdp-cli reload [--ignore-cache]    # Reload page
iwdp-cli screenshot [-o file.png]   # Take screenshot
iwdp-cli cookies                    # Show all cookies (incl. httpOnly)
iwdp-cli dom [selector]             # Inspect DOM tree or element
iwdp-cli console                    # Stream console messages (Ctrl+C)
iwdp-cli network                    # Stream network requests (Ctrl+C)
```

### DOM & CSS

```bash
iwdp-cli query-selector-all "div.item"     # Find all matching elements
iwdp-cli get-attributes 5                   # Get node attributes
iwdp-cli get-computed-style 5               # Get computed style
iwdp-cli get-matched-styles 5               # Get matching CSS rules
iwdp-cli highlight-node 5                   # Highlight element in browser
```

### Interaction

```bash
iwdp-cli click "#submit-btn"              # Click element
iwdp-cli fill "#email" "user@example.com" # Fill input
iwdp-cli type-text "hello"                # Type into focused element
```

### Storage

```bash
iwdp-cli set-cookie name val .example.com  # Set cookie
iwdp-cli delete-cookie name "https://..."  # Delete cookie
iwdp-cli get-local-storage "https://..."   # Get localStorage
iwdp-cli get-session-storage "https://..." # Get sessionStorage
```

### Debugger

```bash
iwdp-cli debugger-enable
iwdp-cli set-breakpoint "app.js" 42
iwdp-cli pause / resume / step-over / step-into / step-out
iwdp-cli get-script-source "1"
iwdp-cli eval-on-frame "0" "myVar"
```

### Performance & Profiling

Duration-based commands collect for 3 seconds by default. Use `-d N` to change.

```bash
iwdp-cli timeline-record [-d 5]    # Record timeline events
iwdp-cli memory-track [-d 5]       # Track memory usage
iwdp-cli heap-snapshot              # Take heap snapshot
iwdp-cli heap-gc                    # Force garbage collection
iwdp-cli cpu-profile [-d 5]        # CPU profiling
iwdp-cli script-profile [-d 5]     # Script execution profiling
```

### Full Command List

Run `iwdp-cli help` to see all 90+ commands organized by category.

## Tips

- Use `list_devices` first if you have multiple iOS devices connected. It shows port assignments.
- `list_pages` defaults to port 9222 (first device). Pass the port from `list_devices` for other devices.
- `get_cookies` returns httpOnly and secure cookies that `document.cookie` cannot access.
- Network/console monitoring only captures events while active. Enable before triggering traffic.
- `evaluate_script` runs arbitrary JS in the page context. Use it for anything the specialized tools don't cover.
- Screenshots are saved as PNG files. Use the Read tool to view the returned file path.
- Large results (DOM trees, network logs, JS output) are automatically saved to temp files when they exceed the inline size limit.
