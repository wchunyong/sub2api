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

// Optional acceptance test against the user's installed OpenCode binary, isolated
// to a temporary home and a mock gateway; no production credentials or Python.
func TestOpenCodeNativeSmoke(t *testing.T) {
	executable := os.Getenv("QUICK_IMPORT_OPENCODE_TEST_EXE")
	if executable == "" {
		t.Skip("set QUICK_IMPORT_OPENCODE_TEST_EXE for real client smoke")
	}
	root := t.TempDir()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer mock-smoke-key" || body.Model != "mock-model" {
			t.Error("unexpected mock request")
			w.WriteHeader(400)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"mock\",\"object\":\"chat.completion.chunk\",\"model\":\"mock-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"IMPORT_OK\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"mock\",\"object\":\"chat.completion.chunk\",\"model\":\"mock-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	if err := Install(root, Payload{Version: 1, Agent: "opencode", APIKey: "mock-smoke-key", BaseURL: server.URL + "/v1", Model: "mock-model", Protocol: "compatible"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "run", "--format", "json", "Reply IMPORT_OK without using tools.")
	cmd.Dir = root
	for _, entry := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if strings.HasPrefix(key, "OPENCODE_") || strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "OPENAI_") || strings.HasPrefix(key, "XDG_") || key == "HOME" || key == "USERPROFILE" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "HOME="+root, "USERPROFILE="+root, "XDG_CONFIG_HOME="+filepath.Join(root, ".config"), "XDG_DATA_HOME="+filepath.Join(root, ".data"), "XDG_STATE_HOME="+filepath.Join(root, ".state"), "XDG_CACHE_HOME="+filepath.Join(root, ".cache"), "OPENCODE_DISABLE_PROJECT_CONFIG=true", "OPENCODE_DISABLE_AUTOUPDATE=true", `OPENCODE_CONFIG_CONTENT={"permission":{"*":"deny"},"share":"disabled","autoupdate":false,"enabled_providers":["sub2api_quick"]}`)
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "IMPORT_OK") || calls.Load() == 0 {
		t.Fatalf("OpenCode smoke failed: %v %s", err, output)
	}
	if err := Clean(root, "opencode"); err != nil {
		t.Fatal(err)
	}
	text, _, err := readText(filepath.Join(root, paths["opencode"]))
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(text, "opencode")
	if err != nil || get(data, []string{"provider", provider}).Exists || get(data, []string{"model"}).Exists {
		t.Fatal("managed fields were not cleaned")
	}
	t.Logf("OpenCode native import, %d mock calls and cleanup passed", calls.Load())
}
