package native

import (
	"context"
	"encoding/json"
	"fmt"
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

// Optional acceptance test against a real Pi CLI using isolated settings and
// a local mock gateway. Set QUICK_IMPORT_PI_LAUNCHER to pi-launcher.js.
func TestPiNativeSmoke(t *testing.T) {
	launcher := os.Getenv("QUICK_IMPORT_PI_LAUNCHER")
	if launcher == "" {
		t.Skip("set QUICK_IMPORT_PI_LAUNCHER for real client smoke")
	}
	root := t.TempDir()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" ||
			r.Header.Get("Authorization") != "Bearer mock-smoke-key" ||
			json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "mock-model" {
			t.Errorf("unexpected Pi mock request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"mock\",\"object\":\"chat.completion.chunk\",\"model\":\"mock-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"IMPORT_OK\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"mock\",\"object\":\"chat.completion.chunk\",\"model\":\"mock-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	if err := Install(root, Payload{Version: 1, Agent: "pi", APIKey: "mock-smoke-key", BaseURL: server.URL + "/v1", Model: "mock-model"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", launcher, "--print", "--provider", "lianjieai", "--model", "mock-model", "--no-tools", "--no-extensions", "--no-mcp", "--no-skills", "--no-context-files", "--no-session", "--offline", "Reply IMPORT_OK without using tools.")
	cmd.Dir = root
	for _, entry := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if strings.HasPrefix(key, "PI_") || strings.HasPrefix(key, "OPENAI_") || strings.HasPrefix(key, "ANTHROPIC_") || key == "HOME" || key == "USERPROFILE" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "PI_CODING_AGENT_DIR="+filepath.Join(root, ".pi", "agent"), "PI_TELEMETRY=0", "HOME="+root, "USERPROFILE="+root)
	output, err := cmd.CombinedOutput()
	if err != nil || calls.Load() == 0 || !strings.Contains(string(output), "IMPORT_OK") {
		t.Fatalf("Pi smoke failed: %v %s", err, output)
	}
	if err := Clean(root, "pi"); err != nil {
		t.Fatal(err)
	}
	text, _, err := readText(filepath.Join(root, paths["pi"]))
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(text, "pi")
	if err != nil || get(data, []string{"providers", "lianjieai"}).Exists {
		t.Fatal("managed fields were not cleaned")
	}
	t.Logf("Pi native import, %d mock calls and cleanup passed", calls.Load())
}
