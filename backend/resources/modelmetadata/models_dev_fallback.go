package modelmetadata

import _ "embed"

// ModelsDevFallbackJSON contains compact provider-agnostic capability metadata
// for common relay model IDs when models.dev is unavailable at runtime.
//
//go:embed models-dev-fallback.json
var ModelsDevFallbackJSON []byte
