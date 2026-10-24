package native

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Optional acceptance test against a real DeepSeek Harness launcher. It checks
// that the web profile can compose the generated patch without contacting an LLM.
func TestDeepSeekHarnessNativeSmoke(t *testing.T) {
	launcher := os.Getenv("QUICK_IMPORT_DSH_LAUNCHER")
	if launcher == "" {
		t.Skip("set QUICK_IMPORT_DSH_LAUNCHER for real client smoke")
	}
	root := t.TempDir()
	if err := Install(root, Payload{Version: 1, Agent: "deepseek-harness", APIKey: "mock-smoke-key", BaseURL: "https://example.test/v1", Model: "mock-model", Models: []Model{{ID: "mock-model", Name: "Mock Model"}}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", launcher, "--profile", "web", "--dump-config")
	cmd.Dir = root
	for _, entry := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if strings.HasPrefix(key, "DSH_") || key == "HOME" || key == "USERPROFILE" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "DSH_HOME="+filepath.Join(root, ".dsh"), "HOME="+root, "USERPROFILE="+root)
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "mock-model") ||
		!strings.Contains(string(output), "https://example.test/v1") ||
		!strings.Contains(string(output), "mock-smoke-key") ||
		!strings.Contains(string(output), "openai-completions") {
		t.Fatalf("DeepSeek Harness did not compose the imported web profile: %v %s", err, output)
	}
	if err := Clean(root, "deepseek-harness"); err != nil {
		t.Fatal(err)
	}
	t.Log("DeepSeek Harness web profile parsed the imported model and cleanup passed")
}
