// Package native implements offline, field-scoped quick-import recovery.
package native

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/quickimport"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const provider = "sub2api_quick"

var paths = map[string]string{"claude": ".claude/settings.json", "codex": ".codex/config.toml", "opencode": ".config/opencode/opencode.json", "workbuddy": ".workbuddy/models.json", "pi": ".pi/agent/models.json", "deepseek-harness": ".dsh/profiles/web/cordis.patch.yml"}

type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Payload struct {
	Version                    int            `json:"version"`
	Agent                      string         `json:"agent"`
	Platform                   string         `json:"platform"`
	APIKey                     string         `json:"api_key"`
	BaseURL                    string         `json:"base_url"`
	Model                      string         `json:"model"`
	Protocol                   string         `json:"protocol"`
	ProbeURL                   string         `json:"probe_url"`
	MCPEndpoint                string         `json:"mcp_endpoint"`
	MCPImageToolsEnabled       *bool          `json:"mcp_image_tools_enabled"`
	Models                     []Model        `json:"models"`
	ClaudeModelPickerSupported bool           `json:"claude_model_picker_supported"`
	CodexManifest              map[string]any `json:"codex_manifest"`
}
type value struct {
	Exists bool `json:"exists"`
	Value  any  `json:"value,omitempty"`
}

// Python's existence wrapper distinguishes an absent key from an explicit null.
func (v value) MarshalJSON() ([]byte, error) {
	if !v.Exists {
		return []byte(`{"exists":false}`), nil
	}
	return json.Marshal(struct {
		Exists bool `json:"exists"`
		Value  any  `json:"value"`
	}{true, v.Value})
}

type change struct {
	Path  []string `json:"path"`
	Value value    `json:"value"`
}
type ownedFile struct {
	Path string `json:"path"`
	Text string `json:"text"`
}
type record struct {
	Agent          string      `json:"agent"`
	Existed        bool        `json:"existed"`
	Before         string      `json:"before_text"`
	After          string      `json:"after_text"`
	Changes        []change    `json:"changes"`
	Inverse        []change    `json:"inverse"`
	Pending        bool        `json:"pending"`
	Owned          []ownedFile `json:"owned_files"`
	CleanupPending bool        `json:"cleanup_pending,omitempty"`
	CleanupBefore  string      `json:"cleanup_before_text,omitempty"`
	CleanupResult  string      `json:"cleanup_result_text,omitempty"`
	CleanupDelete  bool        `json:"cleanup_delete,omitempty"`
}

func (r record) MarshalJSON() ([]byte, error) {
	type plain record
	if !r.CleanupPending {
		return json.Marshal(plain(r))
	}
	// These keys must remain present even when empty/false, for restore.py.
	return json.Marshal(struct {
		plain
		Before string `json:"cleanup_before_text"`
		Result string `json:"cleanup_result_text"`
		Delete bool   `json:"cleanup_delete"`
	}{plain(r), r.CleanupBefore, r.CleanupResult, r.CleanupDelete})
}

func safePath(root, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("invalid configuration path")
	}
	for p := path; ; p = filepath.Dir(p) {
		if err := rejectLink(p); err != nil {
			return err
		}
		if p == root {
			break
		}
	}
	return nil
}
func target(root, agent string) (string, error) {
	p, ok := paths[agent]
	if !ok {
		return "", errors.New("unsupported Agent")
	}
	path := filepath.Join(root, p)
	return path, safePath(root, path)
}

// CheckLocalConfig catches unsupported local state before a one-time ticket is
// consumed. Install repeats the checks under its own lock before writing.
func CheckLocalConfig(root, agent string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	path, err := target(root, agent)
	if err != nil {
		return err
	}
	if agent == "opencode" {
		if err := checkOpenCodeJSONC(path); err != nil {
			return err
		}
	}
	text, _, err := readText(path)
	if err != nil {
		return err
	}
	if _, err := load(text, agent); err != nil {
		return err
	}
	folder := filepath.Join(root, ".sub2api-quick-import", agent)
	if err := safePath(root, folder); err != nil {
		return err
	}
	records, err := readJournal(folder)
	if err != nil {
		return err
	}
	return checkPending(records)
}

