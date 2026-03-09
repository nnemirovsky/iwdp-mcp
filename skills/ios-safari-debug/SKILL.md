---
name: ios-safari-debug
description: Debug iOS Safari pages via ios-webkit-debug-proxy
---

# iOS Safari Debugging

Use this skill to inspect and debug Safari tabs on a connected iOS device through ios-webkit-debug-proxy (iwdp).

## Prerequisites

1. **Check if iwdp is running** (check the listing port 9221, NOT a device port):
   ```bash
   curl -s http://localhost:9221/json
   ```
   A JSON array of connected devices means it is running.

2. **If not running, start it:**
   ```bash
   ios_webkit_debug_proxy --no-frontend &
   ```
   Wait 1 second, then verify:
   ```bash
   sleep 1 && curl -s http://localhost:9221/json
   ```

3. **An iOS device must be connected** via USB with Safari open and Web Inspector enabled (Settings > Safari > Advanced > Web Inspector).

## Port Layout

- **Port 9221** — listing port: returns all connected devices
- **Port 9222** — first device's pages
- **Port 9223** — second device's pages (and so on)

Each device entry from port 9221 includes a URL field (e.g., `localhost:9222`) indicating which port to query for that device's pages.

## Workflow

1. **List devices** — use `list_devices` to see connected iOS devices and their port assignments.

2. **List pages** — use `list_pages` (optionally with `port` for a specific device) to discover open Safari tabs. Each entry includes a title, URL, and `webSocketDebuggerUrl`.

3. **Select a page** — use `select_page` with the WebSocket URL from the listing to connect to the target tab.

4. **Use debugging tools** for the task at hand. Available tools include:
   - `navigate` — go to a URL
   - `evaluate_script` — run JavaScript in the page
   - `take_screenshot` — capture the page as a base64 PNG
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

## Tips

- Use `list_devices` first if you have multiple iOS devices connected — it shows port assignments.
- `list_pages` defaults to port 9222 (first device). Pass the port from `list_devices` for other devices.
- `get_cookies` returns httpOnly and secure cookies that `document.cookie` cannot access.
- Network/console monitoring only captures events while active — enable before triggering traffic.
- `evaluate_script` runs arbitrary JS in the page context — use it for anything the specialized tools don't cover.
