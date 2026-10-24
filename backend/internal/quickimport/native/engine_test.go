package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyPythonJournal(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, ".sub2api-quick-import", "claude")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, paths["claude"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	// Literal Python journal schema, including explicit false values and absence wrappers.
	legacy := `[{"agent":"claude","existed":true,"before_text":"{\"env\":{\"existing\":\"keep\"}}","after_text":"{\"env\":{\"existing\":\"keep\",\"ANTHROPIC_AUTH_TOKEN\":\"test-secret\"},\"modelPicker\":false}","changes":[{"path":["env","ANTHROPIC_AUTH_TOKEN"],"value":{"exists":true,"value":"test-secret"}},{"path":["modelPicker"],"value":{"exists":true,"value":false}}],"inverse":[{"path":["env","ANTHROPIC_AUTH_TOKEN"],"value":{"exists":false}},{"path":["modelPicker"],"value":{"exists":false}}],"pending":false,"owned_files":[]}]`
	if err := os.WriteFile(filepath.Join(folder, "journal.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"env":{"existing":"keep","ANTHROPIC_AUTH_TOKEN":"test-secret"},"modelPicker":false,"later":42}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	data, err := load(string(b), "claude")
	if err != nil {
		t.Fatal(err)
	}
	if !equal(data, map[string]any{"env": map[string]any{"existing": "keep"}, "later": 42}) {
		t.Fatalf("unexpected cleanup: %s", b)
	}
}

func TestCleanLegacyClaudeImageMCPRecord(t *testing.T) {
	root := t.TempDir()
	before := `{"env":{"existing":"keep"}}`
	server := map[string]any{"type": "http", "url": "https://example.test/mcp", "headers": map[string]any{"Authorization": "Bearer test-secret"}}
	added := change{Path: []string{"mcpServers", "lianjieai-image"}, Value: value{Exists: true, Value: server}}
	after, err := render(before, "claude", []change{added})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, paths["claude"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(after), 0600); err != nil {
		t.Fatal(err)
	}
	userConfig := filepath.Join(root, ".claude.json")
	userText, err := render(`{"other":"keep"}`, "claude", []change{added})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userConfig, []byte(userText), 0600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, ".sub2api-quick-import", "claude")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	rec := record{Agent: "claude", Existed: true, Before: before, After: after, Changes: []change{added}, Inverse: []change{{Path: added.Path, Value: value{Exists: false}}}}
	if err := writeJournal(folder, []record{rec}); err != nil {
		t.Fatal(err)
	}
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	settings, err := os.ReadFile(path)
	if err != nil || string(settings) != before {
		t.Fatalf("legacy settings were not restored: %v %s", err, settings)
	}
	userAfter, err := os.ReadFile(userConfig)
	if err != nil {
		t.Fatal(err)
	}
	userData, err := load(string(userAfter), "claude")
	if err != nil || get(userData, []string{"mcpServers", "lianjieai-image"}).Exists || get(userData, []string{"other"}).Value != "keep" {
		t.Fatalf("legacy user MCP cleanup lost unrelated settings: %v %s", err, userAfter)
	}
}

func TestNewImportsUseLianjieAIProviderID(t *testing.T) {
	for _, agent := range []string{"codex", "opencode"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			payload := Payload{Version: 1, Agent: agent, APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model"}
			if err := Install(root, payload); err != nil {
				t.Fatal(err)
			}
			text, _, err := readText(filepath.Join(root, paths[agent]))
			if err != nil {
				t.Fatal(err)
			}
			data, err := load(text, agent)
			if err != nil {
				t.Fatal(err)
			}
			if agent == "codex" {
				if get(data, []string{"model_provider"}).Value != "lianjieai_quick" || !get(data, []string{"model_providers", "lianjieai_quick"}).Exists {
					t.Fatal("Codex import did not use the connection AI provider ID")
				}
			} else if get(data, []string{"model"}).Value != "lianjieai_quick/test-model" || !get(data, []string{"provider", "lianjieai_quick"}).Exists {
				t.Fatal("OpenCode import did not use the connection AI provider ID")
			}
			if err := Clean(root, agent); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCleanLegacyCodexProviderPreservesLaterSettings(t *testing.T) {
	root := t.TempDir()
	before := "theme = \"old\"\n"
	changes := []change{
		{Path: []string{"model"}, Value: value{Exists: true, Value: "test-model"}},
		{Path: []string{"model_provider"}, Value: value{Exists: true, Value: "sub2api_quick"}},
		{Path: []string{"model_providers", "sub2api_quick"}, Value: value{Exists: true, Value: map[string]any{"name": "legacy", "base_url": "https://example.test/v1", "wire_api": "responses", "experimental_bearer_token": "test-secret"}}},
	}
	after, err := render(before, "codex", changes)
	if err != nil {
		t.Fatal(err)
	}
	original, err := load(before, "codex")
	if err != nil {
		t.Fatal(err)
	}
	rec := record{Agent: "codex", Existed: true, Before: before, After: after, Changes: changes, Owned: []ownedFile{}}
	for _, c := range changes {
		rec.Inverse = append(rec.Inverse, change{Path: c.Path, Value: get(original, c.Path)})
	}
	path := filepath.Join(root, paths["codex"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("later = \"keep\"\n"+after), 0600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, ".sub2api-quick-import", "codex")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeJournal(folder, []record{rec}); err != nil {
		t.Fatal(err)
	}
	if err := Clean(root, "codex"); err != nil {
		t.Fatal(err)
	}
	text, _, err := readText(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(text, "codex")
	if err != nil || get(data, []string{"later"}).Value != "keep" || get(data, []string{"model_providers", "sub2api_quick"}).Exists {
		t.Fatalf("legacy cleanup lost newer settings: %v %s", err, text)
	}
}

func TestCleanLegacyOpenCodeProviderPreservesLaterSettings(t *testing.T) {
	root := t.TempDir()
	before := `{"theme":"old"}`
	changes := []change{
		{Path: []string{"model"}, Value: value{Exists: true, Value: "sub2api_quick/test-model"}},
		{Path: []string{"provider", "sub2api_quick"}, Value: value{Exists: true, Value: map[string]any{"name": "legacy", "options": map[string]any{"apiKey": "test-secret"}}}},
	}
	after, err := render(before, "opencode", changes)
	if err != nil {
		t.Fatal(err)
	}
	original, err := load(before, "opencode")
	if err != nil {
		t.Fatal(err)
	}
	rec := record{Agent: "opencode", Existed: true, Before: before, After: after, Changes: changes, Owned: []ownedFile{}}
	for _, c := range changes {
		rec.Inverse = append(rec.Inverse, change{Path: c.Path, Value: get(original, c.Path)})
	}
	current, err := render(after, "opencode", []change{{Path: []string{"later"}, Value: value{Exists: true, Value: "keep"}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, paths["opencode"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(current), 0600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, ".sub2api-quick-import", "opencode")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeJournal(folder, []record{rec}); err != nil {
		t.Fatal(err)
	}
	if err := Clean(root, "opencode"); err != nil {
		t.Fatal(err)
	}
	text, _, err := readText(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(text, "opencode")
	if err != nil || get(data, []string{"later"}).Value != "keep" || get(data, []string{"provider", "sub2api_quick"}).Exists {
		t.Fatalf("legacy cleanup lost newer settings: %v %s", err, text)
	}
}

func TestPendingAndLockRefuseChanges(t *testing.T) {
	for _, kind := range []string{"pending", "lock"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "test"}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			folder := filepath.Join(root, ".sub2api-quick-import", "claude")
			path := filepath.Join(root, paths["claude"])
			before, _ := os.ReadFile(path)
			if kind == "lock" {
				os.WriteFile(filepath.Join(folder, "lock"), nil, 0600)
			} else {
				r, e := readJournal(folder)
				if e != nil {
					t.Fatal(e)
				}
				r[0].Pending = true
				if e = writeJournal(folder, r); e != nil {
					t.Fatal(e)
				}
			}
			if err := Install(root, p); err == nil {
				t.Fatal("install should refuse")
			}
			if err := Clean(root, "claude"); err == nil {
				t.Fatal("cleanup should refuse")
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("refused operation changed config")
			}
		})
	}
}

func TestInterruptedCleanupResumes(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-write", true: "after-write"}[applied], func(t *testing.T) {
			root := t.TempDir()
			p := Payload{Version: 1, Agent: "codex", APIKey: "test-secret", BaseURL: "https://example.test", Model: "test", CodexManifest: map[string]any{"models": []any{map[string]any{"slug": "test"}}}}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			folder := filepath.Join(root, ".sub2api-quick-import", "codex")
			path := filepath.Join(root, paths["codex"])
			r, err := readJournal(folder)
			if err != nil {
				t.Fatal(err)
			}
			r[0].CleanupPending = true
			r[0].CleanupBefore = r[0].After
			r[0].CleanupResult = ""
			r[0].CleanupDelete = true
			if err = writeJournal(folder, r); err != nil {
				t.Fatal(err)
			}
			if applied {
				os.Remove(path)
				aux, err := ownedPath(root, folder, r[0].Owned[0])
				if err != nil {
					t.Fatal(err)
				}
				os.Remove(aux)
			}
			if err = Clean(root, "codex"); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("cleanup did not remove config")
			}
			r, err = readJournal(folder)
			if err != nil || len(r) != 0 {
				t.Fatal("cleanup did not finish journal")
			}
		})
	}
}

func TestCatalogConflict(t *testing.T) {
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "codex", APIKey: "test-secret", BaseURL: "https://example.test", Model: "test", CodexManifest: map[string]any{"models": []any{map[string]any{"slug": "test"}}}}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, ".sub2api-quick-import", "codex")
	r, err := readJournal(folder)
	if err != nil {
		t.Fatal(err)
	}
	aux, err := ownedPath(root, folder, r[0].Owned[0])
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(aux, []byte("later edit"), 0600)
	if err = Clean(root, "codex"); err == nil {
		t.Fatal("catalog conflict accepted")
	}
	b, _ := os.ReadFile(aux)
	if string(b) != "later edit" {
		t.Fatal("catalog edit lost")
	}
}

func TestJSONCAndComplexTOML(t *testing.T) {
	for _, tc := range []struct {
		agent, text string
		safe        bool
	}{{"opencode", `{"$schema":"https://example.test/schema"}`, true}, {"opencode", "", false}, {"opencode", `{"model":"existing"}`, false}, {"opencode", "// comments\n{}", false}, {"codex", "\"model\" = \"existing\"\n", false}, {"codex", "model = \"existing\"\n# keep comment\n[other]\nkey = \"untouched\"\n", true}} {
		t.Run(tc.agent+tc.text, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, paths[tc.agent])
			if tc.agent == "opencode" {
				path = strings.TrimSuffix(path, ".json") + ".jsonc"
			}
			os.MkdirAll(filepath.Dir(path), 0700)
			os.WriteFile(path, []byte(tc.text), 0600)
			p := Payload{Version: 1, Agent: tc.agent, APIKey: "test-secret", BaseURL: "https://example.test", Model: "test"}
			err := Install(root, p)
			if (err == nil) != tc.safe {
				t.Fatalf("safe=%v err=%v", tc.safe, err)
			}
			if err == nil {
				if err = Clean(root, tc.agent); err != nil {
					t.Fatal(err)
				}
			}
			b, _ := os.ReadFile(path)
			if string(b) != tc.text {
				t.Fatal("original text not preserved")
			}
		})
	}
}

