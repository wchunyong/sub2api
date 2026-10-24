package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/quickimport/native"
)

func TestSixAgentsInstallAndRestoreFromIsolatedInput(t *testing.T) {
	for _, name := range []string{"CLAUDE_CONFIG_DIR", "ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY", "CODEX_HOME", "OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "XDG_CONFIG_HOME", "PI_CODING_AGENT_DIR", "DSH_HOME"} {
		t.Setenv(name, "")
	}
	steps := map[string]struct{ label, launch string }{
		"claude":           {"Claude Code", "在项目目录运行 claude"},
		"codex":            {"Codex", "在项目目录运行 codex"},
		"deepseek-harness": {"DeepSeek Harness", "运行 dsh web"},
		"workbuddy":        {"WorkBuddy", "打开 WorkBuddy 桌面应用"},
		"opencode":         {"OpenCode", "在项目目录运行 opencode"},
		"pi":               {"Pi", "在项目目录运行 pi"},
	}
	for agent, step := range steps {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			payload := native.Payload{Version: 1, Agent: agent, APIKey: "private-test-key", BaseURL: "https://example.test/v1", Model: "selected-model", Models: []native.Model{{ID: "selected-model", Name: "Selected Model"}}}
			if agent == "claude" {
				payload.BaseURL = "https://example.test"
			}
			if agent == "codex" {
				payload.CodexManifest = map[string]any{"models": []any{map[string]any{"slug": "selected-model"}}}
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			if code := run([]string{"install", "--agent", agent, "--stdin", "--home", root, "--skip-client-check"}, bytes.NewReader(body), &out, &stderr); code != 0 {
				t.Fatalf("install failed: %s", stderr.String())
			}
			if !strings.Contains(out.String(), step.label+" 配置已写入") || !strings.Contains(out.String(), step.launch) || strings.Contains(out.String(), "private-test-key") {
				t.Fatalf("wrong install output: %s", out.String())
			}
			if agent == "opencode" && !strings.Contains(out.String(), "lianjieai_quick/selected-model") {
				t.Fatalf("OpenCode model picker hint is missing: %s", out.String())
			}
			out.Reset()
			stderr.Reset()
			if code := run([]string{"clean", "--agent", agent, "--home", root, "--yes"}, strings.NewReader(""), &out, &stderr); code != 0 {
				t.Fatalf("cleanup failed: %s", stderr.String())
			}
			if !strings.Contains(out.String(), "恢复完成") {
				t.Fatalf("wrong cleanup output: %s", out.String())
			}
		})
	}
}

func TestDeepSeekHarnessDesktopSelectionSurvivesCleanup(t *testing.T) {
	t.Setenv("DSH_HOME", "")
	root := t.TempDir()
	for _, profile := range []string{"web", "desktop"} {
		if err := os.MkdirAll(filepath.Join(root, ".dsh", "profiles", profile), 0700); err != nil {
			t.Fatal(err)
		}
	}
	payload := native.Payload{Version: 1, Agent: "deepseek-harness", APIKey: "private-test-key", BaseURL: "https://example.test/v1", Model: "selected-model", Models: []native.Model{{ID: "selected-model"}}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	args := []string{"install", "--agent", "deepseek-harness", "--dsh-profile", "desktop", "--stdin", "--home", root, "--skip-client-check"}
	if code := run(args, bytes.NewReader(body), &out, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if !strings.Contains(out.String(), "打开 DeepSeek Harness 桌面应用") {
		t.Fatalf("wrong desktop next step: %s", out.String())
	}
	desktopPatch := filepath.Join(root, ".dsh", "profiles", "desktop", "cordis.patch.yml")
	if _, err := os.Stat(desktopPatch); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".dsh", "profiles", "web", "cordis.patch.yml")); !os.IsNotExist(err) {
		t.Fatalf("web profile changed: %v", err)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"clean", "--agent", "deepseek-harness", "--home", root, "--yes"}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if _, err := os.Stat(desktopPatch); !os.IsNotExist(err) {
		t.Fatalf("desktop patch remains: %v", err)
	}
}

func TestDeepSeekHarnessAmbiguousProfileStopsBeforeImport(t *testing.T) {
	root := t.TempDir()
	for _, profile := range []string{"web", "desktop"} {
		if err := os.MkdirAll(filepath.Join(root, ".dsh", "profiles", profile), 0700); err != nil {
			t.Fatal(err)
		}
	}
	var out, stderr bytes.Buffer
	if run([]string{"install", "--agent", "deepseek-harness", "--home", root}, strings.NewReader("\n"), &out, &stderr) == 0 {
		t.Fatal("ambiguous profile accepted")
	}
	if !strings.Contains(out.String(), "web 或 desktop") || !strings.Contains(stderr.String(), "尚未兑换配置码") {
		t.Fatalf("unclear profile error: %s %s", out.String(), stderr.String())
	}
}

func TestCleanReportsConflictWithoutCredentials(t *testing.T) {
	root := t.TempDir()
	if err := native.Install(root, native.Payload{Version: 1, Agent: "codex", APIKey: "private-test-key", BaseURL: "https://example.test/v1", Model: "imported"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".codex", "config.toml")
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.ReplaceAll(string(text), "private-test-key", "changed-private-key")
	if err := os.WriteFile(path, []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if run([]string{"clean", "--agent", "codex", "--home", root, "--yes"}, strings.NewReader(""), &out, &stderr) == 0 {
		t.Fatal("accepted changed credentials")
	}
	if !strings.Contains(stderr.String(), "model_providers.lianjieai_quick") {
		t.Fatal("missing conflict field")
	}
	if strings.Contains(stderr.String(), "private") {
		t.Fatal("credential leaked")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != edited {
		t.Fatal("conflict changed configuration")
	}
}

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

func TestInstallChecksExistingConfigBeforeReadingOneTimePayload(t *testing.T) {
	t.Setenv("CODEX_HOME", "")
	root := t.TempDir()
	path := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("model = ["), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := run([]string{"install", "--agent", "codex", "--stdin", "--home", root, "--skip-client-check"}, strings.NewReader(`{"api_key":"private-test-key"}`), &out, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "尚未兑换配置码") || !strings.Contains(stderr.String(), "configuration syntax") {
		t.Fatalf("missing early config error: %s", stderr.String())
	}
	if strings.Contains(stderr.String(), "private-test-key") {
		t.Fatal("credential leaked")
	}
}
