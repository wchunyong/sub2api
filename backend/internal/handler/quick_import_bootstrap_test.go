package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQuickImportBootstrapShortCommands(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/setup/:target", QuickImportBootstrap)
	for _, tc := range []struct{ path, expected string }{
		{"opencode.ps1", "$Agent = 'opencode'"},
		{"codex.sh", "set -- install codex 'https://example.com'"},
		{"claude-clean.ps1", "$Action = 'clean'"},
		{"opencode-clean.sh", "</dev/tty"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "https://example.com/setup/"+tc.path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), tc.expected) {
			t.Fatalf("bad bootstrap %s: %d", tc.path, w.Code)
		}
		if strings.Contains(w.Body.String(), "python3") || strings.Contains(w.Body.String(), "python ") {
			t.Fatal("Python dependency remains")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("bootstrap should not be cached")
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "https://example.com/setup/invalid.ps1", nil))
	if w.Code != 404 {
		t.Fatal("unknown agent accepted")
	}
}