func TestLinkedConfigAndRecoveryRefused(t *testing.T) {
	for _, directory := range []string{".claude", ".sub2api-quick-import"} {
		t.Run(directory, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(root, directory)); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "test"}
			if err := Install(root, p); err == nil {
				t.Fatal("linked path accepted")
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatal("linked target modified")
			}
		})
	}
}

func TestOwnedPathTraversal(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, ".sub2api-quick-import", "codex")
	for _, path := range []string{"../models-00000000000000000000000000000000.json", ".claude/settings.json", ".sub2api-quick-import/codex/journal.json"} {
		if _, err := ownedPath(root, folder, ownedFile{Path: path}); err == nil {
			t.Fatal("accepted", path)
		}
	}
}

func TestLegacyExistenceWrapperFalse(t *testing.T) {
	var v value
	if err := json.Unmarshal([]byte(`{"exists":true,"value":false}`), &v); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(v)
	if string(b) != `{"exists":true,"value":false}` {
		t.Fatal(string(b))
	}
}

func TestJournalNullAndEmptyCleanupCompatibility(t *testing.T) {
	b, err := json.Marshal(value{Exists: true, Value: nil})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"exists":true,"value":null}` {
		t.Fatalf("legacy null lost: %s", b)
	}
	b, err = json.Marshal(record{CleanupPending: true, CleanupResult: "", CleanupDelete: false})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"cleanup_before_text", "cleanup_result_text", "cleanup_delete"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("missing legacy cleanup key %s", key)
		}
	}
}

func TestLargeJSONNumbersPreserved(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, paths["claude"])
	os.MkdirAll(filepath.Dir(path), 0700)
	before := `{"unrelated":9007199254740993,"modelPicker":{"options":[],"large":9007199254740995}}`
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "test", ClaudeModelPickerSupported: true}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "9007199254740993") {
		t.Fatalf("unrelated number changed: %s", b)
	}
	edited := strings.Replace(string(b), "{", `{"later":9007199254740997,`, 1)
	os.WriteFile(path, []byte(edited), 0600)
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	for _, n := range []string{"9007199254740993", "9007199254740995", "9007199254740997"} {
		if !strings.Contains(string(b), n) {
			t.Fatalf("number %s changed: %s", n, b)
		}
	}
}

func TestClaudeImportPinsEveryModelRoute(t *testing.T) {
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "claude-test"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, paths["claude"]))
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(string(content), "claude")
	if err != nil {
		t.Fatal(err)
	}
	env, ok := data["env"].(map[string]any)
	if !ok {
		t.Fatalf("missing Claude environment: %s", content)
	}
	for _, name := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL", "ANTHROPIC_DEFAULT_FABLE_MODEL", "CLAUDE_CODE_SUBAGENT_MODEL", "ANTHROPIC_CUSTOM_MODEL_OPTION"} {
		if env[name] != "claude-test" {
			t.Errorf("%s must use the selected model, got %v", name, env[name])
		}
	}
	if _, injected := env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"]; injected {
		t.Fatal("quick import must not disable Claude Code updates and optional features")
	}
}

func TestClaudeReimportRemovesPreviouslyManagedTrafficFlag(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, paths["claude"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	oldConfig := `{"env":{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":"1","USER_SETTING":"keep"}}`
	if err := os.WriteFile(path, []byte(oldConfig), 0600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, ".sub2api-quick-import", "claude")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	flag := []string{"env", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"}
	oldRecord := record{Agent: "claude", Existed: true, Before: `{"env":{"USER_SETTING":"keep"}}`, After: oldConfig,
		Changes: []change{{Path: flag, Value: value{Exists: true, Value: "1"}}},
		Inverse: []change{{Path: flag, Value: value{Exists: false}}}}
	if err := writeJournal(folder, []record{oldRecord}); err != nil {
		t.Fatal(err)
	}
	p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "claude-test"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(string(content), "claude")
	if err != nil {
		t.Fatal(err)
	}
	env := data["env"].(map[string]any)
	if _, stillPresent := env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"]; stillPresent || env["USER_SETTING"] != "keep" {
		t.Fatalf("reimport did not remove the managed flag while preserving user settings: %s", content)
	}
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	content, _ = os.ReadFile(path)
	if string(content) != oldConfig {
		t.Fatalf("cleanup did not restore the previous configuration: %s", content)
	}
}

func TestClaudeReimportPreservesEditedTrafficFlag(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, paths["claude"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"env":{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":"user-choice"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, ".sub2api-quick-import", "claude")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	flag := []string{"env", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"}
	if err := writeJournal(folder, []record{{Agent: "claude", Changes: []change{{Path: flag, Value: value{Exists: true, Value: "1"}}}}}); err != nil {
		t.Fatal(err)
	}
	p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "claude-test"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(string(content), "claude")
	if err != nil {
		t.Fatal(err)
	}
	if got := data["env"].(map[string]any)["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"]; got != "user-choice" {
		t.Fatalf("reimport changed the user's value: %v", got)
	}
}

func TestImportCatalogOmitsDedicatedMediaModels(t *testing.T) {
	for _, agent := range []string{"claude", "opencode"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			p := Payload{Version: 1, Agent: agent, APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "deepseek-chat", ClaudeModelPickerSupported: true, Models: []Model{{ID: "deepseek-chat"}, {ID: "gpt-image-1"}}}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(filepath.Join(root, paths[agent]))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(content), "gpt-image-1") {
				t.Fatalf("media model was offered as a coding model: %s", content)
			}
		})
	}
}

