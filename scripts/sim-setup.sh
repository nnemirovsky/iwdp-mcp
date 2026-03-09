#!/usr/bin/env bash
#
# Boot an iOS Simulator, open Safari to a test page, and start ios_webkit_debug_proxy.
# Exports IWDP_SIM_WS_URL to stdout (last line) for tests to use.
#
# Usage:
#   eval "$(./scripts/sim-setup.sh)"           # sets IWDP_SIM_WS_URL
#   ./scripts/sim-setup.sh --teardown          # clean shutdown
#
set -euo pipefail

PIDFILE="/tmp/iwdp-sim-test.pid"
SIMFILE="/tmp/iwdp-sim-test.device"

teardown() {
  if [ -f "$PIDFILE" ]; then
    kill "$(cat "$PIDFILE")" 2>/dev/null || true
    rm -f "$PIDFILE"
  fi
  if [ -f "$SIMFILE" ]; then
    xcrun simctl shutdown "$(cat "$SIMFILE")" 2>/dev/null || true
    rm -f "$SIMFILE"
  fi
}

if [ "${1:-}" = "--teardown" ]; then
  teardown
  exit 0
fi

# Teardown any previous run.
teardown

# Kill any existing iwdp processes to avoid port conflicts.
pkill -f ios_webkit_debug_proxy 2>/dev/null || true
sleep 1

# 1. Find an available iPhone simulator.
DEVICE_NAME=$(xcrun simctl list devices available --json | \
  python3 -c "
import json, sys
data = json.load(sys.stdin)
for runtime, devs in sorted(data['devices'].items(), reverse=True):
    if 'iOS' not in runtime and 'SimRuntime.iOS' not in runtime:
        continue
    for d in devs:
        if 'iPhone' in d['name'] and d['isAvailable']:
            print(d['name'])
            sys.exit(0)
print('', end='')
")

if [ -z "$DEVICE_NAME" ]; then
  echo "ERROR: No available iPhone simulator found." >&2
  echo "Install one with: xcodebuild -downloadPlatform iOS" >&2
  exit 1
fi

echo "Booting simulator: $DEVICE_NAME" >&2
echo "$DEVICE_NAME" > "$SIMFILE"

# 2. Boot and wait.
xcrun simctl bootstatus "$DEVICE_NAME" -b 2>&1 | while read -r line; do
  echo "  [sim] $line" >&2
done

# 3. Open Safari to a test page.
echo "Opening Safari..." >&2
xcrun simctl openurl booted "https://example.com"
sleep 3

# 4. Find the webinspectord socket.
SOCKET_PATH=""
for i in $(seq 1 10); do
  SOCKET_PATH=$(lsof -aUc launchd_sim 2>/dev/null \
    | grep 'com.apple.webinspectord_sim.socket' \
    | head -1 | awk '{print $NF}') || true
  if [ -n "$SOCKET_PATH" ]; then
    break
  fi
  echo "  Waiting for webinspectord socket (attempt $i)..." >&2
  sleep 1
done

if [ -z "$SOCKET_PATH" ]; then
  echo "ERROR: Could not find webinspectord_sim.socket" >&2
  echo "The simulator may not have Safari Web Inspector available." >&2
  teardown
  exit 1
fi
echo "Found socket: $SOCKET_PATH" >&2

# 5. Start ios_webkit_debug_proxy (redirect stdout/stderr to stderr to avoid
#    contaminating the eval-able output on stdout).
ios_webkit_debug_proxy -s "unix:$SOCKET_PATH" --no-frontend >&2 2>&1 &
IWDP_PID=$!
echo "$IWDP_PID" > "$PIDFILE"

# Wait for iwdp to become responsive.
for i in $(seq 1 15); do
  if curl -sf http://localhost:9221/json > /dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 15 ]; then
    echo "ERROR: iwdp not responding on port 9221 after 15 seconds" >&2
    teardown
    exit 1
  fi
  sleep 1
done

# 6. Find the WebSocket URL by scanning all device ports (not just 9222,
#    since a physical device may also be connected and shift port assignments).
WS_URL=""
for i in $(seq 1 15); do
  WS_URL=$(curl -sf http://localhost:9221/json 2>/dev/null | python3 -c "
import json, sys, urllib.request
try:
    devices = json.load(sys.stdin)
    for dev in devices:
        url = dev.get('url', '')
        if not url:
            continue
        if not url.startswith('http'):
            url = 'http://' + url
        try:
            resp = urllib.request.urlopen(url + '/json', timeout=2)
            pages = json.loads(resp.read())
            for p in pages:
                ws = p.get('webSocketDebuggerUrl', '')
                if ws:
                    print(ws)
                    sys.exit(0)
        except:
            continue
except:
    pass
print('', end='')
" 2>/dev/null) || true
  if [ -n "$WS_URL" ]; then
    break
  fi
  echo "  Waiting for debuggable page (attempt $i)..." >&2
  sleep 2
done

if [ -z "$WS_URL" ]; then
  echo "ERROR: No debuggable Safari page found" >&2
  echo "Devices:" >&2
  curl -sf http://localhost:9221/json 2>/dev/null | python3 -m json.tool >&2 || true
  teardown
  exit 1
fi

echo "Simulator ready. WebSocket URL: $WS_URL" >&2
echo "export IWDP_SIM_WS_URL=\"$WS_URL\""
