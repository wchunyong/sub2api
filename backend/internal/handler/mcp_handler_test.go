package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/mcp"
)

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