func TestStrictJSONTrailingValue(t *testing.T) {
	if _, err := load(`{} {}`, "claude"); err == nil {
		t.Fatal("accepted trailing config JSON")
	}
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "journal.json"), []byte(`[] {}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readJournal(folder); err == nil {
		t.Fatal("accepted trailing journal JSON")
	}
}

func TestTOMLDollarValuesRemainLiteral(t *testing.T) {
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "codex", APIKey: "test-$1-${secret}-$$", BaseURL: "https://example.test", Model: "test-$model"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, paths["codex"]))
	data, err := load(string(b), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if get(data, []string{"model"}).Value != p.Model || get(data, []string{"model_providers", provider, "experimental_bearer_token"}).Value != p.APIKey {
		t.Fatal("dollar values changed")
	}
}

func TestTOMLNonfiniteCannotBypassPreservation(t *testing.T) {
	for _, number := range []string{"nan", "inf", "-inf"} {
		t.Run(number, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, paths["codex"])
			os.MkdirAll(filepath.Dir(path), 0700)
			before := "threshold = " + number + "\n[model_providers.sub2api_quick]\nname = \"old\"\n  [unrelated]\nkeep = true\n"
			os.WriteFile(path, []byte(before), 0600)
			p := Payload{Version: 1, Agent: "codex", APIKey: "test-secret", BaseURL: "https://example.test", Model: "test"}
			if err := Install(root, p); err == nil {
				t.Fatal("nonfinite complex layout must be refused")
			}
			b, _ := os.ReadFile(path)
			if string(b) != before {
				t.Fatalf("refusal modified original: %s", b)
			}
		})
	}
}

func TestInstallCleanPreservesUnrelated(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "opencode"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			p := Payload{Version: 1, Agent: agent, APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model"}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, paths[agent])
			before, _ := os.ReadFile(path)
			var edited string
			if agent == "codex" {
				edited = "unrelated = true\n" + string(before)
			} else {
				edited = strings.Replace(string(before), "{", "{\"unrelated\": true,", 1)
			}
			if err := os.WriteFile(path, []byte(edited), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Clean(root, agent); err != nil {
				t.Fatal(err)
			}
			after, _ := os.ReadFile(path)
			if !strings.Contains(string(after), "unrelated") || strings.Contains(string(after), "test-secret") {
				t.Fatalf("bad cleanup: %s", after)
			}
		})
	}
}

func TestCheckLocalConfigRejectsUnsupportedFilesBeforeTicketExchange(t *testing.T) {
	for _, tc := range []struct {
		name, agent, path, content, want string
	}{
		{"invalid-codex-toml", "codex", ".codex/config.toml", "model = [", "configuration syntax"},
		{"opencode-jsonc-overlay", "opencode", ".config/opencode/opencode.jsonc", `{ "provider": { "existing": {} } }`, "manual configuration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			if err := CheckLocalConfig(root, tc.agent); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestImportNewAgentsPreservesExistingConfigurationAndRestores(t *testing.T) {
	for _, tc := range []struct{ agent, before, expected string }{
		{"workbuddy", `[{"id":"existing-model","url":"https://other.test/v1/chat/completions"}]`, `"url": "https://example.test/v1/chat/completions"`},
		{"pi", `{"providers":{"other":{"baseUrl":"https://other.test/v1"}}}`, `"api": "openai-completions"`},
		{"deepseek-harness", "- id: existing-plugin\n  config:\n    value: true\n", "llm-pi-ai"},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, paths[tc.agent])
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.before), 0600); err != nil {
				t.Fatal(err)
			}
			p := Payload{Version: 1, Agent: tc.agent, APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model", Models: []Model{{ID: "test-model", Name: "Test Model"}}}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(after), tc.expected) || !strings.Contains(string(after), "test-secret") {
				t.Fatalf("bad %s configuration: %s %v", tc.agent, after, err)
			}
			if err := Clean(root, tc.agent); err != nil {
				t.Fatal(err)
			}
			restored, err := os.ReadFile(path)
			if err != nil || tc.agent == "deepseek-harness" && !strings.Contains(string(restored), "existing-plugin") || tc.agent == "workbuddy" && string(restored) != tc.before {
				t.Fatalf("bad cleanup: %s %v", restored, err)
			}
			if strings.Contains(string(restored), "test-secret") {
				t.Fatalf("credential left after cleanup: %s", restored)
			}
		})
	}
}

func TestWorkBuddyFreshImportWritesDesktopArrayAndCleanRemovesIt(t *testing.T) {
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "workbuddy", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model", Models: []Model{{ID: "test-model", Name: "Test Model"}}}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".workbuddy", "models.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var models []map[string]any
	if err := json.Unmarshal(content, &models); err != nil || len(models) != 1 || models[0]["id"] != "test-model" {
		t.Fatalf("WorkBuddy desktop requires a model array: %s %v", content, err)
	}
	if models[0]["vendor"] != "Custom" || models[0]["useCustomProtocol"] != true || models[0]["url"] != "https://example.test/v1/chat/completions" {
		t.Fatalf("WorkBuddy must send the exact gateway endpoint without appending a second path: %s", content)
	}
	if models[0]["name"] != "连接AI · Test Model" {
		t.Fatalf("WorkBuddy model picker must identify the imported source: %s", content)
	}
	if _, set := models[0]["maxInputTokens"]; set {
		t.Fatalf("WorkBuddy import must not invent the selected model's input limit: %s", content)
	}
	if _, set := models[0]["maxOutputTokens"]; set {
		t.Fatalf("WorkBuddy import must not invent the selected model's output limit: %s", content)
	}
	if err := Clean(root, "workbuddy"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup left a new WorkBuddy config: %v", err)
	}
}

func TestDeepSeekHarnessMergesExistingPiAIProvider(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, paths["deepseek-harness"])
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	before := "- id: llm-pi-ai\n  config:\n    providers:\n      other:\n        api: openai-completions\n        baseURL: https://other.test/v1\n        models:\n          - id: other-model\n"
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	p := Payload{Version: 1, Agent: "deepseek-harness", Platform: "deepseek", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model", Models: []Model{{ID: "test-model", Name: "Test Model"}, {ID: "image-only", Name: "Image"}}}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "other-model") || !strings.Contains(string(content), "test-model") || strings.Contains(string(content), "image-only") || !strings.Contains(string(content), "test-secret") || !strings.Contains(string(content), "supportsDeveloperRole: false") || !strings.Contains(string(content), "maxTokensField: max_tokens") {
		t.Fatalf("existing provider was not preserved: %s %v", content, err)
	}
	if err := Clean(root, "deepseek-harness"); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil || string(content) != before {
		t.Fatalf("cleanup did not restore original profile: %s %v", content, err)
	}
}

func TestDeepSeekHarnessDoesNotForceDeepSeekCompatOnOpenAI(t *testing.T) {
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "deepseek-harness", Platform: "openai", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "o-series-model"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, paths["deepseek-harness"]))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "maxTokensField: max_tokens") || strings.Contains(string(content), "supportsDeveloperRole: false") {
		t.Fatalf("OpenAI group must keep its own request defaults: %s", content)
	}
}

func TestDeepSeekHarnessUsesInstalledDesktopProfileAndRestoresIt(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, ".dsh", "profiles", "desktop")
	if err := os.MkdirAll(desktop, 0700); err != nil {
		t.Fatal(err)
	}
	p := Payload{Version: 1, Agent: "deepseek-harness", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(desktop, "cordis.patch.yml")
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "test-model") {
		t.Fatalf("desktop profile missing model: %v %s", err, content)
	}
	if _, err := os.Stat(filepath.Join(root, paths["deepseek-harness"])); !os.IsNotExist(err) {
		t.Fatalf("wrote web profile: %v", err)
	}
	if err := Clean(root, "deepseek-harness"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup left desktop patch: %v", err)
	}
}

func TestDeepSeekHarnessBothProfilesRequireExplicitSelection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"web", "desktop"} {
		if err := os.MkdirAll(filepath.Join(root, ".dsh", "profiles", name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckLocalConfig(root, "deepseek-harness"); err == nil {
		t.Fatal("ambiguous profile accepted")
	}
	p := Payload{Version: 1, Agent: "deepseek-harness", Profile: "desktop", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	if err := Clean(root, "deepseek-harness"); err != nil {
		t.Fatal(err)
	}
}

func TestImportDoesNotInjectImageMCPByDefault(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "opencode"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			p := Payload{Version: 1, Agent: agent, APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model"}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(filepath.Join(root, paths[agent]))
			if err != nil || strings.Contains(string(content), "lianjieai_image") || strings.Contains(string(content), "lianjieai-image") {
				t.Fatalf("unexpected MCP: %s %v", content, err)
			}
		})
	}
}

func TestImportNeverInjectsImageMCPEvenWithLegacyOptIn(t *testing.T) {
	enabled := true
	for _, agent := range []string{"claude", "codex", "deepseek-harness", "workbuddy", "opencode", "pi"} {
		t.Run(agent, func(t *testing.T) {
			root := t.TempDir()
			p := Payload{Version: 1, Agent: agent, APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model", MCPEndpoint: "https://example.test/mcp", MCPImageToolsEnabled: &enabled}
			if agent == "deepseek-harness" {
				p.Profile = "web"
				path := filepath.Join(root, paths[agent])
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("[]\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := Install(root, p); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(filepath.Join(root, paths[agent]))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(content), "/mcp") || strings.Contains(string(content), "lianjieai_image") || strings.Contains(string(content), "lianjieai-image") {
				t.Fatalf("one-click import injected image MCP: %s", content)
			}
			if _, err := os.Stat(filepath.Join(root, ".claude.json")); agent == "claude" && !os.IsNotExist(err) {
				t.Fatalf("Claude user MCP file was created: %v", err)
			}
		})
	}
}

func TestImportPreservesExistingOpenCodeMCPWithoutAddingOne(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, paths["opencode"])
	legacy := `{"mcp":{"sub2api_image":{"type":"remote","url":"https://legacy.example/mcp","headers":{"Authorization":"Bearer old-secret"}}}}`
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	enabled := true
	p := Payload{Version: 1, Agent: "opencode", APIKey: "new-secret", BaseURL: "https://example.test/v1", Model: "test-model", MCPImageToolsEnabled: &enabled}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := load(string(text), "opencode")
	if err != nil {
		t.Fatal(err)
	}
	if get(data, []string{"mcp", "servers", "lianjieai_image"}).Exists {
		t.Fatalf("one-click import added image MCP: %s", text)
	}
	if !get(data, []string{"mcp", "sub2api_image"}).Exists {
		t.Fatalf("one-click import changed an existing MCP: %s", text)
	}
}

func TestInstallCanDisableImageMCP(t *testing.T) {
	disabled := false
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "opencode", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "test-model", MCPImageToolsEnabled: &disabled}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(filepath.Join(root, paths["opencode"]))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), "sub2api_image") || strings.Contains(string(text), "/mcp") {
		t.Fatalf("disabled MCP was written: %s", text)
	}
}

func TestCleanConflictAndStack(t *testing.T) {
	root := t.TempDir()
	p := Payload{Version: 1, Agent: "claude", APIKey: "test-secret", BaseURL: "https://example.test", Model: "first"}
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	p.Model = "second"
	if err := Install(root, p); err != nil {
		t.Fatal(err)
	}
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, paths["claude"])
	text, _ := os.ReadFile(path)
	edited := strings.ReplaceAll(string(text), "first", "user-edited")
	os.WriteFile(path, []byte(edited), 0600)
	if err := Clean(root, "claude"); err == nil {
		t.Fatal("expected conflict")
	}
	actual, _ := os.ReadFile(path)
	if string(actual) != edited {
		t.Fatal("conflict modified file")
	}
	os.WriteFile(path, text, 0600)
	if err := Clean(root, "claude"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("new config was not removed")
	}
}

func TestCodexCleanupAfterModelSwitch(t *testing.T) {
	for _, original := range []string{"", "model = \"original\"\nmodel_provider = \"original-provider\"\n"} {
		t.Run(original, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, paths["codex"])
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if original != "" {
				if err := os.WriteFile(path, []byte(original), 0600); err != nil {
					t.Fatal(err)
				}
			}
			payload := Payload{Version: 1, Agent: "codex", APIKey: "test-secret", BaseURL: "https://example.test/v1", Model: "imported", CodexManifest: map[string]any{"models": []any{map[string]any{"slug": "imported"}}}}
			if err := Install(root, payload); err != nil {
				t.Fatal(err)
			}
			text, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			edited := "unrelated = true\n" + strings.Replace(string(text), `model = "imported"`, `model = "selected-later"`, 1)
			if err := os.WriteFile(path, []byte(edited), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Clean(root, "codex"); err != nil {
				t.Fatal(err)
			}
			text, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data, err := load(string(text), "codex")
			if err != nil {
				t.Fatal(err)
			}
			if data["unrelated"] != true || strings.Contains(string(text), "test-secret") || strings.Contains(string(text), "selected-later") {
				t.Fatal("cleanup did not restore managed fields and preserve unrelated settings")
			}
			if original != "" && (data["model"] != "original" || data["model_provider"] != "original-provider") {
				t.Fatal("original model/provider not restored")
			}
			catalogs, err := filepath.Glob(filepath.Join(root, ".sub2api-quick-import", "codex", "models-*.json"))
			if err != nil || len(catalogs) != 0 {
				t.Fatal("catalog was not removed")
			}
		})
	}
}