func checkOpenCodeJSONC(path string) error {
	overlay, exists, err := readText(strings.TrimSuffix(path, ".json") + ".jsonc")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	data, err := load(overlay, "opencode")
	if err != nil || strings.TrimSpace(overlay) == "" {
		return errors.New("existing OpenCode JSONC configuration requires manual configuration")
	}
	for key := range data {
		if key != "$schema" {
			return errors.New("existing OpenCode JSONC configuration requires manual configuration")
		}
	}
	return nil
}

func claudeUserMCPConfigPath(root string) (string, error) {
	path := filepath.Join(root, ".claude.json")
	return path, safePath(root, path)
}
func lock(root, agent string) (string, func(), error) {
	if _, err := target(root, agent); err != nil {
		return "", nil, err
	}
	folder := filepath.Join(root, ".sub2api-quick-import", agent)
	if err := safePath(root, folder); err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(folder, 0700); err != nil {
		return "", nil, err
	}
	if err := protect(folder, true); err != nil {
		return "", nil, err
	}
	f, err := os.OpenFile(filepath.Join(folder, "lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", nil, errors.New("another import/cleanup is active; inspect recovery directory before retrying")
	}
	f.Close()
	return folder, func() { os.Remove(filepath.Join(folder, "lock")) }, nil
}
func atomicWrite(path, text string) error {
	if err := rejectLink(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".sub2api-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = protect(name, false); err != nil {
		return err
	}
	return replaceFile(name, path)
}
func readText(path string) (string, bool, error) {
	if err := rejectLink(path); err != nil {
		return "", false, err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	return strings.TrimPrefix(string(b), "\ufeff"), err == nil, err
}
func readJournal(folder string) ([]record, error) {
	text, _, err := readText(filepath.Join(folder, "journal.json"))
	if err != nil {
		return nil, err
	}
	if text == "" {
		return []record{}, nil
	}
	var r []record
	err = decodeJSON(text, &r)
	return r, err
}

// Keep JSON integer and decimal spellings intact in unmanaged fields and inverse
// journal values; float64 would round integers above 2^53 during a field merge.
func decodeJSON(text string, dst any) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("unexpected trailing JSON content")
	}
	return nil
}
func writeJournal(folder string, r []record) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(folder, "journal.json"), string(b))
}
func checkPending(r []record) error {
	if len(r) > 0 && (r[len(r)-1].Pending || r[len(r)-1].CleanupPending) {
		return errors.New("interrupted operation: inspect journal and restore its before_text before retrying")
	}
	return nil
}
func load(text, agent string) (map[string]any, error) {
	data := map[string]any{}
	var err error
	if agent == "codex" {
		err = toml.Unmarshal([]byte(text), &data)
	} else if agent == "workbuddy" {
		entries := []any{}
		if strings.TrimSpace(text) != "" {
			err = decodeJSON(text, &entries)
		}
		data["entries"] = entries
	} else if agent == "deepseek-harness" {
		entries := []any{}
		if strings.TrimSpace(text) != "" {
			err = yaml.Unmarshal([]byte(text), &entries)
		}
		data["entries"] = entries
	} else if strings.TrimSpace(text) != "" {
		err = decodeJSON(text, &data)
	}
	if err != nil || data == nil {
		return nil, errors.New("configuration syntax is invalid or is not an object")
	}
	return data, nil
}
func get(data map[string]any, parts []string) value {
	var cur any = data
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return value{}
		}
		cur, ok = m[part]
		if !ok {
			return value{}
		}
	}
	return value{true, cur}
}
func put(data map[string]any, parts []string, v value) error {
	if len(parts) == 0 {
		return errors.New("invalid recovery field")
	}
	key := parts[0]
	if len(parts) == 1 {
		if v.Exists {
			data[key] = v.Value
		} else {
			delete(data, key)
		}
		return nil
	}
	cur, ok := data[key]
	if !ok {
		if !v.Exists {
			return nil
		}
		cur = map[string]any{}
		data[key] = cur
	}
	m, ok := cur.(map[string]any)
	if !ok {
		return errors.New("configuration field conflicts with an object")
	}
	if err := put(m, parts[1:], v); err != nil {
		return err
	}
	if len(m) == 0 {
		delete(data, key)
	}
	return nil
}
func equal(a, b any) bool {
	aa, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aa) == string(bb)
}
func render(text, agent string, changes []change) (string, error) {
	desired, err := load(text, agent)
	if err != nil {
		return "", err
	}
	for _, c := range changes {
		if err := put(desired, c.Path, c.Value); err != nil {
			return "", err
		}
	}
	if agent == "workbuddy" {
		b, err := json.MarshalIndent(desired["entries"], "", "  ")
		return string(b) + "\n", err
	}
	if agent == "deepseek-harness" {
		b, err := yaml.Marshal(desired["entries"])
		return string(b), err
	}
	if agent != "codex" {
		b, err := json.MarshalIndent(desired, "", "  ")
		return string(b) + "\n", err
	}
	result := text
	for _, c := range changes {
		if len(c.Path) == 1 {
			end := len(result)
			if loc := regexp.MustCompile(`(?m)^\s*\[`).FindStringIndex(result); loc != nil {
				end = loc[0]
			}
			head, tail := result[:end], result[end:]
			head = regexp.MustCompile(`(?m)^[ \t]*`+regexp.QuoteMeta(c.Path[0])+`[ \t]*=.*(?:\n|$)`).ReplaceAllString(head, "")
			if c.Value.Exists {
				b, e := json.Marshal(c.Value.Value)
				if e != nil {
					return "", e
				}
				head = c.Path[0] + " = " + string(b) + "\n" + head
			}
			result = head + tail
		} else {
			tableName := ""
			switch {
			case reflect.DeepEqual(c.Path, []string{"model_providers", provider}):
				tableName = "model_providers." + provider
			case reflect.DeepEqual(c.Path, []string{"mcp_servers", "lianjieai_image"}):
				tableName = "mcp_servers.lianjieai_image"
			default:
				return "", errors.New("unsupported TOML change")
			}
			lines := strings.SplitAfter(result, "\n")
			var out strings.Builder
			skip := false
			for _, line := range lines {
				if strings.HasPrefix(line, "[") {
					skip = strings.TrimSpace(line) == "["+tableName+"]"
				}
				if !skip {
					out.WriteString(line)
				}
			}
			result = out.String()
			if c.Value.Exists {
				m, ok := c.Value.Value.(map[string]any)
				if !ok {
					return "", errors.New("invalid TOML table")
				}
				result = strings.TrimRight(result, "\r\n ") + "\n\n[" + tableName + "]\n"
				keys := make([]string, 0, len(m))
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					b, e := tomlScalar(m[k])
					if e != nil {
						return "", e
					}
					result += k + " = " + b + "\n"
				}
			}
		}
	}
	actual, err := load(result, agent)
	if err != nil || !equal(actual, desired) {
		return "", errors.New("complex TOML layout requires manual configuration")
	}
	return result, nil
}

