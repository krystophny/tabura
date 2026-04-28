package web

import (
	"testing"
)

func TestLocalAssistantWebMCPToolsFiltersAndTagsURL(t *testing.T) {
	tools := []mcpListedTool{
		{Name: "web_search", Description: "SearXNG web search.", InputSchema: map[string]any{"type": "object"}},
		{Name: "web_fetch", Description: "Fetch a URL.", InputSchema: map[string]any{"type": "object"}},
		{Name: "calendar_events", Description: "List events."},
		{Name: "mail_account_list", Description: "List mail accounts."},
	}
	got := localAssistantWebMCPTools(tools)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 web tools, got: %+v", len(got), got)
	}
	names := map[string]bool{}
	for _, tool := range got {
		if !tool.WebTool {
			t.Errorf("tool %q WebTool = false, want true", tool.InternalName)
		}
		if tool.Kind != localAssistantToolKindMCP {
			t.Errorf("tool %q Kind = %q, want mcp", tool.InternalName, tool.Kind)
		}
		names[tool.InternalName] = true
	}
	for _, want := range []string{"web_search", "web_fetch"} {
		if !names[want] {
			t.Errorf("missing web tool %q in catalog: %+v", want, names)
		}
	}
}

func TestLocalAssistantIsWebToolAcceptsCommonNames(t *testing.T) {
	accept := []string{"web_search", "web_fetch", "searxng_search", "searxng_fetch", "mcp__web_search"}
	for _, name := range accept {
		if !localAssistantIsWebTool(name) {
			t.Errorf("localAssistantIsWebTool(%q) = false, want true", name)
		}
	}
	reject := []string{"mail_account_list", "calendar_events", "item_list", ""}
	for _, name := range reject {
		if localAssistantIsWebTool(name) {
			t.Errorf("localAssistantIsWebTool(%q) = true, want false", name)
		}
	}
}
