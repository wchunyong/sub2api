package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

type GenerateImageInput struct {
	Prompt       string `json:"prompt"`
	Model        string `json:"model,omitempty"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	N            int    `json:"n,omitempty"`
}

type EditImageInput struct {
	Image        string `json:"image"`
	Prompt       string `json:"prompt"`
	Model        string `json:"model,omitempty"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

const DefaultImageModel = "gpt-image-2"

type ImageResult struct {
	URL           string `json:"url"`
	MIMEType      string `json:"mime_type"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Model         string `json:"model,omitempty"`
	ImageCount    int    `json:"image_count,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageGateway interface {
	GenerateImage(ctx context.Context, input GenerateImageInput) (ImageResult, error)
	EditImage(ctx context.Context, input EditImageInput) (ImageResult, error)
}

type Server struct {
	imageGateway ImageGateway
}

const maxRequestBodyBytes int64 = 1 << 20

func NewServer(imageGateway ImageGateway) *Server {
	return &Server{imageGateway: imageGateway}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(errorResponse(nil, ErrInvalidRequest))
		return
	}
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_ = json.NewEncoder(w).Encode(errorResponse(nil, ErrInvalidRequest))
		return
	}
	if r.ContentLength > maxRequestBodyBytes {
		_ = json.NewEncoder(w).Encode(errorResponse(nil, ErrRequestTooLarge))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			_ = json.NewEncoder(w).Encode(errorResponse(nil, ErrRequestTooLarge))
			return
		}
		_ = json.NewEncoder(w).Encode(errorResponse(nil, ErrParse))
		return
	}
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		_ = json.NewEncoder(w).Encode(errorResponse(req.ID, ErrInvalidRequest))
		return
	}

	switch req.Method {
	case "initialize":
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: initializeResult()})
	case "tools/list":
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": imageTools()}})
	case "notifications/initialized":
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})
	case "tools/call":
		result, code, message := s.callTool(r.Context(), req.Params)
		if code.Code != 0 {
			_ = json.NewEncoder(w).Encode(errorResponseWithMessage(req.ID, code, message))
			return
		}
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
	default:
		_ = json.NewEncoder(w).Encode(errorResponse(req.ID, ErrMethodNotFound))
	}
}

func (s *Server) callTool(ctx context.Context, raw json.RawMessage) (any, ErrorCode, string) {
	var req struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, ErrInvalidParams, ""
	}
	switch req.Name {
	case "generate_image":
		var input GenerateImageInput
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return nil, ErrInvalidParams, ""
		}
		input.Prompt = strings.TrimSpace(input.Prompt)
		input.Model = strings.TrimSpace(input.Model)
		if input.Model == "" {
			input.Model = DefaultImageModel
		}
		input.Size = strings.TrimSpace(input.Size)
		input.Quality = strings.TrimSpace(input.Quality)
		input.OutputFormat = strings.TrimSpace(input.OutputFormat)
		if input.Prompt == "" {
			return nil, ErrInvalidParams, ""
		}
		if input.N == 0 {
			input.N = 1
		}
		if input.N < 1 || input.N > 4 {
			return nil, ErrInvalidParams, ""
		}
		result, err := s.imageGateway.GenerateImage(ctx, input)
		if err != nil {
			return toolErrorDetails(err)
		}
		return toolTextResult(result), ErrorCode{}, ""
	case "edit_image":
		var input EditImageInput
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return nil, ErrInvalidParams, ""
		}
		input.Image = strings.TrimSpace(input.Image)
		input.Prompt = strings.TrimSpace(input.Prompt)
		input.Model = strings.TrimSpace(input.Model)
		if input.Model == "" {
			input.Model = DefaultImageModel
		}
		input.Size = strings.TrimSpace(input.Size)
		input.Quality = strings.TrimSpace(input.Quality)
		input.OutputFormat = strings.TrimSpace(input.OutputFormat)
		if input.Image == "" || input.Prompt == "" || !isAllowedImageReference(input.Image) {
			return nil, ErrInvalidParams, ""
		}
		result, err := s.imageGateway.EditImage(ctx, input)
		if err != nil {
			return toolErrorDetails(err)
		}
		return toolTextResult(result), ErrorCode{}, ""
	default:
		return nil, ErrUnknownTool, ""
	}
}

func toolErrorDetails(err error) (any, ErrorCode, string) {
	var toolErr *ToolError
	if errors.As(err, &toolErr) && toolErr != nil {
		return nil, toolErr.Code, toolErr.Message
	}
	return nil, ErrInternal, ""
}

func isJSONContentType(contentType string) bool {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func isAllowedImageReference(value string) bool {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "data:image/") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http"
}

func initializeResult() map[string]any {
	return map[string]any{
		"protocolVersion": "2025-06-18",
		"serverInfo": map[string]any{
			"name":    "sub2api-image-mcp",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
	}
}

func imageTools() []map[string]any {
	return []map[string]any{
		{
			"name":        "generate_image",
			"description": "Generate an image through the authenticated Sub2API image gateway.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"prompt"},
				"properties": map[string]any{
					"prompt":        map[string]any{"type": "string"},
					"model":         map[string]any{"type": "string"},
					"size":          map[string]any{"type": "string"},
					"quality":       map[string]any{"type": "string"},
					"output_format": map[string]any{"type": "string"},
					"n":             map[string]any{"type": "integer", "minimum": 1, "maximum": 4},
				},
			},
		},
		{
			"name":        "edit_image",
			"description": "Edit an existing image through the authenticated Sub2API image gateway.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"image", "prompt"},
				"properties": map[string]any{
					"image":         map[string]any{"type": "string"},
					"prompt":        map[string]any{"type": "string"},
					"model":         map[string]any{"type": "string"},
					"size":          map[string]any{"type": "string"},
					"quality":       map[string]any{"type": "string"},
					"output_format": map[string]any{"type": "string"},
				},
			},
		},
	}
}

func toolTextResult(result ImageResult) map[string]any {
	body, err := json.Marshal(result)
	if err != nil {
		body = []byte(fmt.Sprintf(`{"url":%q}`, result.URL))
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(body)},
		},
	}
}

func errorResponse(id any, code ErrorCode) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code.Code, Message: code.Message}}
}

func errorResponseWithMessage(id any, code ErrorCode, message string) rpcResponse {
	if message == "" {
		return errorResponse(id, code)
	}
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code.Code, Message: message}}
}
