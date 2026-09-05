package native

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Network performs bounded, credential-safe gateway requests. Client and Version
// are injectable to allow local TLS fixtures without weakening production TLS.
type Network struct {
	Client  *http.Client
	Version func(string) string
}

func secureURL(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, errors.New("HTTPS URL without credentials, query or fragment required")
	}
	return u, nil
}

func (n Network) request(method, target, key string, body []byte, limit int64, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return errors.New("invalid request")
	}
	req.Header.Set("User-Agent", "lianjieai-quick-import/1.0")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := http.Client{Timeout: 30 * time.Second}
	if n.Client != nil {
		client = *n.Client
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return errors.New("network or TLS connection error")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d; generate a new import command and retry", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return errors.New("could not read gateway response")
	}
	if int64(len(data)) > limit {
		return errors.New("gateway response exceeded size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(result) != nil {
		return errors.New("gateway returned invalid JSON")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("gateway returned invalid JSON")
	}
	return nil
}

func (n Network) Exchange(server, ticket, agent string) (Payload, error) {
	var result struct {
		Data *Payload `json:"data"`
	}
	if _, err := secureURL(server); err != nil {
		return Payload{}, err
	}
	body, _ := json.Marshal(map[string]string{"ticket": ticket, "agent": agent})
	err := n.request(http.MethodPost, strings.TrimRight(server, "/")+"/api/v1/quick-import/exchange", "", body, 1024*1024, &result)
	if err != nil {
		return Payload{}, err
	}
	if result.Data == nil {
		return Payload{}, errors.New("gateway did not return configuration")
	}
	return *result.Data, nil
}

func (n Network) SynchronizeModels(p *Payload) error {
	base, err := secureURL(p.BaseURL)
	if err != nil {
		return errors.New("invalid connectivity probe URL")
	}
	probe, err := secureURL(p.ProbeURL)
	if err != nil || !strings.EqualFold(base.Host, probe.Host) {
		return errors.New("invalid connectivity probe URL")
	}
	var result struct {
		Data json.RawMessage `json:"data"`
	}
	if err = n.request(http.MethodGet, p.ProbeURL, p.APIKey, nil, 1024*1024, &result); err != nil {
		return err
	}
	var items []map[string]any
	if len(result.Data) == 0 || result.Data[0] != '[' || json.Unmarshal(result.Data, &items) != nil {
		return errors.New("gateway did not return a model list; check the API base URL")
	}
	models := []Model{}
	seen := map[string]bool{}
	for _, item := range items {
		id, _ := item["id"].(string)
		if strings.TrimSpace(id) == "" || len([]rune(id)) > 200 || strings.IndexFunc(id, func(r rune) bool { return r < 32 }) >= 0 || seen[id] {
			continue
		}
		seen[id] = true
		name, _ := item["name"].(string)
		if strings.TrimSpace(name) == "" {
			name, _ = item["display_name"].(string)
		}
		if strings.TrimSpace(name) == "" {
			name = id
		}
		models = append(models, Model{ID: id, Name: name})
	}
	p.Models = models
	versionFn := n.Version
	if versionFn == nil {
		versionFn = ClientVersion
	}
	if p.Agent == "claude" {
		p.ClaudeModelPickerSupported = atLeast(versionFn("claude"), [3]int{2, 1, 242})
	}
	if p.Agent == "codex" {
		version := versionFn("codex")
		if version == "" {
			version = "0.147.0"
		}
		var manifest map[string]any
		if err = n.request(http.MethodGet, p.ProbeURL+"?client_version="+url.QueryEscape(version), p.APIKey, nil, 8*1024*1024, &manifest); err != nil {
			return err
		}
		entries, ok := manifest["models"].([]any)
		if !ok {
			return errors.New("gateway did not return a Codex model catalog")
		}
		found := false
		for _, entry := range entries {
			m, ok := entry.(map[string]any)
			if !ok {
				return errors.New("invalid Codex model descriptor")
			}
			if m["slug"] == p.Model {
				found = true
			}
			messages, _ := m["model_messages"].(map[string]any)
			if _, exists := m["base_instructions"]; !exists {
				if v, ok := messages["instructions_template"].(string); ok {
					m["base_instructions"] = v
				}
			}
			if _, exists := m["supports_reasoning_summaries"]; !exists {
				if v, ok := m["supports_reasoning_summary_parameter"].(bool); ok {
					m["supports_reasoning_summaries"] = v
				}
			}
		}
		if !found {
			return errors.New("selected model is not in the Codex model catalog")
		}
		p.CodexManifest = manifest
	}
	return nil
}

