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
