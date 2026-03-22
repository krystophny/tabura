package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	meetingDocumentFollowTimeout       = 1200 * time.Millisecond
	meetingDocumentFollowResponseLimit = 64 * 1024
)

type meetingDocumentFollowUnit struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

type meetingDocumentFollowRequest struct {
	Transcript       string                    `json:"transcript"`
	ArtifactKind     string                    `json:"artifact_kind"`
	ArtifactTitle    string                    `json:"artifact_title"`
	ArtifactPath     string                    `json:"artifact_path"`
	Current          *meetingDocumentFollowUnit `json:"current"`
	Previous         *meetingDocumentFollowUnit `json:"previous,omitempty"`
	Next             *meetingDocumentFollowUnit `json:"next,omitempty"`
	SnapshotDataURL  string                    `json:"snapshot_data_url,omitempty"`
}

type meetingDocumentFollowResponse struct {
	OK         bool   `json:"ok"`
	Action     string `json:"action"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason,omitempty"`
	Source     string `json:"source,omitempty"`
}

type meetingDocumentFollowLLMResponse struct {
	Action     string `json:"action"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
}

var (
	meetingDocumentNextPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:next|following)\s+(?:slide|page|folie|seite|section|abschnitt)\b`),
		regexp.MustCompile(`(?i)\bon\s+the\s+next\s+(?:slide|page|section)\b`),
		regexp.MustCompile(`(?i)\blet(?:'s| us)\s+move\s+on\b`),
		regexp.MustCompile(`(?i)\bmove\s+on\s+to\s+the\s+next\b`),
		regexp.MustCompile(`(?i)\b(?:zur|auf\s+der)\s+n[aä]chsten\s+(?:folie|seite)\b`),
		regexp.MustCompile(`(?i)\bgehen\s+wir\s+weiter\b`),
		regexp.MustCompile(`(?i)\bweiter\s+zur\s+n[aä]chsten\s+(?:folie|seite)\b`),
	}
	meetingDocumentPreviousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:previous|prior|last)\s+(?:slide|page|folie|seite|section|abschnitt)\b`),
		regexp.MustCompile(`(?i)\bgo\s+back(?:\s+one)?\b`),
		regexp.MustCompile(`(?i)\bback\s+one\s+(?:slide|page)\b`),
		regexp.MustCompile(`(?i)\b(?:zur[uü]ck|noch\s+einmal\s+zur[uü]ck)\b`),
		regexp.MustCompile(`(?i)\beine\s+(?:folie|seite)\s+zur[uü]ck\b`),
		regexp.MustCompile(`(?i)\b(?:vorherige|vorige)\s+(?:folie|seite)\b`),
	}
)

func normalizeMeetingDocumentFollowUnit(raw *meetingDocumentFollowUnit) *meetingDocumentFollowUnit {
	if raw == nil {
		return nil
	}
	unit := &meetingDocumentFollowUnit{
		ID:    strings.TrimSpace(raw.ID),
		Label: strings.Join(strings.Fields(strings.TrimSpace(raw.Label)), " "),
		Text:  strings.Join(strings.Fields(strings.TrimSpace(raw.Text)), " "),
	}
	if unit.ID == "" && unit.Label == "" && unit.Text == "" {
		return nil
	}
	return unit
}

func normalizeMeetingDocumentFollowAction(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "next":
		return "next"
	case "previous", "prev":
		return "previous"
	default:
		return "stay"
	}
}

func normalizeMeetingDocumentFollowConfidence(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return ""
	}
}

func normalizeMeetingDocumentFollowRequest(raw meetingDocumentFollowRequest) meetingDocumentFollowRequest {
	return meetingDocumentFollowRequest{
		Transcript:      strings.Join(strings.Fields(strings.TrimSpace(raw.Transcript)), " "),
		ArtifactKind:    strings.ToLower(strings.TrimSpace(raw.ArtifactKind)),
		ArtifactTitle:   strings.TrimSpace(raw.ArtifactTitle),
		ArtifactPath:    strings.TrimSpace(raw.ArtifactPath),
		Current:         normalizeMeetingDocumentFollowUnit(raw.Current),
		Previous:        normalizeMeetingDocumentFollowUnit(raw.Previous),
		Next:            normalizeMeetingDocumentFollowUnit(raw.Next),
		SnapshotDataURL: normalizeChatVisualDataURL(raw.SnapshotDataURL),
	}
}

func meetingDocumentFollowPromptUnit(label string, unit *meetingDocumentFollowUnit) string {
	if unit == nil {
		return label + ": (none)"
	}
	parts := []string{label + ":"}
	if unit.Label != "" {
		parts = append(parts, unit.Label)
	}
	if unit.Text != "" {
		parts = append(parts, quotePromptText(unit.Text, 600))
	}
	return strings.Join(parts, " ")
}

func buildMeetingDocumentFollowPrompt(req meetingDocumentFollowRequest) string {
	lines := []string{
		"Transcript: " + req.Transcript,
	}
	if req.ArtifactKind != "" || req.ArtifactTitle != "" || req.ArtifactPath != "" {
		lines = append(lines, fmt.Sprintf(
			"Artifact: kind=%q title=%q path=%q",
			req.ArtifactKind,
			req.ArtifactTitle,
			req.ArtifactPath,
		))
	}
	lines = append(lines, meetingDocumentFollowPromptUnit("Current unit", req.Current))
	lines = append(lines, meetingDocumentFollowPromptUnit("Previous unit", req.Previous))
	lines = append(lines, meetingDocumentFollowPromptUnit("Next unit", req.Next))
	lines = append(lines, "Return only one action: stay, next, or previous.")
	return strings.Join(lines, "\n")
}

func meetingDocumentFollowHeuristicAction(req meetingDocumentFollowRequest) (string, string) {
	transcript := strings.ToLower(strings.TrimSpace(req.Transcript))
	if transcript == "" {
		return "stay", ""
	}
	for _, pattern := range meetingDocumentPreviousPatterns {
		if pattern.MatchString(transcript) {
			return "previous", "explicit_previous_phrase"
		}
	}
	for _, pattern := range meetingDocumentNextPatterns {
		if pattern.MatchString(transcript) {
			return "next", "explicit_next_phrase"
		}
	}
	return "stay", ""
}

func buildMeetingDocumentFollowSystemPrompt() string {
	return strings.TrimSpace(`
