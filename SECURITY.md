# Security policy

## Reporting a vulnerability

Report privately through GitHub, not in a public issue:

**[Open a private security advisory](https://github.com/nnemirovsky/iwdp-mcp/security/advisories/new)**

The report is visible only to the maintainer ([@nnemirovsky](https://github.com/nnemirovsky)) until an advisory is published, so nothing sensitive is exposed while a fix is prepared. Expect an acknowledgement within a week.

If the form is unavailable, open a normal issue that says only "I would like a private channel for a security report", with no details, and a private advisory will be opened for you.

Never attach real credentials, tokens, personal data or private files to a report. Describe the shape of the problem instead.

## Supported versions

Only the latest release gets security fixes.

## Scope

iwdp-mcp drives Safari through ios-webkit-debug-proxy on your own machine. It is meant to talk only to `localhost` and to write only to its own temp and install directories. A vulnerability is a way to break that, for example:

- a tool, or the `select_page` URL check, that can be made to connect to a host other than `localhost`, `127.0.0.1` or `::1`
- `scripts/run.sh` running a binary whose checksum does not match the release, or one fetched from anywhere but this repository's releases
- a tool writing outside `$TMPDIR/iwdp-mcp/`, the plugin's `bin/` directory, or the output path you passed to the CLI
- the server or CLI running a local command other than starting or stopping `ios_webkit_debug_proxy`
- page data such as cookies, storage or network bodies sent anywhere other than back to the MCP client or CLI that asked for it

Out of scope:

- what the tools are built to do when asked, such as reading httpOnly cookies, running JavaScript in the page or changing its DOM, cookies and storage
- what an AI client decides to do with tool results, including instructions it picks up from a debugged page
- bugs in ios-webkit-debug-proxy, WebKit or Safari. Report those upstream.
- anything that needs an attacker who already runs code on your machine or can reach the proxy's local ports, which ios-webkit-debug-proxy does not authenticate
