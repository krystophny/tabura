package llmcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var utcTimeLineRe = regexp.MustCompile(`(?m)^Current UTC time:\s*(\d{4}-\d{2}-\d{2})T[^\n]*$`)
var runningTasksLineRe = regexp.MustCompile(`(?m)^Running tasks:\s*[^\n]*$`)

// BuildKey produces a SHA256 cache key from the LLM request components.
func BuildKey(messages []map[string]any, tools []map[string]any, model string, enableThinking bool) string {
	normalized := normalizeMessages(messages)
	messagesJSON, _ := json.Marshal(normalized)
	toolsJSON, _ := json.Marshal(tools)
	thinking := "0"
	if enableThinking {
		thinking = "1"
	}
	h := sha256.New()
	fmt.Fprintf(h, "m:%d:%s\n", len(messagesJSON), messagesJSON)
	fmt.Fprintf(h, "t:%d:%s\n", len(toolsJSON), toolsJSON)
	fmt.Fprintf(h, "model:%s\n", model)
	fmt.Fprintf(h, "think:%s\n", thinking)
	return hex.EncodeToString(h.Sum(nil))
}

// ContainsToolResults returns true if any message has role "tool",
// indicating this is a follow-up round with live tool results that
// should not be cached.
func ContainsToolResults(messages []map[string]any) bool {
	for _, msg := range messages {
		if role, _ := msg["role"].(string); role == "tool" {
			return true
		}
	}
	return false
}

func normalizeMessages(messages []map[string]any) []map[string]any {
	out := make([]map[string]any, len(messages))
	for i, msg := range messages {
		cp := make(map[string]any, len(msg))
		for k, v := range msg {
			cp[k] = v
		}
		if content, ok := cp["content"].(string); ok {
			cp["content"] = normalizeContent(content)
		}
		out[i] = cp
	}
	return out
}

func normalizeContent(content string) string {
	// Replace full ISO timestamp with date-only so same-day queries
	// produce the same cache key but next-day queries miss.
	content = utcTimeLineRe.ReplaceAllString(content, "Current UTC time: $1")
	// Strip ephemeral scheduler state.
	content = runningTasksLineRe.ReplaceAllString(content, "")
	return strings.TrimRight(content, " \t\n")
}
