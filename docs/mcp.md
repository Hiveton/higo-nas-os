# HiGoOS MCP Server

HiGoOS exposes its full control surface to AI clients through the
[Model Context Protocol](https://modelcontextprotocol.io). Every `/api/v1` REST
endpoint is published 1:1 as an MCP tool (~185 tools), so assistants and agents
can drive files, storage, Docker, downloads, video, music, media, monitoring,
remote access, accounts, security, and the governance surfaces.

## Two ways to connect

The same tool catalog (`internal/mcp`) is served two ways, both backed by the
typed REST client in `internal/apiclient`:

| Transport | Who uses it | How |
| --- | --- | --- |
| **Embedded Streamable HTTP** at `/mcp` | in-product AI, remote MCP clients | Mounted inside `higo-api`. Tool calls dispatch in-process into the same handlers (no network hop) and forward the caller's session/Authorization so they run as the real actor. |
| **stdio binary** `higo-mcp` | Claude Desktop / Claude Code | A standalone process that proxies every tool call to a running `higo-api` over HTTP. |

## Configuration

`higo-api` (embedded endpoint):

| Env | Default | Meaning |
| --- | --- | --- |
| `HIGO_MCP_ENABLED` | `true` | Mount the `/mcp` endpoint. |
| `HIGO_MCP_DOMAINS` | _(empty = all)_ | Comma-separated domain allowlist, e.g. `files,storage,docker`. Reduces the tool count a client sees. |

`higo-mcp` (stdio binary):

| Env | Default | Meaning |
| --- | --- | --- |
| `HIGO_MCP_API_BASE` | `http://127.0.0.1:8080` | Base URL of the HiGoOS API to proxy to. |
| `HIGO_MCP_API_TOKEN` | _(none)_ | Sent as `Authorization` on every call (a bare token is prefixed with `Bearer `). |
| `HIGO_MCP_DOMAINS` | _(empty = all)_ | Same domain allowlist as above. |

## Tool conventions

- **Naming:** `higo.<domain>.<action>` — e.g. `higo.files.search`,
  `higo.docker.containers.restart`, `higo.storage.disks.remove`.
- **Annotations:** read-only tools set `readOnlyHint`; deletes/removes/format/revoke
  set `destructiveHint`. Clients use these to gate confirmation UX.
- **Governance:** medium/high-risk endpoints return a `confirmationId` and an
  impact summary instead of executing. Call the matching `*.confirm` tool
  (e.g. `higo.steward.suggestions.confirm`, `higo.security.risk-actions.confirm`)
  with that id to proceed. The MCP layer adds no governance of its own — all of
  it (auth, risk, audit, rollback) lives in the HTTP handlers the tools call.
- **Binary/stream endpoints** (`*/stream`, `*/download`, `*/poster`, `*/cover`,
  `*/lyrics`, `*/subtitle`, `*/preview`) are exposed as `*.url` tools that return
  the API-relative URL to fetch out-of-band, not raw bytes.
- **Not exposed:** the Docker WebSocket terminal (use `higo.docker.containers.exec`);
  SSE event streams are exposed as plain read tools returning current state.

## Using it

### Claude Desktop / Claude Code (stdio)

Build the binary and point a client at it:

```bash
cd server-go && go build -o higo-mcp ./cmd/higo-mcp
```

MCP client config:

```json
{
  "mcpServers": {
    "higoos": {
      "command": "/path/to/higo-mcp",
      "env": {
        "HIGO_MCP_API_BASE": "http://10.211.55.3:8080",
        "HIGO_MCP_API_TOKEN": "<session-or-api-token>"
      }
    }
  }
}
```

### Embedded HTTP

Point any Streamable HTTP MCP client at `http://<host>:8080/mcp`. Outside
`dev`/`test` the endpoint is behind the same session guard as the API, so send a
valid `higo_session` cookie or `Authorization` header.

## Architecture

```
internal/mcp        tool catalog (one file per domain, ~185 tools)
   │ each tool: typed input struct -> apiclient call -> JSON text result
internal/apiclient  typed client for /api/v1/* (envelope-aware, auth-forwarding)
   ├─ in-process round-tripper  → embedded /mcp in higo-api
   └─ net/http                  → higo-mcp stdio binary
```

Adding a new endpoint = add one `apiclient` method + one `addTool(...)` call in
the domain's `tools_<domain>.go`.
