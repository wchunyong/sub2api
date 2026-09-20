package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/mcp"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type mcpHandlerSavedImage struct {
	key         string
	contentType string
	data        []byte
}

type mcpHandlerFakeImageStorage struct {
	saved []mcpHandlerSavedImage
}

func (f *mcpHandlerFakeImageStorage) Save(_ context.Context, key, contentType string, data []byte) (string, error) {
	f.saved = append(f.saved, mcpHandlerSavedImage{
		key:         key,
		contentType: contentType,
		data:        append([]byte(nil), data...),
	})
	return "https://img.example.test/" + key, nil
}

func TestMaterializeMCPRemoteImageAsDataURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	}))
	defer server.Close()

	result, err := materializeMCPImageResult(context.Background(), mcp.ImageResult{
		URL:      server.URL + "/image.png",
		MIMEType: "image/png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.URL, "data:image/png;base64,") {
		t.Fatalf("expected embedded image data URL, got %q", result.URL)
	}
}

func TestRewriteMCPImageResponseOffloadsB64JSON(t *testing.T) {
	storage := &mcpHandlerFakeImageStorage{}
	uploader := service.NewImageResultUploader(storage, "generated/", 0, nil)
	resolver := func() (*service.ImageResultUploader, bool) {
		return uploader, true
	}
	rawPNG := []byte("\x89PNG\r\n\x1a\nmcp-image")
	body := []byte(`{"created":1,"data":[{"b64_json":"` + base64.StdEncoding.EncodeToString(rawPNG) + `","revised_prompt":"kept"}]}`)

	rewritten, offloaded, err := rewriteMCPImageResponse(context.Background(), resolver, "mcpreq_test", body)
	if err != nil {
		t.Fatal(err)
	}
	if !offloaded {
		t.Fatal("expected MCP image response to be offloaded when image storage is enabled")
	}
	if len(storage.saved) != 1 {
		t.Fatalf("expected one uploaded image, got %#v", storage.saved)
	}
	if storage.saved[0].key != "generated/mcpreq_test-0.png" {
		t.Fatalf("unexpected storage key %q", storage.saved[0].key)
	}
	if storage.saved[0].contentType != "image/png" {
		t.Fatalf("unexpected content type %q", storage.saved[0].contentType)
	}
	if string(storage.saved[0].data) != string(rawPNG) {
		t.Fatalf("uploaded bytes changed: %#v", storage.saved[0].data)
	}
	if strings.Contains(string(rewritten), "b64_json") {
		t.Fatalf("rewritten MCP image response still contains base64: %s", string(rewritten))
	}

	parsed, err := parseOpenAIImageResult(rewritten, "gpt-image-2", 1)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.URL != "https://img.example.test/generated/mcpreq_test-0.png" {
		t.Fatalf("unexpected parsed URL %q", parsed.URL)
	}
	if parsed.RevisedPrompt != "kept" {
		t.Fatalf("revised prompt was not preserved: %#v", parsed)
	}
}

func TestBuildMCPImageRequestForcesBase64Output(t *testing.T) {
	body, err := buildMCPImageRequest("gpt-image-2.5-flare", "draw a puppy", "", "", "png", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if got := payload["response_format"]; got != "b64_json" {
		t.Fatalf("expected MCP image request to force b64_json, got %#v", got)
	}
}

func TestClassifyMCPImageResponse(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		body        string
		want        mcp.ErrorCode
		wantMessage string
	}{
		{
			name:        "permission",
			code:        http.StatusForbidden,
			body:        `{"error":{"type":"permission_error","message":"Image generation is not enabled for this group"}}`,
			want:        mcp.ErrPermission,
			wantMessage: "Image generation is not enabled for this group",
		},
		{
			name: "insufficient balance",
			code: http.StatusForbidden,
			body: `{"error":{"type":"billing_error","code":"INSUFFICIENT_BALANCE","message":"insufficient balance"}}`,
			want: mcp.ErrInsufficient,
		},
		{
			name: "rate limit",
			code: http.StatusTooManyRequests,
			body: `{"error":{"type":"rate_limit_exceeded","message":"slow down"}}`,
			want: mcp.ErrRateLimited,
		},
		{
			name: "upstream",
			code: http.StatusBadGateway,
			body: `{"error":{"type":"upstream_error","message":"upstream failed with Authorization: sk-secret"}}`,
			want: mcp.ErrUpstream,
		},
		{
			name: "no account",
			code: http.StatusServiceUnavailable,
			body: `{"error":{"type":"server_error","message":"No available compatible accounts"}}`,
			want: mcp.ErrNoAccount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyMCPImageResponse(tt.code, []byte(tt.body))
			var toolErr *mcp.ToolError
			if !errors.As(err, &toolErr) {
				t.Fatalf("expected ToolError, got %T", err)
			}
			if toolErr.Code.Code != tt.want.Code {
				t.Fatalf("expected code %d, got %d", tt.want.Code, toolErr.Code.Code)
			}
			wantMessage := tt.wantMessage
			if wantMessage == "" {
				wantMessage = tt.want.Message
			}
			if toolErr.Message != wantMessage {
				t.Fatalf("expected safe message %q, got %q", wantMessage, toolErr.Message)
			}
			if strings.Contains(toolErr.Message, "sk-secret") {
				t.Fatalf("classified error leaked upstream secret: %q", toolErr.Message)
			}
		})
	}
}
