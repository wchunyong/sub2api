package mcp

type ErrorCode struct {
	Code    int
	Message string
}

var (
	ErrParse          = ErrorCode{Code: -32700, Message: "Parse error"}
	ErrInvalidRequest = ErrorCode{Code: -32600, Message: "Invalid Request"}
	ErrMethodNotFound = ErrorCode{Code: -32601, Message: "Method not found"}
	ErrInvalidParams  = ErrorCode{Code: -32602, Message: "Invalid params"}
	ErrInternal       = ErrorCode{Code: -32603, Message: "Internal error"}
	ErrUnknownTool    = ErrorCode{Code: -32001, Message: "Unknown tool"}
)
