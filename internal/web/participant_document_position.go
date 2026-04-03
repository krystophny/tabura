package web

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/krystophny/slopshell/internal/store"
)

const participantDocumentPositionEventType = "document_position_changed"

type participantDocumentPositionPayload struct {
	Gesture      string  `json:"gesture,omitempty"`
	ArtifactTitle string `json:"artifact_title,omitempty"`
	ArtifactPath  string `json:"artifact_path,omitempty"`
	View         string  `json:"view,omitempty"`
	Element      string  `json:"element,omitempty"`
	Page         int     `json:"page,omitempty"`
	Line         int     `json:"line,omitempty"`
	RelativeX    float64 `json:"relative_x,omitempty"`
	RelativeY    float64 `json:"relative_y,omitempty"`
	SelectedText string  `json:"selected_text,omitempty"`
}

type participantTranscriptEntry struct {
	Kind      string                            `json:"kind"`
	StartTS   int64                             `json:"start_ts"`
	EndTS     int64                             `json:"end_ts,omitempty"`
	Speaker   string                            `json:"speaker,omitempty"`
	Text      string                            `json:"text,omitempty"`
	SegmentID int64                             `json:"segment_id,omitempty"`
	EventID   int64                             `json:"event_id,omitempty"`
	EventType string                            `json:"event_type,omitempty"`
	Document  *participantDocumentPositionPayload `json:"document,omitempty"`
}

func buildParticipantDocumentPositionPayload(cursor *chatCursorContext, gesture string) *participantDocumentPositionPayload {
	cursor = normalizeChatCursorContext(cursor)
	if cursor == nil {
		return nil
	}
	payload := &participantDocumentPositionPayload{
		Gesture:       strings.TrimSpace(gesture),
		ArtifactTitle: strings.TrimSpace(cursor.Title),
		ArtifactPath:  strings.TrimSpace(cursor.Path),
		View:          strings.TrimSpace(cursor.View),
		Element:       strings.TrimSpace(cursor.Element),
		Page:          cursor.Page,
		Line:          cursor.Line,
		RelativeX:     cursor.RelativeX,
		RelativeY:     cursor.RelativeY,
		SelectedText:  strings.TrimSpace(cursor.SelectedText),
	}
	if payload.Gesture == "" {
		payload.Gesture = "tap"
	}
	if payload.ArtifactTitle == "" &&
		payload.ArtifactPath == "" &&
		payload.View == "" &&
		payload.Element == "" &&
		payload.Page == 0 &&
		payload.Line == 0 &&
		payload.RelativeX == 0 &&
		payload.RelativeY == 0 &&
		payload.SelectedText == "" {
		return nil
	}
	return payload
}

func (a *App) recordParticipantDocumentPositionIfActive(conn *chatWSConn, cursor *chatCursorContext, gesture string) {
	if a == nil || a.store == nil || conn == nil {
		return
	}
	payload := buildParticipantDocumentPositionPayload(cursor, gesture)
	if payload == nil {
		return
	}
	conn.participantMu.Lock()
	sessionID := strings.TrimSpace(conn.participantSessionID)
	active := conn.participantActive && sessionID != ""
	conn.participantMu.Unlock()
	if !active {
		return
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = a.store.AddParticipantEvent(sessionID, 0, participantDocumentPositionEventType, string(encoded))
	a.syncProjectCompanionArtifactsBySessionID(sessionID)
}

func parseParticipantDocumentPositionPayload(raw string) *participantDocumentPositionPayload {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil
	}
	var payload participantDocumentPositionPayload
	if err := json.Unmarshal([]byte(clean), &payload); err != nil {
		return nil
	}
	return buildParticipantDocumentPositionPayload(&chatCursorContext{
		View:         payload.View,
		Element:      payload.Element,
		Title:        payload.ArtifactTitle,
		Page:         payload.Page,
		Line:         payload.Line,
		RelativeX:    payload.RelativeX,
		RelativeY:    payload.RelativeY,
		SelectedText: payload.SelectedText,
		Path:         payload.ArtifactPath,
	}, payload.Gesture)
}

func buildParticipantTranscriptEntries(segments []store.ParticipantSegment, events []store.ParticipantEvent) []participantTranscriptEntry {
	entries := make([]participantTranscriptEntry, 0, len(segments)+len(events))
	for _, seg := range segments {
		entries = append(entries, participantTranscriptEntry{
			Kind:      "segment",
			StartTS:   seg.StartTS,
			EndTS:     seg.EndTS,
			Speaker:   seg.Speaker,
			Text:      seg.Text,
			SegmentID: seg.ID,
		})
	}
	for _, event := range events {
		if strings.TrimSpace(event.EventType) != participantDocumentPositionEventType {
			continue
		}
		payload := parseParticipantDocumentPositionPayload(event.PayloadJSON)
		if payload == nil {
			continue
		}
		entries = append(entries, participantTranscriptEntry{
			Kind:      "document_position",
			StartTS:   event.CreatedAt,
			EventID:   event.ID,
			EventType: event.EventType,
			Document:  payload,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].StartTS == entries[j].StartTS {
			if entries[i].Kind == entries[j].Kind {
				return entries[i].EventID < entries[j].EventID
			}
			return entries[i].Kind == "document_position"
		}
		return entries[i].StartTS < entries[j].StartTS
	})
	return entries
}

func describeParticipantDocumentPosition(payload *participantDocumentPositionPayload) string {
	if payload == nil {
		return ""
	}
	parts := make([]string, 0, 6)
	if payload.ArtifactTitle != "" {
		parts = append(parts, fmt.Sprintf("document %q", payload.ArtifactTitle))
	} else if payload.ArtifactPath != "" {
		parts = append(parts, fmt.Sprintf("document %q", payload.ArtifactPath))
	}
	if payload.Page > 0 {
		parts = append(parts, fmt.Sprintf("page %d", payload.Page))
	}
	if payload.Line > 0 {
		parts = append(parts, fmt.Sprintf("line %d", payload.Line))
	}
	if payload.RelativeX > 0 || payload.RelativeY > 0 {
		parts = append(parts, fmt.Sprintf("point %.0f%%, %.0f%%", payload.RelativeX*100, payload.RelativeY*100))
	}
	if payload.SelectedText != "" {
		parts = append(parts, "selection "+quotePromptText(payload.SelectedText, 120))
	}
	if len(parts) == 0 {
		return "document position updated"
	}
	return "Document position updated: " + strings.Join(parts, ", ")
}