var versionPattern = regexp.MustCompile(`\b([0-9]+)\.([0-9]+)\.([0-9]+)\b`)

func ClientVersion(agent string) string {
	if !ValidAgent(agent) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Bound captured output even when an installed client behaves unexpectedly.
	var output boundedOutput
	executable, err := exec.LookPath(agent)
	if err != nil {
		return ""
	}
	command := exec.CommandContext(ctx, executable, "--version")
	if runtime.GOOS == "windows" && (strings.EqualFold(filepath.Ext(executable), ".cmd") || strings.EqualFold(filepath.Ext(executable), ".bat")) {
		// npm installs Windows command shims. Keep shell expansion and control
		// characters out of the command; os/exec quotes paths containing spaces.
		if strings.ContainsAny(executable, "%!\"\r\n&|<>^()") {
			return ""
		}
		command = exec.CommandContext(ctx, "cmd.exe", "/d", "/c", executable, "--version")
	}
	command.Stdout = &output
	if command.Run() != nil {
		return ""
	}
	return versionPattern.FindString(output.String())
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	size := len(p)
	if b.Len() < 4096 {
		remaining := 4096 - b.Len()
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return size, nil
}
func atLeast(version string, min [3]int) bool {
	match := versionPattern.FindStringSubmatch(version)
	if len(match) != 4 {
		return false
	}
	for i := 0; i < 3; i++ {
		v, e := strconv.Atoi(match[i+1])
		if e != nil {
			return false
		}
		if v != min[i] {
			return v > min[i]
		}
	}
	return true
}

// Preflight preserves the supported clients' default user-config semantics.
func Preflight(agent string, checkClient bool) error {
	overrides := map[string][]string{"opencode": {"OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "XDG_CONFIG_HOME"}, "claude": {"CLAUDE_CONFIG_DIR", "ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"}, "codex": {"CODEX_HOME"}}
	names, ok := overrides[agent]
	if !ok {
		return errors.New("unsupported agent")
	}
	for _, name := range names {
		if os.Getenv(name) != "" {
			return errors.New("custom configuration environment detected; use manual setup or clear overrides first")
		}
	}
	if !checkClient {
		return nil
	}
	if _, err := exec.LookPath(agent); err == nil {
		return nil
	}
	if agent == "opencode" {
		home, _ := os.UserHomeDir()
		candidates := []string{"/Applications/OpenCode.app", filepath.Join(home, "Applications", "OpenCode.app")}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			candidates = append(candidates, filepath.Join(local, "Programs", "@opencode-aidesktop", "OpenCode.exe"), filepath.Join(local, "OpenCode", "OpenCode.exe"))
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return nil
			}
		}
	}
	instructions := map[string]string{"claude": "https://code.claude.com/docs/en/setup", "codex": "https://developers.openai.com/codex/cli", "opencode": "https://opencode.ai/download"}
	return fmt.Errorf("install %s first: %s; reopen the terminal and generate a new command", agent, instructions[agent])
}

// ValidAgent keeps untrusted agent names out of filesystem paths.
func ValidAgent(agent string) bool {
	return agent == "claude" || agent == "codex" || agent == "opencode"
}

// ValidateTransportPayload runs before a credential can be sent on the network.
func ValidateTransportPayload(p Payload, agent string) error {
	if !ValidAgent(agent) || p.Agent != agent {
		return errors.New("agent mismatch")
	}
	if p.Version != 1 || strings.TrimSpace(p.APIKey) == "" || strings.IndexFunc(p.APIKey, unicode.IsControl) >= 0 {
		return errors.New("invalid import configuration")
	}
	if _, err := secureURL(p.BaseURL); err != nil {
		return errors.New("invalid API base URL")
	}
	return nil
}
