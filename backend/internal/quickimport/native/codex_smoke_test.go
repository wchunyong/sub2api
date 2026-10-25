package native

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Optional acceptance test against a real Codex CLI. All client settings and
// requests stay in a temporary home and a local mock gateway.
func TestCodexNativeSmoke(t *testing.T) {
	executable := os.Getenv("QUICK_IMPORT_CODEX_TEST_EXE")
	if executable == "" {
		t.Skip("set QUICK_IMPORT_CODEX_TEST_EXE for real client smoke")
	}
	root := t.TempDir()
	manifestBytes, err := service.BuildCodexModelsManifest([]string{"mock-model"})
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(manifestBytes)
			return
		}
		var body struct {
			Model string `json:"model"`
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" ||
			r.Header.Get("Authorization") != "Bearer mock-smoke-key" ||
			json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "mock-model" {
			t.Errorf("unexpected Codex mock request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"mock request observed","type":"authentication_error"}}`))
	}))
	defer server.Close()
	if err := Install(root, Payload{Version: 1, Agent: "codex", APIKey: "mock-smoke-key", BaseURL: server.URL + "/v1", Model: "mock-model", CodexManifest: manifest}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "exec", "--strict-config", "--skip-git-repo-check", "--ephemeral", "--sandbox", "read-only", "Reply with one word.")
	cmd.Dir = root
	for _, entry := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if strings.HasPrefix(key, "CODEX_") || strings.HasPrefix(key, "OPENAI_") || key == "HOME" || key == "USERPROFILE" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "CODEX_HOME="+filepath.Join(root, ".codex"), "HOME="+root, "USERPROFILE="+root)
	output, _ := cmd.CombinedOutput() // The mock intentionally rejects inference.
	if ctx.Err() != nil || calls.Load() == 0 {
		t.Fatalf("Codex did not reach the configured gateway: %v %s", ctx.Err(), output)
	}
	if err := Clean(root, "codex"); err != nil {
		t.Fatal(err)
	}
	text, _, err := readText(filepath.Join(root, paths["codex"]))
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(text, "codex")
	if err != nil || get(data, []string{"model_providers", provider}).Exists || get(data, []string{"model"}).Exists {
		t.Fatal("managed fields were not cleaned")
	}
	t.Logf("Codex native import, %d mock calls and cleanup passed", calls.Load())
}
