# Sub2API Image MCP

Sub2API exposes image generation to tool-based agents through an authenticated
MCP endpoint:

```text
POST /mcp
Authorization: Bearer <Sub2API API key>
Content-Type: application/json
```

The MCP endpoint is part of the main Sub2API gateway. It uses the same API key,
user, group, model allowlist, image permission, balance, routing, billing and
audit path as `/v1/images/generations` and `/v1/images/edits`. It does not
accept an OpenAI upstream key from the client.

## Transport

The first implementation uses JSON-RPC over HTTP. The server supports:

- `initialize`
- `tools/list`
- `tools/call`

Unknown methods return JSON-RPC `-32601`. Unknown tools return `-32001`.
Invalid parameters return `-32602`.

## Tools

`generate_image` is available for text-to-image requests.

```json
{
  "prompt": "draw a small red house",
  "model": "gpt-image-2",
  "size": "1024x1024",
  "quality": "high",
  "output_format": "png",
  "n": 1
}
```

The tool converts this payload to the existing OpenAI-compatible images API and
returns a text MCP result containing JSON metadata:

```json
{
  "url": "https://gateway.example.test/images/result.png",
  "mime_type": "image/png",
  "model": "gpt-image-2",
  "image_count": 1,
  "revised_prompt": ""
}
```

`edit_image` is available for URL or data URL based image editing.

```json
{
  "image": "data:image/png;base64,QUJD",
  "prompt": "replace the background",
  "model": "gpt-image-2",
  "size": "1024x1024",
  "quality": "high",
  "output_format": "png"
}
```

The MCP layer converts this payload to `/v1/images/edits` JSON format:
`images: [{"image_url": "..."}]`. First release MCP editing does not accept
multipart file uploads directly; clients should provide a reachable image URL or
data URL. `file://` and other local path schemes are rejected by the MCP layer.

## Security

API keys must be sent in headers, not query parameters. The quick installer
writes the same Sub2API API key that it already uses for the agent provider
configuration into the agent's MCP header configuration.

The MCP layer rejects non-OpenAI platform groups for the first release and
checks the group's model allowlist before it invokes the images handler. The
inner images handler still performs image permission, billing eligibility,
quota, concurrency, routing and upstream failover checks.

MCP requests must use `Content-Type: application/json`. `generate_image.n` is
limited to `1..4` at the MCP boundary before the request reaches the image
gateway.

## Quick Import

Quick import registers the image MCP server by default when it writes agent
configuration. The payload fields are:

```json
{
  "mcp_endpoint": "https://gateway.example.test/mcp",
  "mcp_image_tools_enabled": true
}
```

If `mcp_endpoint` is omitted, the installer derives it from `base_url`. For
example, `https://gateway.example.test/v1` becomes
`https://gateway.example.test/mcp`.

Set `mcp_image_tools_enabled` to `false` only when a client should receive the
normal API key/model configuration without MCP registration.

The installer only updates the Sub2API-managed MCP entry:

- Codex: `mcp_servers.sub2api_image`
- Claude Code: `mcpServers.sub2api-image` in the normal Claude settings file
  and in `~/.claude.json` for user-scope MCP discovery
- OpenCode: `mcp.sub2api_image`

Cleanup removes only entries recorded in the Sub2API recovery journal. If a user
edits the managed entry later, cleanup reports a conflict and preserves the
current configuration.
