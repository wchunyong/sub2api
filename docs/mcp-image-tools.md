# LianjieAI Image MCP

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

`lianjieai_generate_image` is available for text-to-image requests.

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
returns compact JSON metadata plus a standard MCP `image` content item. The
metadata never contains the inline Base64 payload, which keeps the result below
client event-size limits:

```json
{
  "image_generated": true,
  "image_content": "inline",
  "mime_type": "image/png",
  "model": "gpt-image-2",
  "image_count": 1,
  "revised_prompt": ""
}
```

## Markdown Image URL Mode

Some MCP clients cannot reliably render the standard MCP `image` content item
when the result is carried as inline Base64. For those clients, Sub2API should
support an OSS-backed URL mode: upload the generated image bytes to object
storage and return a Markdown image that points at an HTTPS image URL.

Recommended flow:

```text
server calls image-generation upstream
        ↓
decode Base64 image result into bytes
        ↓
upload bytes to S3-compatible object storage from memory
        ↓
return the uploaded image URL from MCP
        ↓
final answer includes: ![生成的图片](<image URL>)
```

The upload does not need a temporary server-side file. Decode the upstream
`b64_json` payload into memory, detect or keep the response MIME type, and pass
the bytes directly to the existing image-storage abstraction. When the upstream
already returns a URL, the server may either return it as-is when it is known to
be durable and directly renderable, or download and re-host it through the same
storage path when durability or display compatibility matters.

### OSS region and endpoint

Aliyun OSS does not require the bucket to be in mainland China. Choose the
region based on the server and user locations, then measure upload latency and
failure rate:

- Canada server and mostly North America users: prefer testing a US OSS region.
- Mostly mainland China users: mainland OSS can be considered, but test the
  Canada-to-China upload path before relying on it.
- Existing mainland bucket and low image volume: reuse it first, then decide
  whether the measured latency and failure rate justify moving.

For a Canada-hosted Sub2API instance, upload through the public OSS endpoint.
Do not use `-internal` endpoints unless the server is actually running in the
matching Aliyun private network.

Initial OSS deployment target:

```text
S3_ENDPOINT=https://oss-cn-guangzhou.aliyuncs.com
S3_ACCESS_KEY=<set in deployment environment>
S3_SECRET_KEY=<set in deployment environment>
S3_BUCKET=lianjieai-image
S3_REGION=oss-cn-guangzhou
S3_ADDRESSING_STYLE=virtual
```

Do not commit `S3_SECRET_KEY` or real access credentials to the repository. If
the implementation reuses the existing `image_storage` configuration, map these
values to `IMAGE_STORAGE_ENDPOINT`, `IMAGE_STORAGE_ACCESS_KEY_ID`,
`IMAGE_STORAGE_SECRET_ACCESS_KEY`, `IMAGE_STORAGE_BUCKET`,
`IMAGE_STORAGE_REGION`, and `IMAGE_STORAGE_FORCE_PATH_STYLE=false`.

### URL lifetime

Choose the URL strategy before enabling Markdown URL mode:

| Image use case | Recommended access mode |
| --- | --- |
| Public image; historical conversations should keep rendering | Public bucket or CDN/custom HTTPS image domain with stable object URLs |
| Private image; temporary preview is enough | Private bucket plus expiring signed URL |

Signed URLs will stop rendering in historical Markdown after they expire. If
long-term display is required, return a stable HTTPS URL, preferably through a
custom image domain such as:

```markdown
![生成的图片](https://img.example.com/generated/<unique-id>.png)
```

### Renderable response requirements

Uploaded objects must set the correct `Content-Type`, for example `image/png`,
`image/jpeg`, or `image/webp`. For image preview in MCP clients, prefer a custom
HTTPS image domain over the default OSS bucket domain, then verify that the
target client renders the returned Markdown instead of downloading the object.

Object keys should be unique and non-guessable enough for the selected access
model, for example:

```text
generated/{yyyy}/{mm}/{dd}/{request-id}-{image-index}.png
```

The MCP tool result should continue to include compact JSON metadata, but the
image payload mode changes from inline to URL:

```json
{
  "image_generated": true,
  "image_content": "url",
  "image_url": "https://img.example.com/generated/...",
  "mime_type": "image/png",
  "model": "gpt-image-2",
  "image_count": 1,
  "revised_prompt": ""
}
```

The final assistant-visible content should include the Markdown image:

```markdown
![生成的图片](https://img.example.com/generated/...)
```

`lianjieai_edit_image` is available for URL or data URL based image editing.

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

- Codex: `mcp_servers.lianjieai_image`
- Claude Code: `mcpServers.lianjieai-image` in the normal Claude settings file
  and in `~/.claude.json` for user-scope MCP discovery
- OpenCode: `mcp.servers.lianjieai_image`

Cleanup removes only entries recorded in the Sub2API recovery journal. If a user
edits the managed entry later, cleanup reports a conflict and preserves the
current configuration.
