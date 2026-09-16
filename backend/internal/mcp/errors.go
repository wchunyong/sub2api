package mcp

type ErrorCode struct {
	Code    int
	Message string
}

var (
	ErrParse           = ErrorCode{Code: -32700, Message: "Parse error"}
	ErrInvalidRequest  = ErrorCode{Code: -32600, Message: "Invalid Request"}
	ErrMethodNotFound  = ErrorCode{Code: -32601, Message: "Method not found"}
	ErrInvalidParams   = ErrorCode{Code: -32602, Message: "Invalid params"}
	ErrInternal        = ErrorCode{Code: -32603, Message: "Internal error"}
	ErrUnknownTool     = ErrorCode{Code: -32001, Message: "Unknown tool"}
	ErrUnauthorized    = ErrorCode{Code: -32010, Message: "Authentication required"}
	ErrPermission      = ErrorCode{Code: -32011, Message: "Image generation is not permitted"}
	ErrModelNotAllowed = ErrorCode{Code: -32012, Message: "Image model is not allowed"}
	ErrInsufficient    = ErrorCode{Code: -32013, Message: "Insufficient balance or quota"}
	ErrUpstream        = ErrorCode{Code: -32014, Message: "Image upstream request failed"}
	ErrRequestTooLarge = ErrorCode{Code: -32015, Message: "MCP request is too large"}
	ErrNoAccount       = ErrorCode{Code: -32016, Message: "No available image account"}
	ErrRateLimited     = ErrorCode{Code: -32017, Message: "Image request rate limited"}
)

// ToolError is an error returned by an image gateway when the caller can be
// given a stable, safe-to-display MCP error code. Message must not contain
// credentials or raw upstream response bodies.
type ToolError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *ToolError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code.Message
}

func (e *ToolError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewToolError(code ErrorCode, message string, cause error) error {
	if message == "" {
		message = code.Message
	}
	return &ToolError{Code: code, Message: message, Cause: cause}
}
