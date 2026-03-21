package web

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func renderCalendarMarkdown(req calendarActionRequest, activeSphere string, events []calendarEventEntry, deadlines []calendarDeadlineEntry, warnings []string) string {
	switch req.View {
	case calendarViewWeek:
		return renderCalendarRangeMarkdown(req, activeSphere, events, deadlines, warnings, 7)
	case calendarViewAgenda:
		return renderCalendarRangeMarkdown(req, activeSphere, events, deadlines, warnings, 1)
	case calendarViewAvailability:
		return renderCalendarAvailabilityMarkdown(req, activeSphere, events, deadlines, warnings)
	default:
		return renderCalendarRangeMarkdown(req, activeSphere, events, deadlines, warnings, 1)
	}
}

func renderCalendarRangeMarkdown(req calendarActionRequest, activeSphere string, events []calendarEventEntry, deadlines []calendarDeadlineEntry, warnings []string, days int) string {
	start := req.Date.In(time.Local)
	var b strings.Builder
	title := "Calendar"
	switch req.View {
	case calendarViewWeek:
		title = "Calendar Week"
	case calendarViewAgenda:
		title = "Calendar Agenda"
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "- Active sphere: `%s`\n", activeSphere)
	fmt.Fprintf(&b, "- Range: %s to %s\n", start.Format("Monday, January 2, 2006"), start.AddDate(0, 0, days-1).Format("Monday, January 2, 2006"))
	if strings.TrimSpace(req.Query) != "" {
		fmt.Fprintf(&b, "- Filter: `%s`\n", req.Query)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(&b, "- Source warnings: %d\n", len(warnings))
	}
	b.WriteString("\n")
	for day := 0; day < days; day++ {
		current := start.AddDate(0, 0, day)
		dayEvents := eventsForDay(events, current)
		dayDeadlines := deadlinesForDay(deadlines, current)
		fmt.Fprintf(&b, "## %s\n\n", current.Format("Monday, January 2"))
		if len(dayEvents) == 0 && len(dayDeadlines) == 0 {
			b.WriteString("_No events or item deadlines._\n\n")
			continue
		}
		if len(dayEvents) > 0 {
			b.WriteString("### Events\n\n")
			for _, event := range dayEvents {
				fmt.Fprintf(&b, "- %s\n", renderCalendarEventLine(event, activeSphere))
			}
			b.WriteString("\n")
		}
		if len(dayDeadlines) > 0 {
			b.WriteString("### Item Deadlines\n\n")
			for _, deadline := range dayDeadlines {
				fmt.Fprintf(&b, "- %s\n", renderCalendarDeadlineLine(deadline, activeSphere))
			}
			b.WriteString("\n")
		}
	}
	if len(warnings) > 0 {
		b.WriteString("## Source Warnings\n\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func renderCalendarAvailabilityMarkdown(req calendarActionRequest, activeSphere string, events []calendarEventEntry, deadlines []calendarDeadlineEntry, warnings []string) string {
	dayStart := time.Date(req.Date.Year(), req.Date.Month(), req.Date.Day(), calendarAvailabilityFrom, 0, 0, 0, time.Local)
	dayEnd := time.Date(req.Date.Year(), req.Date.Month(), req.Date.Day(), calendarAvailabilityTo, 0, 0, 0, time.Local)
	freeSlots := computeCalendarAvailability(events, dayStart, dayEnd)

	var b strings.Builder
	fmt.Fprintf(&b, "# Availability for %s\n\n", req.Date.In(time.Local).Format("Monday, January 2, 2006"))
	fmt.Fprintf(&b, "- Active sphere: `%s`\n", activeSphere)
	fmt.Fprintf(&b, "- Window: %s to %s\n\n", dayStart.Format("15:04"), dayEnd.Format("15:04"))

	b.WriteString("## Free Slots\n\n")
	if len(freeSlots) == 0 {
		b.WriteString("_No free slots in the default workday window._\n\n")
	} else {
		for _, slot := range freeSlots {
			fmt.Fprintf(&b, "- %s to %s\n", slot[0].Format("15:04"), slot[1].Format("15:04"))
		}
		b.WriteString("\n")
	}

	dayEvents := eventsForDay(events, req.Date)
	if len(dayEvents) > 0 {
		b.WriteString("## Busy Blocks\n\n")
		for _, event := range dayEvents {
			fmt.Fprintf(&b, "- %s\n", renderCalendarEventLine(event, activeSphere))
		}
		b.WriteString("\n")
	}

	dayDeadlines := deadlinesForDay(deadlines, req.Date)
	if len(dayDeadlines) > 0 {
		b.WriteString("## Item Deadlines\n\n")
		for _, deadline := range dayDeadlines {
			fmt.Fprintf(&b, "- %s\n", renderCalendarDeadlineLine(deadline, activeSphere))
		}
		b.WriteString("\n")
	}
	if len(warnings) > 0 {
		b.WriteString("## Source Warnings\n\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func renderCalendarEventLine(event calendarEventEntry, activeSphere string) string {
	label := strings.TrimSpace(event.Summary)
	if label == "" {
		label = "(Untitled event)"
	}
	if !strings.EqualFold(strings.TrimSpace(event.Sphere), strings.TrimSpace(activeSphere)) {
		label = calendarBusyLabel
	}
	parts := []string{calendarTimeLabel(event.Start, event.End, event.AllDay), label}
	if strings.EqualFold(strings.TrimSpace(event.Sphere), strings.TrimSpace(activeSphere)) {
		if strings.TrimSpace(event.Location) != "" {
			parts = append(parts, "@ "+event.Location)
		}
		if len(event.Attendees) > 0 {
			parts = append(parts, "with "+strings.Join(event.Attendees, ", "))
		}
	}
	parts = append(parts, "["+firstNonEmptyCalendarValue(event.Source, event.Provider, "calendar")+"]")
	return strings.Join(parts, " ")
}

func renderCalendarDeadlineLine(entry calendarDeadlineEntry, activeSphere string) string {
	title := strings.TrimSpace(entry.Title)
	if title == "" {
		title = "(Untitled item)"
	}
	if !strings.EqualFold(strings.TrimSpace(entry.Sphere), strings.TrimSpace(activeSphere)) {
		title = fmt.Sprintf("%s item (%s)", entry.Kind, "other sphere")
	}
	parts := []string{entry.Kind, entry.When.Format("15:04"), title}
	if strings.EqualFold(strings.TrimSpace(entry.Sphere), strings.TrimSpace(activeSphere)) {
		if strings.TrimSpace(entry.Workspace) != "" {
			parts = append(parts, "["+entry.Workspace+"]")
		} else if strings.TrimSpace(entry.Project) != "" {
			parts = append(parts, "["+entry.Project+"]")
		}
	}
	return strings.Join(parts, " ")
}

func computeCalendarAvailability(events []calendarEventEntry, dayStart, dayEnd time.Time) [][2]time.Time {
	intervals := make([][2]time.Time, 0, len(events))
	for _, event := range events {
		start := event.Start
		end := event.End
		if event.AllDay {
			start = dayStart
			end = dayEnd
		}
		if end.Before(dayStart) || !start.Before(dayEnd) {
			continue
		}
		if start.Before(dayStart) {
			start = dayStart
		}
		if end.After(dayEnd) {
			end = dayEnd
		}
		if !start.Before(end) {
			continue
		}
		intervals = append(intervals, [2]time.Time{start, end})
	}
	if len(intervals) == 0 {
		return [][2]time.Time{{dayStart, dayEnd}}
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i][0].Equal(intervals[j][0]) {
			return intervals[i][1].Before(intervals[j][1])
		}
		return intervals[i][0].Before(intervals[j][0])
	})
	merged := make([][2]time.Time, 0, len(intervals))
	for _, interval := range intervals {
		if len(merged) == 0 {
			merged = append(merged, interval)
			continue
		}
		last := &merged[len(merged)-1]
		if interval[0].After(last[1]) {
			merged = append(merged, interval)
			continue
		}
		if interval[1].After(last[1]) {
			last[1] = interval[1]
		}
	}
	free := make([][2]time.Time, 0, len(merged)+1)
	cursor := dayStart
	for _, interval := range merged {
		if cursor.Before(interval[0]) {
			free = append(free, [2]time.Time{cursor, interval[0]})
		}
		if interval[1].After(cursor) {
			cursor = interval[1]
		}
	}
	if cursor.Before(dayEnd) {
		free = append(free, [2]time.Time{cursor, dayEnd})
	}
	return free
}

func calendarTimeRange(req calendarActionRequest) (time.Time, time.Time) {
	start := time.Date(req.Date.Year(), req.Date.Month(), req.Date.Day(), 0, 0, 0, 0, time.Local)
	days := 1
	switch req.View {
	case calendarViewWeek:
		days = 7
	case calendarViewAvailability:
		days = 1
	case calendarViewAgenda:
		days = 1
	}
	return start, start.AddDate(0, 0, days)
}

func matchesCalendarQuery(query string, event calendarEventEntry, extra string) bool {
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	if cleanQuery == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		event.Summary,
		event.Description,
		event.Location,
		strings.Join(event.Attendees, " "),
		event.Source,
		event.Provider,
		extra,
	}, " "))
	return strings.Contains(haystack, cleanQuery)
}

