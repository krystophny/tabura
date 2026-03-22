package web

import (
	"regexp"
	"strings"
)

var (
	canvasNextPagePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(?:please\s+)?(?:go\s+to\s+)?(?:the\s+)?next\s+(?:slide|page|folie|seite|section|abschnitt)\b`),
		regexp.MustCompile(`(?i)^(?:let(?:'s| us)\s+)?(?:move|go)\s+on(?:\s+to\s+the\s+next\s+(?:slide|page|section))?\b`),
		regexp.MustCompile(`(?i)^(?:zur|auf\s+die|auf\s+der)\s+n[aä]chsten\s+(?:folie|seite)\b`),
		regexp.MustCompile(`(?i)^gehen\s+wir\s+weiter\b`),
		regexp.MustCompile(`(?i)^weiter\s+zur\s+n[aä]chsten\s+(?:folie|seite)\b`),
	}
	canvasPreviousPagePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(?:go\s+)?back(?:\s+one)?(?:\s+(?:slide|page))?\b`),
		regexp.MustCompile(`(?i)^(?:previous|prior|last)\s+(?:slide|page|folie|seite|section|abschnitt)\b`),
		regexp.MustCompile(`(?i)^(?:noch\s+einmal\s+)?zur[uü]ck\b`),
		regexp.MustCompile(`(?i)^eine\s+(?:folie|seite)\s+zur[uü]ck\b`),
		regexp.MustCompile(`(?i)^(?:vorherige|vorige)\s+(?:folie|seite)\b`),
	}
	canvasNextArtifactPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(?:please\s+)?(?:go\s+to\s+)?(?:the\s+)?next\s+(?:document|artifact|file|doc|deck)\b`),
		regexp.MustCompile(`(?i)^(?:switch|jump|move)\s+to\s+the\s+next\s+(?:document|artifact|file|deck)\b`),
		regexp.MustCompile(`(?i)^(?:zum|zur)\s+n[aä]chsten\s+(?:dokument|artefakt|datei)\b`),
	}
	canvasPreviousArtifactPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(?:go\s+to\s+)?(?:the\s+)?(?:previous|prior|last)\s+(?:document|artifact|file|doc|deck)\b`),
		regexp.MustCompile(`(?i)^(?:switch|jump|move)\s+to\s+the\s+(?:previous|prior|last)\s+(?:document|artifact|file|deck)\b`),
		regexp.MustCompile(`(?i)^(?:zum|zur)\s+(?:vorherigen|vorigen)\s+(?:dokument|artefakt|datei)\b`),
	}
)

func normalizeCanvasNavigationCommandText(raw string) string {
	trimmed, _ := stripHotwordIntentPrefix(strings.TrimSpace(raw))
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Trim(trimmed, " \t\r\n.,!?;:")
	return strings.Join(strings.Fields(trimmed), " ")
}

func parseInlineCanvasNavigationIntent(text string) *SystemAction {
	normalized := normalizeCanvasNavigationCommandText(text)
	if normalized == "" {
		return nil
	}
	for _, pattern := range canvasPreviousArtifactPatterns {
		if pattern.MatchString(normalized) {
			return &SystemAction{Action: "navigate_canvas", Params: map[string]interface{}{"scope": "artifact", "direction": "previous"}}
		}
	}
	for _, pattern := range canvasNextArtifactPatterns {
		if pattern.MatchString(normalized) {
			return &SystemAction{Action: "navigate_canvas", Params: map[string]interface{}{"scope": "artifact", "direction": "next"}}
		}
	}
	for _, pattern := range canvasPreviousPagePatterns {
		if pattern.MatchString(normalized) {
			return &SystemAction{Action: "navigate_canvas", Params: map[string]interface{}{"scope": "page_or_artifact", "direction": "previous"}}
		}
	}
	for _, pattern := range canvasNextPagePatterns {
		if pattern.MatchString(normalized) {
			return &SystemAction{Action: "navigate_canvas", Params: map[string]interface{}{"scope": "page_or_artifact", "direction": "next"}}
		}
	}
	return nil
}

func systemActionNavigationScope(params map[string]interface{}) string {
	scope := strings.ToLower(strings.TrimSpace(systemActionStringParam(params, "scope")))
	switch scope {
	case "artifact", "page_or_artifact":
		return scope
	default:
		return "page_or_artifact"
	}
}

func systemActionNavigationDirection(params map[string]interface{}) string {
	direction := strings.ToLower(strings.TrimSpace(systemActionStringParam(params, "direction")))
	switch direction {
	case "next", "forward":
		return "next"
	case "previous", "prev", "back", "backward":
		return "previous"
	default:
		return ""
	}
}
