package roomstate

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/krystophny/tabura/internal/store"
)

type Result struct {
	SummaryText   string
	Entities      []string
	TopicTimeline []any
	UpdatedAt     int64
}

type timelineItem struct {
	At        int64
	Type      string
	EventType string
	Speaker   string
	Topic     string
	Detail    string
	SegmentID int64
}

var (
	entityPattern          = regexp.MustCompile(`\b(?:[A-Z][a-z0-9]+|[A-Z]{2,})(?:\s+(?:[A-Z][a-z0-9]+|[A-Z]{2,}))*\b`)
	spacePattern           = regexp.MustCompile(`\s+`)
	punctuationTrimPattern = regexp.MustCompile(`^[^A-Za-z0-9]+|[^A-Za-z0-9]+$`)
)

var ignoredEntities = map[string]struct{}{
	"a":         {},
	"an":        {},
	"and":       {},
	"assistant": {},
	"hello":     {},
	"i":         {},
	"meeting":   {},
	"please":    {},
	"session":   {},
	"speaker":   {},
	"tabura":    {},
	"task":      {},
	"thanks":    {},
	"the":       {},
	"we":        {},
}

func Derive(segments []store.ParticipantSegment, events []store.ParticipantEvent) Result {
	entitySet := map[string]struct{}{}
	timeline := make([]timelineItem, 0, len(segments)+len(events))
	updatedAt := int64(0)

	for _, seg := range segments {
		if ts := maxInt64(seg.CommittedAt, maxInt64(seg.EndTS, seg.StartTS)); ts > updatedAt {
			updatedAt = ts
		}
		text := normalizeSpace(seg.Text)
		speaker := cleanEntityName(seg.Speaker)
		if speaker != "" {
			entitySet[speaker] = struct{}{}
		}
		if text == "" {
			continue
		}
		addEntities(entitySet, speaker)
		addEntities(entitySet, extractEntities(text)...)
		topic := summarizeTextTopic(text)
		if topic == "" {
			continue
		}
		timeline = append(timeline, timelineItem{
			At:        maxInt64(seg.StartTS, seg.CommittedAt),
			Type:      "segment",
			Speaker:   speaker,
			Topic:     topic,
			Detail:    text,
			SegmentID: seg.ID,
		})
	}

	for _, event := range events {
		if event.CreatedAt > updatedAt {
			updatedAt = event.CreatedAt
		}
		addEntities(entitySet, extractEntitiesFromPayload(event.PayloadJSON)...)
		item := timelineFromEvent(event)
		if item.Topic == "" && item.Detail == "" {
			continue
		}
		timeline = append(timeline, item)
	}

	sort.SliceStable(timeline, func(i, j int) bool {
		if timeline[i].At == timeline[j].At {
			if timeline[i].SegmentID == timeline[j].SegmentID {
				return timeline[i].EventType < timeline[j].EventType
			}
			return timeline[i].SegmentID < timeline[j].SegmentID
		}
		return timeline[i].At < timeline[j].At
	})
	timeline = dedupeTimeline(timeline)

	entities := make([]string, 0, len(entitySet))
	for entity := range entitySet {
		entities = append(entities, entity)
	}
	sort.Strings(entities)

	resultTimeline := make([]any, 0, len(timeline))
	for _, item := range timeline {
		entry := map[string]any{
			"at":    item.At,
			"type":  item.Type,
			"topic": item.Topic,
		}
		if item.EventType != "" {
			entry["event_type"] = item.EventType
		}
		if item.Speaker != "" {
			entry["speaker"] = item.Speaker
		}
		if item.Detail != "" {
			entry["detail"] = item.Detail
		}
		if item.SegmentID != 0 {
			entry["segment_id"] = item.SegmentID
		}
		resultTimeline = append(resultTimeline, entry)
	}

	return Result{
		SummaryText:   buildSummary(entities, timeline),
		Entities:      entities,
		TopicTimeline: resultTimeline,
		UpdatedAt:     updatedAt,
	}
}

func buildSummary(entities []string, timeline []timelineItem) string {
	topics := latestDistinctTopics(timeline, 3)
	parts := make([]string, 0, 2)
	if len(topics) > 0 {
		parts = append(parts, fmt.Sprintf("Timeline: %s.", strings.Join(topics, "; ")))
	}
	if len(entities) > 0 {
		parts = append(parts, fmt.Sprintf("Entities: %s.", strings.Join(firstN(entities, 4), ", ")))
	}
	return strings.Join(parts, " ")
}