func deadlinesForDay(deadlines []calendarDeadlineEntry, day time.Time) []calendarDeadlineEntry {
	target := day.In(time.Local).Format("2006-01-02")
	out := make([]calendarDeadlineEntry, 0, len(deadlines))
	for _, deadline := range deadlines {
		if deadline.When.In(time.Local).Format("2006-01-02") == target {
			out = append(out, deadline)
		}
	}
	return out
}

func eventsForDay(events []calendarEventEntry, day time.Time) []calendarEventEntry {
	target := day.In(time.Local).Format("2006-01-02")
	out := make([]calendarEventEntry, 0, len(events))
	for _, event := range events {
		if event.Start.In(time.Local).Format("2006-01-02") == target {
			out = append(out, event)
		}
	}
	return out
}

func calendarArtifactTitle(req calendarActionRequest) string {
	base := "Calendar"
	switch req.View {
	case calendarViewWeek:
		base = "Calendar Week"
	case calendarViewAgenda:
		base = "Calendar Agenda"
	case calendarViewAvailability:
		base = "Availability"
	}
	return fmt.Sprintf("%s %s", base, req.Date.In(time.Local).Format("2006-01-02"))
}

func calendarArtifactMeta(req calendarActionRequest, activeSphere string, eventCount, deadlineCount int, warnings []string) string {
	payload := map[string]interface{}{
		"view":           req.View,
		"date":           req.Date.In(time.Local).Format("2006-01-02"),
		"query":          req.Query,
		"active_sphere":  activeSphere,
		"event_count":    eventCount,
		"deadline_count": deadlineCount,
		"warnings":       warnings,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func calendarTimeLabel(start, end time.Time, allDay bool) string {
	if allDay {
		return "All day"
	}
	if end.IsZero() || !start.Before(end) {
		return start.Format("15:04")
	}
	return start.Format("15:04") + "-" + end.Format("15:04")
}

func calendarWorkspaceName(a *App, workspaceID *int64, cache map[int64]string) string {
	if a == nil || workspaceID == nil {
		return ""
	}
	if cached, ok := cache[*workspaceID]; ok {
		return cached
	}
	workspace, err := a.store.GetWorkspace(*workspaceID)
	if err != nil {
		cache[*workspaceID] = ""
		return ""
	}
	cache[*workspaceID] = workspace.Name
	return workspace.Name
}

func calendarProjectName(a *App, workspaceID *int64, cache map[int64]string) string {
	if a == nil || workspaceID == nil {
		return ""
	}
	if cached, ok := cache[*workspaceID]; ok {
		return cached
	}
	workspace, err := a.store.GetWorkspace(*workspaceID)
	if err != nil {
		cache[*workspaceID] = ""
		return ""
	}
	cache[*workspaceID] = workspace.Name
	return workspace.Name
}

func calendarDeadlineSearchText(entry calendarDeadlineEntry) string {
	return strings.Join([]string{entry.Title, entry.Workspace, entry.Project, entry.Kind}, " ")
}
