package quickimport

import (
	"embed"
	"errors"
	"net/url"
	"strings"
)

//go:embed assets/installer.py assets/install.ps1 assets/install.sh
var Assets embed.FS

type Config struct {
	Version  int    `json:"version"`
	Agent    string `json:"agent"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Protocol string `json:"protocol"`
	ProbeURL string `json:"probe_url"`
	// MCPEndpoint is always rooted at the gateway host, even when BaseURL points
	// at protocol-specific paths such as /v1beta or /antigravity/v1.
	MCPEndpoint string `json:"mcp_endpoint,omitempty"`
}

func BuildConfig(agent, platform string, messages bool, baseURL, apiKey, model string) (Config, error) {
	fail := errors.New("unsupported Agent, group or gateway configuration")
	if agent != "claude" && agent != "codex" && agent != "opencode" && agent != "workbuddy" && agent != "pi" && agent != "deepseek-harness" {
		return Config{}, fail
	}
	platforms := map[string]bool{"openai": true, "anthropic": true, "antigravity": true, "gemini": true, "deepseek": true, "grok": true, "kimi": true, "zhipu": true, "composite": true}
	_, ok := platforms[platform]
	if !ok {
		return Config{}, fail
	}
	if agent == "claude" && (platform == "gemini" || platform == "openai" && !messages) {
		return Config{}, fail
	}
	if agent == "codex" && platform != "openai" && platform != "composite" {
		return Config{}, fail
	}
	if strings.TrimSpace(model) == "" || len(model) > 200 || strings.ContainsAny(model, "\r\n\x00") {
		return Config{}, fail
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Config{}, fail
	}
	root := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
	protocol := "openai"
	endpoint := root + "/v1"
	if agent == "claude" {
		protocol = "anthropic"
		endpoint = root
		if platform == "antigravity" {
			endpoint += "/antigravity"
		}
	}
	if agent == "opencode" {
		switch platform {
		case "anthropic":
			protocol = "anthropic"
		case "antigravity":
			protocol = "anthropic"
			endpoint = root + "/antigravity/v1"
		case "gemini":
			protocol = "gemini"
			endpoint = root + "/v1beta"
		case "grok", "deepseek", "kimi", "zhipu":
			protocol = "compatible"
		}
	}
	if agent == "workbuddy" || agent == "pi" || agent == "deepseek-harness" {
		protocol = "openai-completions"
	}
	return Config{Version: 1, Agent: agent, APIKey: apiKey, BaseURL: endpoint, Model: model, Protocol: protocol, ProbeURL: root + "/v1/models", MCPEndpoint: root + "/mcp"}, nil
}
