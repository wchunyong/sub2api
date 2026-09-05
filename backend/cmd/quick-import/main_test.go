package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStdinCatalogPreservesLargeNumberAndRejectsTrailingJSON(t *testing.T) {
	t.Setenv("CODEX_HOME", "")
	for _, suffix := range []string{"", " {}"} {
		root := t.TempDir()
		var out, stderr bytes.Buffer
		body := `{"version":1,"agent":"codex","api_key":"test-key","base_url":"https://example.com/v1","model":"chosen","protocol":"openai","codex_manifest":{"models":[{"slug":"chosen","large":9007199254740993}]}}` + suffix
		code := run([]string{"install", "--agent", "codex", "--stdin", "--home", root, "--skip-client-check"}, strings.NewReader(body), &out, &stderr)
		if suffix != "" {
			if code == 0 {
				t.Fatal("accepted trailing JSON")
			}
			continue
		}
		if code != 0 {
			t.Fatal(stderr.String())
		}
		files, err := filepath.Glob(filepath.Join(root, ".sub2api-quick-import", "codex", "models-*.json"))
		if err != nil || len(files) != 1 {
			t.Fatalf("catalog files: %v, %v", files, err)
		}
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "9007199254740993") {
			t.Fatalf("lost precision: %s", data)
		}
	}
}

func TestCleanDeclined(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := run([]string{"clean", "--agent", "claude", "--home", t.TempDir()}, strings.NewReader("n\n"), &out, &stderr); code != 0 {
		t.Fatalf("declined clean failed: %s", stderr.String())
	}
	if strings.Contains(out.String(), "Restored") {
		t.Fatal("declined clean claimed success")
	}
}

func TestSkipClientCheckRequiresIsolatedStdin(t *testing.T) {
	var out, stderr bytes.Buffer
	if run([]string{"install", "--agent", "claude", "--skip-client-check"}, strings.NewReader(""), &out, &stderr) == 0 {
		t.Fatal("unsafe preflight bypass accepted")
	}
	if !strings.Contains(stderr.String(), "--stdin and --home") {
		t.Fatal(stderr.String())
	}
}

func TestStdinDoesNotExposeInvalidPayload(t *testing.T) {
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		t.Setenv(key, "")
	}
	var out, stderr bytes.Buffer
	if run([]string{"install", "--agent", "claude", "--stdin", "--home", t.TempDir(), "--skip-client-check"}, strings.NewReader(`{"api_key":"super-secret"`), &out, &stderr) == 0 {
		t.Fatal("invalid JSON accepted")
	}
	if strings.Contains(stderr.String(), "super-secret") {
		t.Fatal("credential leaked")
	}
}
