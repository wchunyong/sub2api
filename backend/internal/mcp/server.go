package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	_ "image/png"
	"mime"
	"net/http"
	"net/url"
	"strings"

	xdraw "golang.org/x/image/draw"
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
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params"`
}

type rpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
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
		if req.ID != nil {
			_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})
			return
		}
		w.WriteHeader(http.StatusAccepted)
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
	case "lianjieai_generate_image", "generate_image":
		var input GenerateImageInput
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return nil, ErrInvalidParams, ""
		}
		input.Prompt = strings.TrimSpace(input.Prompt)
		input.Model = strings.TrimSpace(input.Model)
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
	case "lianjieai_edit_image", "edit_image":
		var input EditImageInput
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return nil, ErrInvalidParams, ""
		}
		input.Image = strings.TrimSpace(input.Image)
		input.Prompt = strings.TrimSpace(input.Prompt)
		input.Model = strings.TrimSpace(input.Model)
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
			"name":    "lianjieai-image-mcp",
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
			"name":        "lianjieai_generate_image",
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
			"name":        "lianjieai_edit_image",
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
	body, err := json.Marshal(imageResultSummary(result))
	if err != nil {
		body = []byte(`{"image_generated":true}`)
	}
	content := []map[string]any{
		{"type": "text", "text": string(body)},
	}
	if data, mimeType, ok := decodeImageDataURL(result.URL, result.MIMEType); ok {
		content = append(content, map[string]any{
			"type":     "image",
			"data":     data,
			"mimeType": mimeType,
		})
	} else if parsed, err := url.Parse(strings.TrimSpace(result.URL)); err == nil &&
		(parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" {
		content = append(content, map[string]any{
			"type":     "resource_link",
			"uri":      parsed.String(),
			"name":     "generated-image",
			"mimeType": imageMIMEType(result.MIMEType),
		})
	}
	return map[string]any{
		"content": content,
	}
}

func imageResultSummary(result ImageResult) map[string]any {
	summary := map[string]any{
		"image_generated": true,
	}
	if result.MIMEType != "" {
		summary["mime_type"] = result.MIMEType
	}
	if result.Width > 0 {
		summary["width"] = result.Width
	}
	if result.Height > 0 {
		summary["height"] = result.Height
	}
	if result.Model != "" {
		summary["model"] = result.Model
	}
	if result.ImageCount > 0 {
		summary["image_count"] = result.ImageCount
	}
	if result.RevisedPrompt != "" {
		summary["revised_prompt"] = result.RevisedPrompt
	}
	if _, _, ok := decodeImageDataURL(result.URL, result.MIMEType); ok {
		summary["image_content"] = "inline"
	} else if strings.HasPrefix(strings.ToLower(strings.TrimSpace(result.URL)), "data:image/") {
		summary["image_content"] = "inline"
	} else if strings.TrimSpace(result.URL) != "" {
		summary["image_content"] = "resource_link"
		summary["url"] = result.URL
	}
	return summary
}

func imageMIMEType(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "image/") {
		return value
	}
	return "image/png"
}

func decodeImageDataURL(rawURL, fallbackMIMEType string) (string, string, bool) {
	const prefix = "data:"
	if !strings.HasPrefix(strings.ToLower(rawURL), prefix) {
		return "", "", false
	}
	parts := strings.SplitN(rawURL[len(prefix):], ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	metadata := strings.Split(parts[0], ";")
	if len(metadata) < 2 || !strings.EqualFold(metadata[len(metadata)-1], "base64") {
		return "", "", false
	}
	mimeType := strings.TrimSpace(metadata[0])
	if mimeType == "" {
		mimeType = strings.TrimSpace(fallbackMIMEType)
	}
	if mimeType == "" || !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return "", "", false
	}
	data := strings.TrimSpace(parts[1])
	if data == "" {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", "", false
	}
	data, mimeType, ok := compactMCPImage(decoded, mimeType)
	if !ok {
		return "", "", false
	}
	return data, mimeType, true
}

// Codex and other MCP clients cap persisted tool results at roughly 1 MiB.
// Keep the encoded image well below that limit so a valid image is not
// converted into a truncated text result by the client.
const maxMCPInlineImageBytes = 700 << 10

func compactMCPImage(raw []byte, mimeType string) (string, string, bool) {
	if len(raw) <= maxMCPInlineImageBytes {
		return base64.StdEncoding.EncodeToString(raw), mimeType, true
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil || src.Bounds().Empty() {
		return "", "", false
	}

	bounds := src.Bounds()
	scales := []float64{1, 0.75, 0.5, 0.375, 0.25}
	qualities := []int{82, 72, 62, 52}
	for _, scale := range scales {
		width := max(1, int(float64(bounds.Dx())*scale))
		height := max(1, int(float64(bounds.Dy())*scale))
		dst := image.NewRGBA(image.Rect(0, 0, width, height))
		stddraw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, stddraw.Src)
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, stddraw.Over, nil)

		for _, quality := range qualities {
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
				return "", "", false
			}
			if buf.Len() <= maxMCPInlineImageBytes {
				return base64.StdEncoding.EncodeToString(buf.Bytes()), "image/jpeg", true
			}
		}
	}
	return "", "", false
}

func errorResponse(id *json.RawMessage, code ErrorCode) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code.Code, Message: code.Message}}
}

func errorResponseWithMessage(id *json.RawMessage, code ErrorCode, message string) rpcResponse {
	if message == "" {
		return errorResponse(id, code)
	}
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code.Code, Message: message}}
}
