package roomstate

import (
	"strings"
	"testing"

	"github.com/krystophny/tabura/internal/store"
)

func TestDeriveCarriesEntitiesAndReconstructsTimeline(t *testing.T) {
	segments := []store.ParticipantSegment{
		{
			ID:          1,
			SessionID:   "psess-1",
			StartTS:     100,
			EndTS:       101,
			Speaker:     "Alice",
			Text:        "Review the Acme Cloud budget and API rollout.",
			CommittedAt: 102,
			Status:      "final",
		},
		{
			ID:          2,
			SessionID:   "psess-1",
			StartTS:     130,
			EndTS:       131,
			Speaker:     "Bob",
			Text:        "Bob will send Contoso follow-up notes after the meeting.",
			CommittedAt: 132,
			Status:      "final",
		},
	}
	events := []store.ParticipantEvent{
		{SessionID: "psess-1", EventType: "session_started", PayloadJSON: `{"reason":"manual"}`, CreatedAt: 99},
		{SessionID: "psess-1", SegmentID: 2, EventType: "assistant_triggered", PayloadJSON: `{"chat_session_id":"chat-1"}`, CreatedAt: 140},
		{SessionID: "psess-1", SegmentID: 2, EventType: "assistant_turn_completed", PayloadJSON: `{"chat_session_id":"chat-1"}`, CreatedAt: 150},
		{SessionID: "psess-1", EventType: "session_stopped", PayloadJSON: `{"reason":"manual"}`, CreatedAt: 160},
	}

	result := Derive(segments, events)

	for _, want := range []string{"Alice", "Bob", "Acme Cloud", "Contoso"} {
		if !contains(result.Entities, want) {
			t.Fatalf("entities = %#v, want %q", result.Entities, want)
		}
	}
	if result.UpdatedAt != 160 {
		t.Fatalf("updated_at = %d, want 160", result.UpdatedAt)
	}
	if len(result.TopicTimeline) != 6 {
		t.Fatalf("topic_timeline = %d, want 6", len(result.TopicTimeline))
	}
	first, ok := result.TopicTimeline[0].(map[string]any)
	if !ok {
		t.Fatalf("first timeline item type = %T, want map[string]any", result.TopicTimeline[0])
	}
	if got := strings.TrimSpace(first["topic"].(string)); got != "Session started" {
		t.Fatalf("first topic = %q, want Session started", got)
	}
	last, ok := result.TopicTimeline[len(result.TopicTimeline)-1].(map[string]any)
	if !ok {
		t.Fatalf("last timeline item type = %T, want map[string]any", result.TopicTimeline[len(result.TopicTimeline)-1])
	}
	if got := strings.TrimSpace(last["topic"].(string)); got != "Session stopped" {
		t.Fatalf("last topic = %q, want Session stopped", got)
	}
	if !strings.Contains(result.SummaryText, "Assistant response completed") {
		t.Fatalf("summary_text = %q, want assistant completion topic", result.SummaryText)
	}
	if !strings.Contains(result.SummaryText, "Acme Cloud") {
		t.Fatalf("summary_text = %q, want entity carry-forward", result.SummaryText)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
