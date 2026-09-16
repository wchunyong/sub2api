package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeImageGateway struct {
	generateInput GenerateImageInput
	editInput     EditImageInput
	err           error
	result        ImageResult
}

func (f *fakeImageGateway) GenerateImage(ctx context.Context, input GenerateImageInput) (ImageResult, error) {
	f.generateInput = input
	if f.err != nil {
		return ImageResult{}, f.err
	}
	if f.result.URL != "" {
		return f.result, nil
	}
	return ImageResult{
		URL:        "https://gateway.example.test/images/result.png",
		MIMEType:   "image/png",
		Width:      1024,
		Height:     1024,
		Model:      input.Model,
		ImageCount: 1,
	}, nil
}

func (f *fakeImageGateway) EditImage(ctx context.Context, input EditImageInput) (ImageResult, error) {
	f.editInput = input
	if f.err != nil {
		return ImageResult{}, f.err
	}
	return ImageResult{
		URL:        "https://gateway.example.test/images/result.png",
		MIMEType:   "image/png",
		Width:      1024,
		Height:     1024,
		Model:      input.Model,
		ImageCount: 1,
	}, nil
}

func TestServerListsImageTools(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	request := jsonRPCRequest(t, "tools/list", nil)
	response := serveMCP(t, server, request)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	result := body["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %s", len(tools), response.Body.String())
	}
	names := map[string]bool{}
	for _, item := range tools {
		tool := item.(map[string]any)
		names[tool["name"].(string)] = true
		if tool["inputSchema"] == nil {
			t.Fatalf("tool %s is missing inputSchema", tool["name"])
		}
	}
	if !names["lianjieai_generate_image"] || !names["lianjieai_edit_image"] {
		t.Fatalf("missing image tools: %#v", names)
	}
}

func TestServerCallsGenerateImage(t *testing.T) {
	gateway := &fakeImageGateway{}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name": "lianjieai_generate_image",
		"arguments": map[string]any{
			"prompt":        "draw a small red house",
			"model":         "gpt-image-2",
			"size":          "1024x1024",
			"quality":       "high",
			"output_format": "png",
			"n":             1,
		},
	})
	response := serveMCP(t, server, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	if gateway.generateInput.Prompt != "draw a small red house" || gateway.generateInput.Model != "gpt-image-2" || gateway.generateInput.N != 1 {
		t.Fatalf("bad gateway input: %#v", gateway.generateInput)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	result := body["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !bytes.Contains([]byte(text), []byte("https://gateway.example.test/images/result.png")) {
		t.Fatalf("tool result did not include image URL: %s", text)
	}
	if !bytes.Contains([]byte(text), []byte(`"mime_type":"image/png"`)) {
		t.Fatalf("tool result did not include MIME metadata: %s", text)
	}
}

func TestToolResultIncludesResourceLinkForRemoteImage(t *testing.T) {
	result := toolTextResult(ImageResult{
		URL:      "https://gateway.example.test/images/result.png",
		MIMEType: "image/png",
	})
	content := result["content"].([]map[string]any)
	link := content[len(content)-1]
	if link["type"] != "resource_link" || link["uri"] != "https://gateway.example.test/images/result.png" {
		t.Fatalf("expected remote image resource link, got %#v", link)
	}
	if link["mimeType"] != "image/png" {
		t.Fatalf("expected image MIME type, got %#v", link["mimeType"])
	}
}

func TestServerReturnsImageContentForDataURL(t *testing.T) {
	gateway := &fakeImageGateway{result: ImageResult{
		URL:      "data:image/png;base64,QUJD",
		MIMEType: "image/png",
		Model:    "gpt-image-2",
	}}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "lianjieai_generate_image",
		"arguments": map[string]any{"prompt": "draw a small red house"},
	})
	response := serveMCP(t, server, request)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	content := body["result"].(map[string]any)["content"].([]any)
	foundImage := false
	for _, item := range content {
		entry := item.(map[string]any)
		if entry["type"] == "image" && entry["data"] == "QUJD" && entry["mimeType"] == "image/png" {
			foundImage = true
		}
	}
	if !foundImage {
		t.Fatalf("expected MCP image content, got %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "data:image/png;base64,QUJD") {
		t.Fatalf("tool text must not duplicate inline image data: %s", response.Body.String())
	}
}

func TestServerCompactsLargeInlineImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1400, 1400))
	seed := uint32(0x12345678)
	for y := 0; y < 1400; y++ {
		for x := 0; x < 1400; x++ {
			seed ^= seed << 13
			seed ^= seed >> 17
			seed ^= seed << 5
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(seed),
				G: uint8(seed >> 8),
				B: uint8(seed >> 16),
				A: 255,
			})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	if encoded.Len() <= maxMCPInlineImageBytes {
		t.Fatalf("test image is not large enough: %d bytes", encoded.Len())
	}

	result := toolTextResult(ImageResult{
		URL:      "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes()),
		MIMEType: "image/png",
	})
	content := result["content"].([]map[string]any)
	if len(content) != 2 {
		t.Fatalf("expected compact text and image content, got %#v", content)
	}
	text := content[0]["text"].(string)
	if strings.Contains(text, "data:image/") {
		t.Fatalf("large inline image leaked into text content: %s", text)
	}
	if !strings.Contains(text, `"mime_type":"image/jpeg"`) {
		t.Fatalf("text metadata did not match compact image MIME type: %s", text)
	}
	imageContent := content[1]
	if imageContent["type"] != "image" || imageContent["mimeType"] != "image/jpeg" {
		t.Fatalf("expected compact JPEG image content, got %#v", imageContent)
	}
	data := imageContent["data"].(string)
	if len(data) > maxMCPInlineImageBytes*2 {
		t.Fatalf("encoded image remains too large: %d bytes", len(data))
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		t.Fatalf("compact image is not valid base64: %v", err)
	}
}