func latestDistinctTopics(timeline []timelineItem, n int) []string {
	if n <= 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, n)
	for i := len(timeline) - 1; i >= 0 && len(out) < n; i-- {
		topic := normalizeSpace(timeline[i].Topic)
		if topic == "" {
			continue
		}
		key := strings.ToLower(topic)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, topic)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func firstN(values []string, n int) []string {
	if n <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) <= n {
		return values
	}
	return values[:n]
}

func dedupeTimeline(items []timelineItem) []timelineItem {
	if len(items) == 0 {
		return items
	}
	out := make([]timelineItem, 0, len(items))
	var prev timelineItem
	for i, item := range items {
		if i == 0 || !sameTimelineItem(prev, item) {
			out = append(out, item)
			prev = item
		}
	}
	return out
}

func sameTimelineItem(a, b timelineItem) bool {
	return a.At == b.At &&
		a.Type == b.Type &&
		a.EventType == b.EventType &&
		a.Speaker == b.Speaker &&
		a.Topic == b.Topic &&
		a.Detail == b.Detail &&
		a.SegmentID == b.SegmentID
}

func timelineFromEvent(event store.ParticipantEvent) timelineItem {
	payload := parsePayload(event.PayloadJSON)
	topic := ""
	detail := ""
	switch strings.TrimSpace(event.EventType) {
	case "segment_committed":
		topic = ""
	case "session_started":
		topic = "Session started"
		detail = payloadString(payload, "reason")
	case "session_stopped":
		topic = "Session stopped"
		detail = payloadString(payload, "reason")
	case "assistant_triggered":
		topic = "Assistant response triggered"
	case "assistant_turn_completed":
		topic = "Assistant response completed"
	case "assistant_turn_cancelled":
		topic = "Assistant response cancelled"
	case "assistant_turn_failed":
		topic = "Assistant response failed"
	case "assistant_interrupted":
		topic = "Assistant response interrupted"
	default:
		topic = payloadString(payload, "topic")
		if topic == "" {
			topic = summarizeTextTopic(payloadString(payload, "text"))
		}
	}
	return timelineItem{
		At:        event.CreatedAt,
		Type:      "event",
		EventType: strings.TrimSpace(event.EventType),
		Topic:     normalizeSpace(topic),
		Detail:    normalizeSpace(detail),
		SegmentID: event.SegmentID,
	}
}

func extractEntitiesFromPayload(raw string) []string {
	payload := parsePayload(raw)
	if payload == nil {
		return nil
	}
	out := []string{}
	collectPayloadEntities(&out, "", payload)
	return out
}

func collectPayloadEntities(out *[]string, key string, value any) {
	switch v := value.(type) {
	case map[string]any:
		for childKey, childValue := range v {
			collectPayloadEntities(out, childKey, childValue)
		}
	case []any:
		for _, childValue := range v {
			collectPayloadEntities(out, key, childValue)
		}
	case string:
		clean := normalizeSpace(v)
		if clean == "" {
			return
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "entity", "entities", "name", "participant", "participants", "person", "people", "project", "speaker", "subject", "team":
			*out = append(*out, cleanEntityName(clean))
		case "text", "topic", "title":
			*out = append(*out, extractEntities(clean)...)
		}
	}
}

func parsePayload(raw string) map[string]any {
	clean := strings.TrimSpace(raw)
	if clean == "" || clean == "{}" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(clean), &payload); err != nil {
		return nil
	}
	return payload
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value := normalizeSpace(fmt.Sprint(payload[key]))
	if value == "<nil>" {
		return ""
	}
	return value
}

func addEntities(set map[string]struct{}, entities ...string) {
	for _, entity := range entities {
		clean := cleanEntityName(entity)
		if clean == "" {
			continue
		}
		set[clean] = struct{}{}
	}
}

func extractEntities(text string) []string {
	matches := entityPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		clean := cleanEntityName(match)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func cleanEntityName(raw string) string {
	clean := punctuationTrimPattern.ReplaceAllString(normalizeSpace(raw), "")
	if clean == "" {
		return ""
	}
	if _, ignored := ignoredEntities[strings.ToLower(clean)]; ignored {
		return ""
	}
	return clean
}

func summarizeTextTopic(text string) string {
	clean := normalizeSpace(text)
	if clean == "" {
		return ""
	}
	clean = strings.TrimPrefix(clean, "Tabura, ")
	clean = strings.TrimPrefix(clean, "tabura, ")
	clean = strings.TrimPrefix(clean, "Assistant, ")
	clean = strings.TrimPrefix(clean, "assistant, ")
	clean = strings.TrimRight(clean, ".!?")
	words := strings.Fields(clean)
	if len(words) > 10 {
		words = words[:10]
	}
	return strings.Join(words, " ")
}

func normalizeSpace(raw string) string {
	return strings.TrimSpace(spacePattern.ReplaceAllString(raw, " "))
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
