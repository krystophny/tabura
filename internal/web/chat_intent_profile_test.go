package web

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestParseIntentLLMProfileOptions(t *testing.T) {
	options := parseIntentLLMProfileOptions(" qwen3.6-35b-a3b-q4, qwen3.6-35b-a3b-q4 ")
	if len(options) != 1 {
		t.Fatalf("options length = %d, want 1", len(options))
	}
	if options[0] != "qwen3.6-35b-a3b-q4" {
		t.Fatalf("option[0] = %q, want qwen3.6-35b-a3b-q4", options[0])
	}
}

func TestResolveIntentLLMProfileDefaults(t *testing.T) {
	if got := resolveIntentLLMProfile(""); got != DefaultIntentLLMProfile {
		t.Fatalf("default profile = %q, want %q", got, DefaultIntentLLMProfile)
	}
	if got := resolveIntentLLMProfile("QWEN3.6-35B-A3B-Q4"); got != "qwen3.6-35b-a3b-q4" {
		t.Fatalf("normalized profile = %q, want qwen3.6-35b-a3b-q4", got)
	}
}

func TestRuntimeIncludesIntentDelegatorProfiles(t *testing.T) {
	app := newAuthedTestApp(t)
	rr := doAuthedJSONRequest(t, app.Router(), http.MethodGet, "/api/runtime", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("runtime status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime response: %v", err)
	}
	if got := strFromAny(payload["intent_llm_profile"]); got != "qwen3.6-35b-a3b-q4" {
		t.Fatalf("intent_llm_profile = %q, want qwen3.6-35b-a3b-q4", got)
	}
	profiles, _ := payload["available_intent_llm_profiles"].([]any)
	if len(profiles) != 1 {
		t.Fatalf("available_intent_llm_profiles len = %d, want 1", len(profiles))
	}
	if first := strFromAny(profiles[0]); first != "qwen3.6-35b-a3b-q4" {
		t.Fatalf("available_intent_llm_profiles[0] = %q, want qwen3.6-35b-a3b-q4", first)
	}
}

func TestClassifyIntentWithLLM_NoLocalModelHintShortcut(t *testing.T) {
	app := newAuthedTestApp(t)
	app.intentLLMURL = "http://127.0.0.1:1"

	_, err := app.classifyIntentWithLLM(context.Background(), "let codex audit this repo")
	if err == nil {
		t.Fatal("expected network error when LLM endpoint is unavailable")
	}
}
