package native

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGatewayJSONPreservesLargeNumbersAndRejectsTrailingValues(t *testing.T) {
	for _, suffix := range []string{"", " {}"} {
		t.Run(fmt.Sprintf("suffix=%q", suffix), func(t *testing.T) {
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"data":{"agent":"codex","codex_manifest":{"large":9007199254740993}}}`+suffix)
			}))
			defer s.Close()
			p, err := (Network{Client: s.Client()}).Exchange(s.URL, "ticket", "codex")
			if suffix != "" {
				if err == nil {
					t.Fatal("accepted trailing JSON")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := p.CodexManifest["large"]; got != json.Number("9007199254740993") {
				t.Fatalf("lost precision: %#v", got)
			}
		})
	}
}

func TestNetworkExchangeAndProbe(t *testing.T) {
	var origin string
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "lianjieai-quick-import/1.0" {
			t.Error("missing user agent")
		}
		if r.URL.Path == "/api/v1/quick-import/exchange" {
			if r.Method != "POST" {
				t.Error("method")
			}
			fmt.Fprintf(w, `{"data":{"version":1,"agent":"claude","base_url":%q,"probe_url":%q,"api_key":"secret","model":"chosen"}}`, origin, origin+"/v1/models")
			return
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing authorization")
		}
		fmt.Fprint(w, `{"data":[{"id":"chosen","display_name":"Chosen"},{"id":"chosen"},{"id":"bad\n"}]}`)
	}))
	defer s.Close()
	origin = s.URL
	n := Network{Client: s.Client(), Version: func(string) string { return "2.1.242" }}
	p, err := n.Exchange(s.URL, "ticket", "claude")
	if err != nil {
		t.Fatal(err)
	}
	if err = n.SynchronizeModels(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.Models) != 1 || p.Models[0].Name != "Chosen" || !p.ClaudeModelPickerSupported {
		t.Fatalf("unexpected payload: %#v", p.Models)
	}
}

func TestNetworkRejectsUnsafeURLAndRedirect(t *testing.T) {
	calls := 0
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/secret?key=secret", http.StatusFound)
	}))
	defer s.Close()
	n := Network{Client: s.Client()}
	for _, u := range []string{"http://example.com", "https://user:secret@example.com", "https://example.com?key=secret"} {
		if _, err := n.Exchange(u, "secret", "claude"); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("unsafe exchange URL accepted or leaked")
		}
	}
	_, err := n.Exchange(s.URL, "secret", "claude")
	if err == nil || strings.Contains(err.Error(), "secret") || calls != 1 {
		t.Fatalf("redirect handling: %v, calls %d", err, calls)
	}
	p := Payload{BaseURL: s.URL, ProbeURL: "https://example.com/models", APIKey: "secret"}
	if err = n.SynchronizeModels(&p); err == nil || calls != 1 {
		t.Fatal("cross-origin probe accepted")
	}
}

func TestCodexCatalogCompatibility(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery == "" {
			fmt.Fprint(w, `{"data":[{"id":"chosen"}]}`)
			return
		}
		if r.URL.Query().Get("client_version") != "0.155.3" {
			t.Error("actual version missing")
		}
		fmt.Fprint(w, `{"models":[{"slug":"chosen","model_messages":{"instructions_template":"hello"},"supports_reasoning_summary_parameter":true}]}`)
	}))
	defer s.Close()
	n := Network{Client: s.Client(), Version: func(string) string { return "0.155.3" }}
	p := Payload{Agent: "codex", BaseURL: s.URL, ProbeURL: s.URL + "/models", Model: "chosen"}
	if err := n.SynchronizeModels(&p); err != nil {
		t.Fatal(err)
	}
	m := p.CodexManifest["models"].([]any)[0].(map[string]any)
	if m["base_instructions"] != "hello" || m["supports_reasoning_summaries"] != true {
		t.Fatal("aliases missing")
	}
}

func TestNetworkBoundsAndSanitizesResponses(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"HTTP failure", "super-secret", 401},
		{"invalid JSON", `{"secret":"super-secret"`, 200},
		{"oversized", strings.Repeat("a", 1024*1024+1), 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(test.status); fmt.Fprint(w, test.body) }))
			defer s.Close()
			_, err := (Network{Client: s.Client()}).Exchange(s.URL, "super-secret", "claude")
			if err == nil || strings.Contains(err.Error(), "super-secret") {
				t.Fatalf("unsafe error: %v", err)
			}
		})
	}
}

func TestClaudeVersionGate(t *testing.T) {
	for _, test := range []struct {
		version string
		want    bool
	}{{"", false}, {"2.1.241", false}, {"2.1.242", true}, {"2.2.0", true}, {"3.0.0", true}, {"1.99.999", false}} {
		if got := atLeast(test.version, [3]int{2, 1, 242}); got != test.want {
			t.Errorf("version %s: %v", test.version, got)
		}
	}
}

func TestPreflightRejectsOverridesEvenForStdin(t *testing.T) {
	t.Setenv("CODEX_HOME", "isolated-but-custom")
	if Preflight("codex", false) == nil {
		t.Fatal("override accepted")
	}
}

func TestClientVersionWindowsCommandShim(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows command shim")
	}
	dir := filepath.Join(t.TempDir(), "space directory")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "codex.cmd"), []byte("@echo off\r\necho codex-cli 0.155.3\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got := ClientVersion("codex"); got != "0.155.3" {
		t.Fatalf("shim version: %q", got)
	}
}
