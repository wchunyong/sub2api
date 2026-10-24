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
)

// Optional acceptance test against a real Claude Code executable, using only
// temporary settings and a local mock gateway.
func TestClaudeNativeSmoke(t *testing.T) {
	executable := os.Getenv("QUICK_IMPORT_CLAUDE_TEST_EXE")
	if executable == "" {
		t.Skip("set QUICK_IMPORT_CLAUDE_TEST_EXE for real client smoke")
	}
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/api/hello" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var body struct {
			Model string `json:"model"`
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" ||
			r.Header.Get("Authorization") != "Bearer mock-smoke-key" ||
			json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "mock-model" {
			t.Errorf("unexpected Claude mock request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		calls.Add(1)
		cancel() // The route, bearer token, and model have all been observed.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"mock request observed"}}`))
	}))
	defer server.Close()
	if err := Install(root, Payload{Version: 1, Agent: "claude", APIKey: "mock-smoke-key", BaseURL: server.URL, Model: "mock-model", Models: []Model{{ID: "mock-model", Name: "Mock Model"}}, ClaudeModelPickerSupported: true}); err != nil {
		t.Fatal(err)
	}
	debugPath := filepath.Join(root, "claude-debug.log")
	cmd := exec.CommandContext(ctx, executable, "--print", "--no-session-persistence", "--output-format", "text", "--debug-file", debugPath, "Reply with one word.")
	cmd.Dir = root
	for _, entry := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if strings.HasPrefix(key, "CLAUDE_") || strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "OPENAI_") || key == "HOME" || key == "USERPROFILE" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR="+filepath.Join(root, ".claude"), "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "DISABLE_TELEMETRY=1", "DISABLE_ERROR_REPORTING=1", "DISABLE_AUTOUPDATER=1", "HOME="+root, "USERPROFILE="+root)
	output, _ := cmd.CombinedOutput() // The mock intentionally rejects inference.
	if calls.Load() == 0 {
		debug, _ := os.ReadFile(debugPath)
		if len(debug) > 6000 {
			debug = debug[len(debug)-6000:]
		}
		t.Fatalf("Claude Code did not reach the configured gateway: %v %s\n%s", ctx.Err(), output, debug)
	}
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	text, _, err := readText(filepath.Join(root, paths["claude"]))
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(text, "claude")
	if err != nil || get(data, []string{"env", "ANTHROPIC_AUTH_TOKEN"}).Exists || get(data, []string{"env", "ANTHROPIC_MODEL"}).Exists {
		t.Fatal("managed fields were not cleaned")
	}
	t.Logf("Claude Code native import, %d mock calls and cleanup passed", calls.Load())
}
