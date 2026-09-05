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
}

func BuildConfig(agent, platform string, messages bool, baseURL, apiKey, model string) (Config, error) {
	fail := errors.New("unsupported Agent, group or gateway configuration")
	if agent != "claude" && agent != "codex" && agent != "opencode" {
		return Config{}, fail
	}
	defaults := map[string]string{"openai": "gpt-5.5", "anthropic": "claude-sonnet-4-6", "antigravity": "claude-sonnet-4-6", "gemini": "gemini-2.5-pro", "deepseek": "deepseek-v4-pro", "grok": "grok-4.5", "kimi": "kimi-k2.5", "zhipu": "glm-4.7", "composite": "gpt-5.5"}
	fallback, ok := defaults[platform]
	if !ok {
		return Config{}, fail
	}
	if agent == "claude" && (platform == "gemini" || platform == "openai" && !messages) {
		return Config{}, fail
	}
	if model == "" {
		model = fallback
	}
	if len(model) > 200 || strings.ContainsAny(model, "\r\n\x00") {
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
		case "grok":
			protocol = "compatible"
		}
	}
	return Config{Version: 1, Agent: agent, APIKey: apiKey, BaseURL: endpoint, Model: model, Protocol: protocol}, nil
}
