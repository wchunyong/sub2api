package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	if !names["generate_image"] || !names["edit_image"] {
		t.Fatalf("missing image tools: %#v", names)
	}
}

func TestServerCallsGenerateImage(t *testing.T) {
	gateway := &fakeImageGateway{}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name": "generate_image",
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

func TestServerReturnsImageContentForDataURL(t *testing.T) {
	gateway := &fakeImageGateway{result: ImageResult{
		URL:      "data:image/png;base64,QUJD",
		MIMEType: "image/png",
		Model:    "gpt-image-2",
	}}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "generate_image",
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
}

func TestServerDefaultsGenerateImageModel(t *testing.T) {
	gateway := &fakeImageGateway{}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name":      "generate_image",
		"arguments": map[string]any{"prompt": "draw a small red house"},
	})
	serveMCP(t, server, request)

	if gateway.generateInput.Model != DefaultImageModel {
		t.Fatalf("expected default model %q, got %q", DefaultImageModel, gateway.generateInput.Model)
	}
}

func TestServerCallsEditImage(t *testing.T) {
	gateway := &fakeImageGateway{}
	server := NewServer(gateway)
	request := jsonRPCRequest(t, "tools/call", map[string]any{
		"name": "edit_image",
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
		"name":      "generate_image",
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
		"name":      "edit_image",
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
		"name":      "generate_image",
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
		"name":      "generate_image",
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
