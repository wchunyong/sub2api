package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestRewritePromptForUpstream(t *testing.T) {
	blocked := promptRewriteBlocked
	cases := []struct {
		name    string
		body    string
		enabled bool
		want    string
	}{
		{"system string", `{"messages":[{"role":"system","content":"` + blocked + `"}]}`, true, promptRewriteSafe},
		{"developer blocks", `{"messages":[{"role":"developer","content":[{"type":"text","text":"` + blocked + `"}]}]}`, true, promptRewriteSafe},
		{"user unchanged", `{"messages":[{"role":"user","content":"` + blocked + `"}]}`, true, blocked},
		{"disabled", `{"messages":[{"role":"system","content":"` + blocked + `"}]}`, false, blocked},
		{"nonmatching", `{"messages":[{"role":"system","content":"hello"}]}`, true, "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RewritePromptForUpstream([]byte(tc.body), &Group{PromptRewrite: tc.enabled})
			var payload struct {
				Messages []struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(got, &payload); err != nil {
				t.Fatal(err)
			}
			var content string
			if err := json.Unmarshal(payload.Messages[0].Content, &content); err != nil {
				var parts []struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(payload.Messages[0].Content, &parts); err != nil {
					t.Fatal(err)
				}
				content = parts[0].Text
			}
			if content != tc.want {
				t.Fatalf("content = %q, want %q", content, tc.want)
			}
		})
	}
}

func TestRewritePromptForUpstreamTopLevelSystemFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		path string
	}{
		{name: "responses instructions", body: `{"instructions":"` + promptRewriteBlocked + `"}`, path: "instructions"},
		{name: "responses input developer", body: `{"input":[{"type":"message","role":"developer","content":[{"type":"input_text","text":"` + promptRewriteBlocked + `"}]}]}`, path: "input.0.content.0.text"},
		{name: "anthropic top-level system", body: `{"system":[{"type":"text","text":"` + promptRewriteBlocked + `"}]}`, path: "system.0.text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := RewritePromptForUpstream([]byte(tc.body), &Group{PromptRewrite: true})
			var payload map[string]any
			if err := json.Unmarshal(got, &payload); err != nil {
				t.Fatal(err)
			}
			var value any = payload
			for _, part := range strings.Split(tc.path, ".") {
				switch current := value.(type) {
				case map[string]any:
					value = current[part]
				case []any:
					var index int
					if _, err := fmt.Sscanf(part, "%d", &index); err != nil {
						t.Fatalf("invalid path segment %q: %v", part, err)
					}
					value = current[index]
				default:
					t.Fatalf("path %q stopped at %T", tc.path, value)
				}
			}
			if value != promptRewriteSafe {
				t.Fatalf("value = %v, want %q", value, promptRewriteSafe)
			}
		})
	}
}

func TestRewritePromptForUpstreamPreservesUnrelatedFields(t *testing.T) {
	body := []byte(`{"model":"x","tools":[{"type":"function"}],"messages":[{"role":"system","name":"keep","content":"` + promptRewriteBlocked + `"}]}`)
	got := RewritePromptForUpstream(body, &Group{PromptRewrite: true})
	if !bytes.Contains(got, []byte(`"model":"x"`)) || !bytes.Contains(got, []byte(`"tools":[{"type":"function"}]`)) || !bytes.Contains(got, []byte(`"name":"keep"`)) {
		t.Fatalf("unrelated fields were lost: %s", got)
	}
}