You decide whether a live meeting/presentation should move one visible document unit.

Return JSON only:
{"action":"stay|next|previous","confidence":"high|medium|low","reason":"short_reason"}

Rules:
- Never jump more than one unit.
- Default to "stay" unless the transcript clearly indicates moving.
- Prefer "previous" only for explicit go-back language.
- Use "next" when the transcript explicitly asks to continue or when the transcript clearly fits the next unit better than the current one.
- If the adjacent unit required by the action does not exist, return "stay".
- The transcript may be in English or German.
- Treat phrases like "next slide", "next page", "zur nächsten Folie", "gehen wir weiter", "go back", "noch einmal zurück", and "eine Folie zurück" as strong cues.
- Be conservative. If uncertain, stay.
`)
}

func (a *App) requestMeetingDocumentFollowDecision(ctx context.Context, req meetingDocumentFollowRequest) (meetingDocumentFollowResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(a.assistantLLMBaseURL()), "/")
	if baseURL == "" {
		return meetingDocumentFollowResponse{}, nil
	}
	prompt := buildMeetingDocumentFollowPrompt(req)
	visual := (*chatVisualAttachment)(nil)
	if req.SnapshotDataURL != "" {
		visual = &chatVisualAttachment{DataURL: req.SnapshotDataURL}
	}
	requestBody, _ := json.Marshal(map[string]any{
		"model":       a.localAssistantLLMModel(),
		"temperature": 0,
		"max_tokens":  180,
		"response_format": map[string]any{
			"type": "json_object",
		},
		"chat_template_kwargs": map[string]any{
			"enable_thinking": false,
		},
		"messages": []map[string]any{
			{"role": "system", "content": buildMeetingDocumentFollowSystemPrompt()},
			{"role": "user", "content": buildLocalAssistantUserContent(prompt, visual)},
		},
	})
	requestCtx, cancel := context.WithTimeout(ctx, meetingDocumentFollowTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		baseURL+"/v1/chat/completions",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return meetingDocumentFollowResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return meetingDocumentFollowResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, meetingDocumentFollowResponseLimit))
		return meetingDocumentFollowResponse{}, fmt.Errorf("meeting document follow llm HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload localIntentLLMChatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, meetingDocumentFollowResponseLimit)).Decode(&payload); err != nil {
		return meetingDocumentFollowResponse{}, err
	}
	if len(payload.Choices) == 0 {
		return meetingDocumentFollowResponse{}, nil
	}
	content := strings.TrimSpace(stripCodeFence(payload.Choices[0].Message.Content))
	if content == "" {
		return meetingDocumentFollowResponse{}, nil
	}
	var result meetingDocumentFollowLLMResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return meetingDocumentFollowResponse{}, err
	}
	action := normalizeMeetingDocumentFollowAction(result.Action)
	return meetingDocumentFollowResponse{
		OK:         true,
		Action:     action,
		Confidence: normalizeMeetingDocumentFollowConfidence(result.Confidence),
		Reason:     strings.TrimSpace(result.Reason),
		Source:     "llm",
	}, nil
}

func (a *App) decideMeetingDocumentFollow(ctx context.Context, raw meetingDocumentFollowRequest) (meetingDocumentFollowResponse, error) {
	req := normalizeMeetingDocumentFollowRequest(raw)
	if req.Transcript == "" || req.Current == nil {
		return meetingDocumentFollowResponse{
			OK:     true,
			Action: "stay",
			Source: "validation",
			Reason: "missing_transcript_or_current_unit",
		}, nil
	}
	if action, reason := meetingDocumentFollowHeuristicAction(req); reason != "" {
		return meetingDocumentFollowResponse{
			OK:         true,
			Action:     action,
			Confidence: "high",
			Reason:     reason,
			Source:     "heuristic",
		}, nil
	}
	response, err := a.requestMeetingDocumentFollowDecision(ctx, req)
	if err != nil {
		return meetingDocumentFollowResponse{
			OK:     true,
			Action: "stay",
			Reason: err.Error(),
			Source: "llm_error",
		}, nil
	}
	if response.Action == "" {
		response.Action = "stay"
	}
	if !response.OK {
		response.OK = true
	}
	if response.Source == "" {
		response.Source = "llm"
	}
	return response, nil
}

func (a *App) handleMeetingDocumentFollowDecide(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		http.Error(w, "app unavailable", http.StatusInternalServerError)
		return
	}
	var req meetingDocumentFollowRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	response, err := a.decideMeetingDocumentFollow(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, response)
}