func TestServerLeavesOmittedGenerateImageModelUnresolved(t *testing.T) {
	gateway := &fakeImageGateway{}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "lianjieai_generate_image",
		"arguments": map[string]any{"prompt": "draw a small red house"},
	})
	serveMCP(t, server, request)

	if gateway.generateInput.Model != "" {
		t.Fatalf("expected omitted model to remain unresolved, got %q", gateway.generateInput.Model)
	}
}

func TestServerCallsEditImage(t *testing.T) {
	gateway := &fakeImageGateway{}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name": "lianjieai_edit_image",
		"arguments": map[string]any{
			"image":         "data:image/png;base64,QUJD",
			"prompt":        "replace the background",
			"model":         "gpt-image-2",
			"size":          "1024x1024",
			"quality":       "high",
			"output_format": "png",
		},
	})
	response := serveMCP(t, server, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	if gateway.editInput.Image != "data:image/png;base64,QUJD" ||
		gateway.editInput.Prompt != "replace the background" ||
		gateway.editInput.Model != "gpt-image-2" {
		t.Fatalf("bad edit input: %#v", gateway.editInput)
	}
}

func TestServerRejectsInvalidGenerateImageCount(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "lianjieai_generate_image",
		"arguments": map[string]any{"prompt": "draw a small red house", "n": 5},
	})
	response := serveMCP(t, server, request)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"].(float64) != float64(ErrInvalidParams.Code) {
		t.Fatalf("unexpected error: %#v", errObj)
	}
}

func TestServerRejectsInvalidEditImageReference(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "lianjieai_edit_image",
		"arguments": map[string]any{"image": "file:///etc/passwd", "prompt": "replace the background"},
	})
	response := serveMCP(t, server, request)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"].(float64) != float64(ErrInvalidParams.Code) {
		t.Fatalf("unexpected error: %#v", errObj)
	}
}

func TestServerRejectsNonJSONContentType(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(jsonRPCRequest(t, "tools/list", nil)))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerMapsToolErrorsToStableCodes(t *testing.T) {
	server := NewServer(&fakeImageGateway{err: NewToolError(ErrInsufficient, "", nil)})
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "lianjieai_generate_image",
		"arguments": map[string]any{"prompt": "draw a house"},
	})
	response := serveMCP(t, server, request)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj := body["error"].(map[string]any)
	if got := int(errObj["code"].(float64)); got != ErrInsufficient.Code {
		t.Fatalf("expected %d, got %d: %s", ErrInsufficient.Code, got, response.Body.String())
	}
	if got := errObj["message"].(string); got != ErrInsufficient.Message {
		t.Fatalf("unexpected safe message %q", got)
	}
}

func TestServerDoesNotLeakUnknownGatewayErrors(t *testing.T) {
	server := NewServer(&fakeImageGateway{err: errors.New("upstream Authorization: sk-secret")})
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "lianjieai_generate_image",
		"arguments": map[string]any{"prompt": "draw a house"},
	})
	response := serveMCP(t, server, request)
	if bytes.Contains(response.Body.Bytes(), []byte("sk-secret")) {
		t.Fatalf("gateway error leaked credentials: %s", response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if got := int(body["error"].(map[string]any)["code"].(float64)); got != ErrInternal.Code {
		t.Fatalf("expected generic internal code, got %d", got)
	}
}

func TestServerRejectsOversizedRequest(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	body := bytes.Repeat([]byte("x"), int(maxRequestBodyBytes)+1)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if got := int(payload["error"].(map[string]any)["code"].(float64)); got != ErrRequestTooLarge.Code {
		t.Fatalf("expected %d, got %d: %s", ErrRequestTooLarge.Code, got, rec.Body.String())
	}
}

func TestServerAcceptsInitializedNotification(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	request := jsonRPCRequest(t, "notifications/initialized", map[string]any{})
	response := serveMCP(t, server, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != nil {
		t.Fatalf("initialized notification returned error: %s", response.Body.String())
	}
}

func TestServerDoesNotRespondToInitializedNotificationWithoutID(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := serveMCP(t, server, body)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for notification, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != "" {
		t.Fatalf("notification should not have a response body: %q", response.Body.String())
	}
}

func TestServerRejectsUnknownTool(t *testing.T) {
	server := NewServer(&fakeImageGateway{})
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "delete_everything",
		"arguments": map[string]any{},
	})
	response := serveMCP(t, server, request)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"].(float64) != float64(ErrUnknownTool.Code) {
		t.Fatalf("unexpected error: %#v", errObj)
	}
}

func jsonRPCRequest(t *testing.T, method string, params any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func serveMCP(t *testing.T, server *Server, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return rec
}
