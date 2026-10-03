# Privacy policy

iwdp-mcp runs entirely on your machine. It has no server of its own, collects no
telemetry, and sends nothing to the author or to any third party.

## Network access

* On first run, `scripts/run.sh` downloads the server binary for the installed
  plugin version from this repository's GitHub releases, together with the
  release's checksums file. GitHub sees that request like any other download.
* After that, the server talks only to `localhost`: to ios-webkit-debug-proxy,
  which it may start for you, and through it to Safari on a device or simulator
  you have connected and enabled Web Inspector on.

## What it reads

Only what you or Claude ask for through its tools, from the Safari page being
debugged. Depending on the tool, that can include the page's DOM, styles, console
output, network requests and responses, cookies (including httpOnly ones), local
and session storage, IndexedDB, and screenshots. That data can contain personal
information if the page you are debugging holds any.

## What it writes

* The downloaded binary, under the plugin's `bin/` directory.
* Screenshots, heap snapshots and large JSON results, under
  `$TMPDIR/iwdp-mcp/`, so they can be handed to Claude as files.
* Changes to the debugged page when a tool is asked to make them, such as
  navigating, running JavaScript, editing the DOM or CSS, or setting cookies and
  storage.

## What it passes to Claude

Tool results go back to Claude Code, which handles them under Anthropic's terms,
not this plugin's. Avoid debugging pages that hold data you do not want in your
session.

## Contact

Questions go to [GitHub issues](https://github.com/nnemirovsky/iwdp-mcp/issues).