func tomlScalar(v any) (string, error) {
	switch m := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			b, err := tomlScalar(m[k])
			if err != nil {
				return "", err
			}
			parts = append(parts, k+" = "+b)
		}
		return "{ " + strings.Join(parts, ", ") + " }", nil
	default:
		b, err := json.Marshal(v)
		return string(b), err
	}
}
func configuration(p Payload, catalogPath string) ([]change, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	u, err := url.Parse(base)
	if p.Version != 1 || paths[p.Agent] == "" || p.APIKey == "" || strings.TrimSpace(p.Model) == "" || len([]rune(p.Model)) > 200 {
		return nil, errors.New("invalid configuration")
	}
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery || !(u.Scheme == "https" || (u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
		return nil, errors.New("HTTPS gateway required")
	}
	mcpEndpoint, err := resolveMCPEndpoint(p, base)
	if err != nil {
		return nil, err
	}
	models := p.Models
	if models == nil {
		models = []Model{{ID: p.Model, Name: p.Model}}
	}
	selected := false
	seen := map[string]bool{}
	normalized := []Model{}
	for _, m := range models {
		valid := strings.TrimSpace(m.ID) != "" && len([]rune(m.ID)) <= 200
		for _, r := range m.ID {
			if r < 32 {
				valid = false
			}
		}
		if !valid || seen[m.ID] || quickimport.IsClearlyNonChatModel(m.ID) {
			continue
		}
		seen[m.ID] = true
		if m.ID == p.Model {
			selected = true
		}
		if strings.TrimSpace(m.Name) == "" {
			m.Name = m.ID
		}
		normalized = append(normalized, m)
	}
	if !selected {
		return nil, errors.New("selected model is not in the gateway model list")
	}
	changes := []change{}
	add := func(path []string, v any) { changes = append(changes, change{path, value{true, v}}) }
	switch p.Agent {
	case "claude":
		for _, pair := range [][2]string{{"ANTHROPIC_BASE_URL", base}, {"ANTHROPIC_AUTH_TOKEN", p.APIKey}, {"ANTHROPIC_MODEL", p.Model}, {"ANTHROPIC_DEFAULT_OPUS_MODEL", p.Model}, {"ANTHROPIC_DEFAULT_SONNET_MODEL", p.Model}, {"ANTHROPIC_DEFAULT_HAIKU_MODEL", p.Model}, {"ANTHROPIC_DEFAULT_FABLE_MODEL", p.Model}, {"CLAUDE_CODE_SUBAGENT_MODEL", p.Model}, {"ANTHROPIC_CUSTOM_MODEL_OPTION", p.Model}, {"ANTHROPIC_CUSTOM_MODEL_OPTION_NAME", "lianjieai · " + p.Model}, {"ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION", "lianjieai gateway"}, {"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", "1"}} {
			add([]string{"env", pair[0]}, pair[1])
		}
		if p.ClaudeModelPickerSupported {
			options := []any{}
			for _, m := range normalized {
				options = append(options, map[string]any{"model": m.ID, "label": "lianjieai · " + m.Name})
			}
			add([]string{"modelPicker"}, map[string]any{"options": options, "replaceBuiltInOptions": true})
		}
		if mcpImageToolsEnabled(p) {
			add([]string{"mcpServers", "lianjieai-image"}, map[string]any{
				"type":    "http",
				"url":     mcpEndpoint,
				"headers": map[string]any{"Authorization": "Bearer " + p.APIKey},
			})
		}
	case "codex":
		add([]string{"model"}, p.Model)
		add([]string{"model_provider"}, provider)
		add([]string{"model_providers", provider}, map[string]any{"name": "lianjieai", "base_url": base, "wire_api": "responses", "experimental_bearer_token": p.APIKey, "requires_openai_auth": false})
		if catalogPath != "" {
			add([]string{"model_catalog_json"}, catalogPath)
		}
		if mcpImageToolsEnabled(p) {
			add([]string{"mcp_servers", "lianjieai_image"}, map[string]any{
				"url":                 mcpEndpoint,
				"http_headers":        map[string]any{"Authorization": "Bearer " + p.APIKey},
				"startup_timeout_sec": 20,
				"tool_timeout_sec":    300,
			})
		}
	case "opencode":
		protocol := p.Protocol
		if protocol == "" {
			protocol = "openai"
		}
		npm := map[string]string{"openai": "@ai-sdk/openai", "anthropic": "@ai-sdk/anthropic", "compatible": "@ai-sdk/openai-compatible", "gemini": "@ai-sdk/google"}[protocol]
		if npm == "" {
			return nil, errors.New("unsupported protocol")
		}
		models := map[string]any{}
		for _, m := range normalized {
			models[m.ID] = map[string]any{"name": m.Name}
		}
		add([]string{"provider", provider}, map[string]any{"npm": npm, "name": "lianjieai", "options": map[string]any{"baseURL": base, "apiKey": p.APIKey}, "models": models})
		add([]string{"model"}, provider+"/"+p.Model)
		if mcpImageToolsEnabled(p) {
			changes = append(changes, change{Path: []string{"mcp", "sub2api_image"}, Value: value{Exists: false}})
			changes = append(changes, change{Path: []string{"mcp", "servers", "sub2api_image"}, Value: value{Exists: false}})
			// OpenCode v2 places each server under mcp.servers.
			add([]string{"mcp", "servers", "lianjieai_image"}, map[string]any{
				"type":    "remote",
				"url":     mcpEndpoint,
				"oauth":   false,
				"headers": map[string]any{"Authorization": "Bearer " + p.APIKey},
			})
		}
	case "workbuddy":
		entries := make([]any, 0, 1)
		for _, m := range normalized {
			if m.ID != p.Model {
				continue
			}
			// WorkBuddy must use the full endpoint as-is, without adding a second /chat/completions.
			entries = append(entries, map[string]any{"id": m.ID, "name": m.Name, "vendor": "Custom", "url": base + "/chat/completions", "apiKey": p.APIKey, "supportsToolCall": true, "supportsImages": false, "useCustomProtocol": true})
		}
		add([]string{"entries"}, entries)
	case "pi":
		entries := make([]any, 0, 1)
		for _, m := range normalized {
			if m.ID != p.Model {
				continue
			}
			entries = append(entries, map[string]any{"id": m.ID, "name": m.Name})
		}
		add([]string{"$schema"}, "https://pi.dev/schemas/models.schema.json")
		add([]string{"providers", "lianjieai"}, map[string]any{"baseUrl": base, "api": "openai-completions", "apiKey": p.APIKey, "models": entries})
	case "deepseek-harness":
		// The existing profile entries are read and preserved by Install below.
	}
	return changes, nil
}

func claudeMCPChange(changes []change) (change, bool) {
	for _, c := range changes {
		if reflect.DeepEqual(c.Path, []string{"mcpServers", "lianjieai-image"}) {
			return c, true
		}
	}
	return change{}, false
}

func syncClaudeUserMCPConfig(root string, changes []change) error {
	c, ok := claudeMCPChange(changes)
	if !ok || !c.Value.Exists {
		return nil
	}
	path, err := claudeUserMCPConfigPath(root)
	if err != nil {
		return err
	}
	before, _, err := readText(path)
	if err != nil {
		return err
	}
	after, err := render(before, "claude", []change{c})
	if err != nil {
		return err
	}
	if after == before {
		return nil
	}
	return atomicWrite(path, after)
}

func cleanClaudeUserMCPConfig(root string, rec *record) error {
	if rec == nil {
		return nil
	}
	for _, c := range rec.Changes {
		if !reflect.DeepEqual(c.Path, []string{"mcpServers", "lianjieai-image"}) || !c.Value.Exists {
			continue
		}
		path, err := claudeUserMCPConfigPath(root)
		if err != nil {
			return err
		}
		current, exists, err := readText(path)
		if err != nil || !exists {
			return err
		}
		data, err := load(current, "claude")
		if err != nil {
			return err
		}
		if !equal(get(data, c.Path), c.Value) {
			return nil
		}
		result, err := render(current, "claude", []change{{Path: c.Path, Value: value{Exists: false}}})
		if err != nil {
			return err
		}
		restored, err := load(result, "claude")
		if err != nil {
			return err
		}
		if len(restored) == 0 {
			if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		return atomicWrite(path, result)
	}
	return nil
}

func mcpImageToolsEnabled(p Payload) bool {
	return p.MCPImageToolsEnabled != nil && *p.MCPImageToolsEnabled
}

func resolveMCPEndpoint(p Payload, base string) (string, error) {
	endpoint := strings.TrimSpace(p.MCPEndpoint)
	if endpoint == "" && mcpImageToolsEnabled(p) {
		endpoint = deriveMCPEndpoint(base)
	}
	if endpoint == "" {
		return "", nil
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery ||
		!(u.Scheme == "https" || (u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
		return "", errors.New("invalid MCP endpoint")
	}
	return strings.TrimRight(endpoint, "/"), nil
}

func deriveMCPEndpoint(base string) string {
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return ""
	}
	path := strings.TrimRight(u.Path, "/")
	if path == "/v1" {
		path = ""
	} else if strings.HasSuffix(path, "/v1") {
		path = strings.TrimSuffix(path, "/v1")
	}
	u.Path = strings.TrimRight(path, "/") + "/mcp"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
func ownedPath(root, folder string, item ownedFile) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(strings.ReplaceAll(item.Path, `\`, "/")))
	if filepath.Dir(path) != folder || !regexp.MustCompile(`^models-[a-f0-9]{32}\.json$`).MatchString(filepath.Base(path)) {
		return "", errors.New("invalid recovery catalog path")
	}
	return path, safePath(root, path)
}
func Install(root string, p Payload) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	path, err := target(root, p.Agent)
	if err != nil {
		return err
	}
	if p.Agent == "opencode" {
		if err := checkOpenCodeJSONC(path); err != nil {
			return err
		}
	}
	folder, unlock, err := lock(root, p.Agent)
	if err != nil {
		return err
	}
	defer unlock()
	records, err := readJournal(folder)
	if err != nil {
		return err
	}
	if err = checkPending(records); err != nil {
		return err
	}
	owned := []ownedFile{}
	catalogPath := ""
	if p.Agent == "codex" && p.CodexManifest != nil {
		items, ok := p.CodexManifest["models"].([]any)
		found := false
		if ok {
			for _, item := range items {
				m, ok := item.(map[string]any)
				if ok && m["slug"] == p.Model {
					found = true
				}
			}
		}
		if !found {
			return errors.New("selected model is not in the Codex model catalog")
		}
		id := make([]byte, 16)
		if _, err = rand.Read(id); err != nil {
			return err
		}
		catalogPath = filepath.Join(folder, "models-"+hex.EncodeToString(id)+".json")
		rel, _ := filepath.Rel(root, catalogPath)
		b, e := json.MarshalIndent(p.CodexManifest, "", "  ")
		if e != nil {
			return e
		}
		owned = append(owned, ownedFile{rel, string(b) + "\n"})
	}
	changes, err := configuration(p, catalogPath)
	if err != nil {
		return err
	}
	before, existed, err := readText(path)
	if err != nil {
		return err
	}
	original, err := load(before, p.Agent)
	if err != nil {
		return err
	}
	if p.Agent == "workbuddy" {
		for i := range changes {
			if changes[i].Path[0] == "entries" {
				incoming, _ := changes[i].Value.Value.([]any)
				ids := map[string]bool{}
				for _, item := range incoming {
					if m, ok := item.(map[string]any); ok {
						if id, ok := m["id"].(string); ok {
							ids[id] = true
						}
					}
				}
				if existing, ok := original["entries"].([]any); ok {
					for _, item := range existing {
						m, ok := item.(map[string]any)
						if !ok {
							incoming = append(incoming, item)
							continue
						}
						id, ok := m["id"].(string)
						if !ok || !ids[id] {
							incoming = append(incoming, item)
						}
					}
				}
				changes[i].Value.Value = incoming
			}
		}
	}
	if p.Agent == "deepseek-harness" {
		entries, ok := original["entries"].([]any)
		if !ok {
			return errors.New("invalid DeepSeek Harness profile")
		}
		modelName := p.Model
		for _, m := range p.Models {
			if m.ID == p.Model {
				modelName = m.Name
				break
			}
		}
		route := map[string]any{
			"displayName": "lianjieai",
			"api":         "openai-completions",
			"baseURL":     strings.TrimRight(p.BaseURL, "/"),
			"headers":     map[string]any{"Authorization": "Bearer " + p.APIKey},
			"models":      []any{map[string]any{"id": p.Model, "name": modelName}},
		}
		if p.Platform == "deepseek" {
			route["compat"] = map[string]any{
				"supportsDeveloperRole": false,
				"maxTokensField":        "max_tokens",
			}
		}
		updated := append([]any{}, entries...)
		found := false
		for i, item := range entries {
			entry, ok := item.(map[string]any)
			if !ok || (entry["id"] != "llm-pi-ai" && entry["name"] != "@deepseek-ai/dsh-llm-pi-ai") {
				continue
			}
			if found {
				return errors.New("multiple DeepSeek Harness model providers require manual configuration")
			}
			found = true
			copyEntry := make(map[string]any, len(entry))
			for key, value := range entry {
				copyEntry[key] = value
			}
			config, ok := entry["config"].(map[string]any)
			if !ok && entry["config"] != nil {
				return errors.New("invalid DeepSeek Harness provider configuration")
			}
			copyConfig := make(map[string]any, len(config)+1)
			for key, value := range config {
				copyConfig[key] = value
			}
			providers, ok := config["providers"].(map[string]any)
			if !ok && config["providers"] != nil {
				return errors.New("invalid DeepSeek Harness provider list")
			}
			copyProviders := make(map[string]any, len(providers)+1)
			for key, value := range providers {
				copyProviders[key] = value
			}
			copyProviders["lianjieai"] = route
			copyConfig["providers"] = copyProviders
			copyEntry["config"] = copyConfig
			updated[i] = copyEntry
		}
		if !found {
			updated = append(updated, map[string]any{"id": "llm-pi-ai", "config": map[string]any{"providers": map[string]any{"lianjieai": route}}})
		}
		changes = append(changes, change{Path: []string{"entries"}, Value: value{Exists: true, Value: updated}})
	}
	after, err := render(before, p.Agent, changes)
	if err != nil {
		return err
	}
	if after == before {
		return nil
	}
	rec := record{Agent: p.Agent, Existed: existed, Before: before, After: after, Changes: changes, Pending: true, Owned: owned}
	for _, c := range changes {
		rec.Inverse = append(rec.Inverse, change{c.Path, get(original, c.Path)})
	}
	records = append(records, rec)
	if err = writeJournal(folder, records); err != nil {
		return err
	}
	operation := func() error {
		for _, item := range owned {
			aux, e := ownedPath(root, folder, item)
			if e != nil {
				return e
			}
			if e = atomicWrite(aux, item.Text); e != nil {
				return e
			}
		}
		if e := atomicWrite(path, after); e != nil {
			return e
		}
		records[len(records)-1].Pending = false
		return writeJournal(folder, records)
	}
	if err = operation(); err != nil {
		rollback := func() error {
			if existed {
				if e := atomicWrite(path, before); e != nil {
					return e
				}
			} else if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
				return e
			}
			for _, item := range owned {
				aux, e := ownedPath(root, folder, item)
				if e != nil {
					return e
				}
				if e = os.Remove(aux); e != nil && !os.IsNotExist(e) {
					return e
				}
			}
			return writeJournal(folder, records[:len(records)-1])
		}()
		return errors.Join(err, rollback)
	}
	if p.Agent == "claude" {
		if err = syncClaudeUserMCPConfig(root, changes); err != nil {
			return err
		}
	}
	return nil
}

type RecoveryConflict struct {
	Field string
}

func (conflict *RecoveryConflict) Error() string {
	return "configuration conflict at " + conflict.Field + "; later edits preserved. Review this field before retrying cleanup"
}

func Clean(root, agent string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	path, err := target(root, agent)
	if err != nil {
		return err
	}
	folder, unlock, err := lock(root, agent)
	if err != nil {
		return err
	}
	defer unlock()
	records, err := readJournal(folder)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.New("no configuration to clean for this Agent")
	}
	rec := &records[len(records)-1]
	if rec.Agent != agent {
		return errors.New("Agent mismatch")
	}
	current, exists, err := readText(path)
	if err != nil {
		return err
	}
	for _, item := range rec.Owned {
		aux, e := ownedPath(root, folder, item)
		if e != nil {
			return e
		}
		b, e := os.ReadFile(aux)
		if os.IsNotExist(e) && rec.CleanupPending {
			continue
		}
		if e != nil || string(b) != item.Text {
			return errors.New("model catalog conflict: later edits preserved")
		}
	}
	removeOwned := func() error {
		for _, item := range rec.Owned {
			aux, e := ownedPath(root, folder, item)
			if e != nil {
				return e
			}
			if e = os.Remove(aux); e != nil && !os.IsNotExist(e) {
				return e
			}
		}
		return nil
	}
	if rec.CleanupPending {
		if current == rec.CleanupResult || (!exists && rec.CleanupDelete) {
			if err = removeOwned(); err != nil {
				return err
			}
			return writeJournal(folder, records[:len(records)-1])
		}
		if current != rec.CleanupBefore {
			return errors.New("interrupted cleanup conflict: subsequent edits preserved")
		}
		rec.CleanupPending = false
		if err = writeJournal(folder, records); err != nil {
			return err
		}
	}
	if err = checkPending(records); err != nil {
		return err
	}
	if agent == "claude" {
		if err = cleanClaudeUserMCPConfig(root, rec); err != nil {
			return err
		}
	}
	data, err := load(current, agent)
	if err != nil {
		return err
	}
	for _, c := range rec.Changes {
		if !equal(get(data, c.Path), c.Value) {
			if agent == "codex" && reflect.DeepEqual(c.Path, []string{"model"}) && data["model_provider"] == provider {
				if selected, ok := data["model"].(string); ok && strings.TrimSpace(selected) != "" {
					continue
				}
			}
			field := "managed configuration"
			if agent == "codex" {
				switch strings.Join(c.Path, ".") {
				case "model", "model_provider", "model_catalog_json", "model_providers.sub2api_quick", "mcp_servers.lianjieai_image":
					field = strings.Join(c.Path, ".")
				}
			}
			return &RecoveryConflict{Field: field}
		}
	}
	result := rec.Before
	if current != rec.After {
		result, err = render(current, agent, rec.Inverse)
		if err != nil {
			return err
		}
	}
	restored, err := load(result, agent)
	if err != nil {
		return err
	}
	rec.CleanupPending = true
	rec.CleanupBefore = current
	rec.CleanupResult = result
	rec.CleanupDelete = !rec.Existed && (len(restored) == 0 || (agent == "workbuddy" || agent == "deepseek-harness") && equal(restored["entries"], []any{}))
	if err = writeJournal(folder, records); err != nil {
		return err
	}
	operation := func() error {
		if rec.CleanupDelete {
			if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
				return e
			}
		} else if e := atomicWrite(path, result); e != nil {
			return e
		}
		if e := removeOwned(); e != nil {
			return e
		}
		return writeJournal(folder, records[:len(records)-1])
	}
	if err = operation(); err != nil {
		rollback := func() error {
			for _, item := range rec.Owned {
				aux, e := ownedPath(root, folder, item)
				if e != nil {
					return e
				}
				if e = atomicWrite(aux, item.Text); e != nil {
					return e
				}
			}
			if e := atomicWrite(path, current); e != nil {
				return e
			}
			rec.CleanupPending = false
			return writeJournal(folder, records)
		}()
		return errors.Join(err, rollback)
	}
	return nil
}

// ValidatePayload checks configuration without writing files or exposing parser details.
func ValidatePayload(p Payload) error {
	_, err := configuration(p, "")
	if err != nil {
		return fmt.Errorf("invalid quick-import payload: %w", err)
	}
	return nil
}
