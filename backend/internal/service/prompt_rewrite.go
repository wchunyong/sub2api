package service

import "encoding/json"

const (
	promptRewriteBlocked = "Tokens prefixed with @ are paths the user explicitly referenced. Relative paths resolve from the workspace root"
	promptRewriteSafe    = "Tokens prefixed with an at-sign are paths the user explicitly referenced. Relative paths resolve from the workspace root"
)

// RewritePromptForUpstream applies the opt-in group rewrite immediately before
// an upstream request. It changes only system/developer message text in
// OpenAI Chat Completions messages and Responses input items.
func RewritePromptForUpstream(body []byte, group *Group) []byte {
	if group == nil || !group.PromptRewrite {
		return body
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(body, &root) != nil {
		return body
	}
	touched := false
	for _, key := range []string{"messages", "input"} {
		raw, ok := root[key]
		if !ok {
			continue
		}
		var items []map[string]json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			continue
		}
		changed := false
		for _, item := range items {
			var role string
			_ = json.Unmarshal(item["role"], &role)
			if role != "system" && role != "developer" {
				continue
			}
			if rewriteMessageContent(item) {
				changed = true
			}
		}
		if changed {
			root[key], _ = json.Marshal(items)
			touched = true
		}
	}
	// Responses requests commonly carry the system prompt in the top-level
	// instructions field. Anthropic Messages requests use a top-level system
	// field instead of a system message, so cover both shapes as well.
	for _, key := range []string{"instructions", "system"} {
		raw, ok := root[key]
		if !ok {
			continue
		}
		if next, changed := rewriteRawPromptContent(raw); changed {
			root[key] = next
			touched = true
		}
	}
	if !touched {
		return body
	}
	out, err := json.Marshal(root)
	if err != nil {
		return body
	}
	return out
}

func rewriteMessageContent(message map[string]json.RawMessage) bool {
	raw, ok := message["content"]
	if !ok {
		return false
	}
	next, changed := rewriteRawPromptContent(raw)
	if changed {
		message["content"] = next
	}
	return changed
}

func rewriteRawPromptContent(raw json.RawMessage) (json.RawMessage, bool) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		next := rewritePromptText(text)
		if next == text {
			return nil, false
		}
		encoded, err := json.Marshal(next)
		if err != nil {
			return nil, false
		}
		return encoded, true
	}
	var parts []map[string]json.RawMessage
	if json.Unmarshal(raw, &parts) != nil {
		return nil, false
	}
	changed := false
	for _, part := range parts {
		var partText string
		if json.Unmarshal(part["text"], &partText) != nil {
			continue
		}
		next := rewritePromptText(partText)
		if next != partText {
			part["text"], _ = json.Marshal(next)
			changed = true
		}
	}
	if changed {
		encoded, err := json.Marshal(parts)
		if err != nil {
			return nil, false
		}
		return encoded, true
	}
	return nil, false
}

func rewritePromptText(text string) string {
	return replaceAll(text, promptRewriteBlocked, promptRewriteSafe)
}

func replaceAll(text, old, new string) string {
	for {
		i := indexOf(text, old)
		if i < 0 {
			return text
		}
		text = text[:i] + new + text[i+len(old):]
	}
}

func indexOf(text, needle string) int {
	for i := 0; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
