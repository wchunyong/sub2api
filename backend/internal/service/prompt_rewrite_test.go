package service

import (
	"bytes"
	"encoding/json"
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

func TestRewritePromptForUpstreamPreservesUnrelatedFields(t *testing.T) {
	body := []byte(`{"model":"x","tools":[{"type":"function"}],"messages":[{"role":"system","name":"keep","content":"` + promptRewriteBlocked + `"}]}`)
	got := RewritePromptForUpstream(body, &Group{PromptRewrite: true})
	if !bytes.Contains(got, []byte(`"model":"x"`)) || !bytes.Contains(got, []byte(`"tools":[{"type":"function"}]`)) || !bytes.Contains(got, []byte(`"name":"keep"`)) {
		t.Fatalf("unrelated fields were lost: %s", got)
	}
}
