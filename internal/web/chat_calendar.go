package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	tabcalendar "github.com/sloppy-org/slopshell/internal/calendar"
	"github.com/sloppy-org/slopshell/internal/ics"
	"github.com/sloppy-org/slopshell/internal/providerdata"
	"github.com/sloppy-org/slopshell/internal/store"
)

const (
	calendarViewDay          = "day"
	calendarViewWeek         = "week"
	calendarViewAgenda       = "agenda"
	calendarViewAvailability = "availability"
	calendarBusyLabel        = "Busy (other sphere)"
	calendarAvailabilityFrom = 8
	calendarAvailabilityTo   = 18
	calendarCreateDefaultMin = 60
)

var (
	calendarForPattern     = regexp.MustCompile(`(?i)^\s*(?:show|display|open)\s+(?:my\s+)?calendar\s+for\s+(.+?)\s*$`)
	calendarCreatePattern  = regexp.MustCompile(`(?i)^\s*(?:please\s+|bitte\s+)?(?:create|make|schedule|add|erstelle|mach|lege(?:\s+mir)?\s+an|trag(?:\s+mir)?\s+ein)\s+(?:ein(?:en|e)?\s+|an?\s+)?(?:calendar\s+event|appointment|meeting|event|termin|kalendereintrag)(?:\s+in\s+meinem\s+kalender)?\s+(?:for|für)\s+([0-9]{1,2}\.[0-9]{1,2}(?:\.(?:[0-9]{2,4})?)?)\s+(?:at|um)\s+([0-9]{1,2})(?::([0-9]{2}))?\s*(?:uhr)?\s+(.+?)\s*$`)
	calendarTokenSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

type googleCalendarClient interface {
	ListCalendars(ctx context.Context) ([]providerdata.Calendar, error)
	GetEvents(ctx context.Context, opts tabcalendar.GetEventsOptions) ([]providerdata.Event, error)
	CreateEvent(ctx context.Context, opts tabcalendar.CreateEventOptions) (providerdata.Event, error)
	UpdateEvent(ctx context.Context, opts tabcalendar.UpdateEventOptions) (providerdata.Event, error)
	DeleteEvent(ctx context.Context, calendarID, eventID string) error
}

type icsCalendarClient interface {
	ListCalendars() []providerdata.Calendar
	GetEvents(calendarName string, timeMin, timeMax time.Time) ([]ics.ICSEvent, error)
}

type calendarActionRequest struct {
	View  string
	Date  time.Time
	Query string
}

type calendarCreateRequest struct {
	CalendarID  string
	Summary     string
	Description string
	Location    string
	Attendees   []string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

type calendarEventEntry struct {
	Summary     string
	Description string
	Location    string
	Attendees   []string
	Source      string
	Provider    string
	Sphere      string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

type calendarDeadlineEntry struct {
	Title     string
	Sphere    string
	Kind      string
	When      time.Time
	Workspace string
	Project   string
}

func parseInlineCalendarIntent(text string, now time.Time) *SystemAction {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if action := parseInlineCalendarCreateIntent(trimmed, now); action != nil {
		return action
	}
	normalized := normalizeItemCommandText(trimmed)
	switch normalized {
	case "show calendar", "show my calendar", "show my schedule", "show schedule":
		return &SystemAction{
			Action: "show_calendar",
			Params: map[string]interface{}{"view": calendarViewDay},
		}
	case "what's today", "whats today", "what is today":
		return &SystemAction{
			Action: "show_calendar",
			Params: map[string]interface{}{
				"view": calendarViewAgenda,
				"date": now.In(time.Local).Format("2006-01-02"),
			},
		}
	case "what's this week", "whats this week", "what is this week":
		return &SystemAction{
			Action: "show_calendar",
			Params: map[string]interface{}{
				"view": calendarViewWeek,
				"date": now.In(time.Local).Format("2006-01-02"),
			},
		}
	case "when am i free tomorrow", "when am i available tomorrow":
		return &SystemAction{
			Action: "show_calendar",
			Params: map[string]interface{}{
				"view": calendarViewAvailability,
				"date": now.In(time.Local).AddDate(0, 0, 1).Format("2006-01-02"),
			},
		}
	}
	if match := calendarForPattern.FindStringSubmatch(trimmed); len(match) == 2 {
		query := strings.TrimSpace(match[1])
		if query != "" {
			return &SystemAction{
				Action: "show_calendar",
				Params: map[string]interface{}{
					"view":  calendarViewDay,
					"query": query,
				},
			}
		}
	}
	return nil
}

func calendarActionFailurePrefix(string) string {
	return "I couldn't build the calendar view: "
}

func calendarCreateActionFailurePrefix(string) string {
	return "I couldn't create the calendar event: "
}

func (a *App) executeCalendarAction(session store.ChatSession, action *SystemAction) (string, map[string]interface{}, error) {
	if a == nil || action == nil {
		return "", nil, fmt.Errorf("calendar action is required")
	}
	req, err := a.parseCalendarActionRequest(action)
	if err != nil {
		return "", nil, err
	}
	activeSphere, err := a.store.ActiveSphere()
	if err != nil || strings.TrimSpace(activeSphere) == "" {
		activeSphere = store.SpherePrivate
	}

	targetProject, err := a.systemActionTargetWorkspace(session)
	if err != nil {
		return "", nil, err
	}
	cwd := ""
	if workspace, workspaceErr := a.store.ActiveWorkspace(); workspaceErr == nil {
		cwd = strings.TrimSpace(workspace.DirPath)
	}
	if cwd == "" {
		cwd = strings.TrimSpace(a.systemActionTargetCWD(session, targetProject, nil))
	}
	if cwd == "" {
		return "", nil, fmt.Errorf("calendar view cwd is not available")
	}

	events, warnings, err := a.collectCalendarEvents(context.Background(), req, activeSphere)
	if err != nil {
		return "", nil, err
	}
	deadlines, err := a.collectCalendarDeadlines(req)
	if err != nil {
		return "", nil, err
	}
	content := renderCalendarMarkdown(req, activeSphere, events, deadlines, warnings)

	pathSeed := []string{req.Date.In(time.Local).Format("2006-01-02"), req.View}
	if strings.TrimSpace(req.Query) != "" {
		pathSeed = append(pathSeed, sanitizeCalendarFileToken(req.Query))
	}
	relativePath := filepath.ToSlash(filepath.Join(".slopshell", "artifacts", "calendar", strings.Join(pathSeed, "-")+".md"))
	absPath, canvasTitle, err := resolveCanvasFilePath(cwd, relativePath)
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", nil, err
	}

	artifactTitle := calendarArtifactTitle(req)
	metaJSON := calendarArtifactMeta(req, activeSphere, len(events), len(deadlines), warnings)
	artifact, err := a.store.CreateArtifact(store.ArtifactKind("calendar_view"), &absPath, nil, &artifactTitle, &metaJSON)
	if err != nil {
		return "", nil, err
	}
	if workspace, workspaceErr := a.store.ActiveWorkspace(); workspaceErr == nil {
		_ = a.store.LinkArtifactToWorkspace(workspace.ID, artifact.ID)
	}

	canvasSessionID := strings.TrimSpace(a.canvasSessionIDForWorkspace(targetProject))
	if canvasSessionID == "" {
		return "", nil, fmt.Errorf("canvas session is not available")
	}
	ep, ok := a.tunnels.getEndpoint(canvasSessionID)
	if !ok {
		return "", nil, fmt.Errorf("no active MCP tunnel for project %q", targetProject.Name)
	}
	if _, err := a.mcpToolsCall(ep, "canvas_artifact_show", map[string]interface{}{
		"session_id":       canvasSessionID,
		"kind":             "text",
		"title":            canvasTitle,
		"markdown_or_text": content,
	}); err != nil {
		return "", nil, err
	}
	a.markWorkspaceOutput(targetProject.WorkspacePath)

	return fmt.Sprintf("Opened %s on canvas.", artifactTitle), map[string]interface{}{
		"type":           "show_calendar",
		"artifact_id":    artifact.ID,
		"path":           canvasTitle,
		"view":           req.View,
		"date":           req.Date.In(time.Local).Format("2006-01-02"),
		"query":          req.Query,
		"event_count":    len(events),
		"deadline_count": len(deadlines),
		"warnings":       warnings,
	}, nil
}

func (a *App) parseCalendarActionRequest(action *SystemAction) (calendarActionRequest, error) {
	now := time.Now()
	if a != nil && a.calendarNow != nil {
		now = a.calendarNow()
	}
	req := calendarActionRequest{
		View:  strings.ToLower(calendarOptionalParam(action.Params, "view")),
		Date:  now.In(time.Local),
		Query: calendarOptionalParam(action.Params, "query"),
	}
	switch req.View {
	case "", calendarViewDay:
		req.View = calendarViewDay
	case calendarViewWeek, calendarViewAgenda, calendarViewAvailability:
	default:
		return calendarActionRequest{}, fmt.Errorf("unsupported calendar view %q", req.View)
	}
	if rawDate := calendarOptionalParam(action.Params, "date"); rawDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", rawDate, time.Local)
		if err != nil {
			return calendarActionRequest{}, fmt.Errorf("calendar date must be YYYY-MM-DD")
		}
		req.Date = parsed
	}
	return req, nil
}

func (a *App) executeCalendarCreateAction(session store.ChatSession, action *SystemAction) (string, map[string]interface{}, error) {
	if a == nil || action == nil {
		return "", nil, fmt.Errorf("calendar create action is required")
	}
	req, err := a.parseCalendarCreateRequest(action)
	if err != nil {
		return "", nil, err
	}
	activeSphere, err := a.store.ActiveSphere()
	if err != nil || strings.TrimSpace(activeSphere) == "" {
		activeSphere = store.SpherePrivate
	}
	accounts, err := tabcalendar.GoogleCalendarAccounts(a.store)
	if err != nil {
		return "", nil, err
	}
	if len(accounts) == 0 {
		return "", nil, fmt.Errorf("no Google Calendar accounts are configured")
	}
	if a.newGoogleCalendarClient == nil {
		return "", nil, fmt.Errorf("google calendar client is unavailable")
	}
	client, err := a.newGoogleCalendarClient(context.Background())
	if err != nil {
		return "", nil, err
	}
	calendars, err := client.ListCalendars(context.Background())
	if err != nil {
		return "", nil, err
	}
	target, err := tabcalendar.SelectCalendar(calendars, a.store, accounts, req.CalendarID, activeSphere)
	if err != nil {
		return "", nil, err
	}
	event, err := client.CreateEvent(context.Background(), tabcalendar.CreateEventOptions{
		CalendarID:  target.ID,
		Summary:     req.Summary,
		Description: req.Description,
		Location:    req.Location,
		Start:       req.Start,
		End:         req.End,
		AllDay:      req.AllDay,
		Attendees:   req.Attendees,
	})
	if err != nil {
		return "", nil, err
	}
	targetSphere := tabcalendar.ResolveCalendarSphere(a.store, store.ExternalProviderGoogleCalendar, target.ID, target.Name, activeSphere, accounts)
	return fmt.Sprintf("Created calendar event %q for %s.", event.Summary, calendarConfirmationTimeLabel(event.Start, event.AllDay)), map[string]interface{}{
		"type":  "create_calendar_event",
		"event": calendarEventPayload(event, target.Name, targetSphere),
	}, nil
}

func (a *App) executeCalendarUpdateAction(session store.ChatSession, action *SystemAction) (string, map[string]interface{}, error) {
	if a == nil || action == nil {
		return "", nil, fmt.Errorf("calendar update action is required")
	}
	eventID := calendarOptionalParam(action.Params, "event_id")
	if eventID == "" {
		return "", nil, fmt.Errorf("event_id is required to update a calendar event")
	}
	calendarID := calendarOptionalParam(action.Params, "calendar_id")
	if a.newGoogleCalendarClient == nil {
		return "", nil, fmt.Errorf("google calendar client is unavailable")
	}
	client, err := a.newGoogleCalendarClient(context.Background())
	if err != nil {
		return "", nil, err
	}
	opts := tabcalendar.UpdateEventOptions{
		CalendarID:  calendarID,
		EventID:     eventID,
		Summary:     firstNonEmptyCalendarValue(calendarOptionalParam(action.Params, "summary"), calendarOptionalParam(action.Params, "title")),
		Description: calendarOptionalParam(action.Params, "description"),
		Location:    calendarOptionalParam(action.Params, "location"),
		Attendees:   calendarStringListParam(action.Params, "attendees"),
	}
	if startStr := calendarOptionalParam(action.Params, "start"); startStr != "" {
		opts.Start = parseCalendarInputTimeOrZero(startStr)
	}
	if endStr := calendarOptionalParam(action.Params, "end"); endStr != "" {
		opts.End = parseCalendarInputTimeOrZero(endStr)
	}
	event, err := client.UpdateEvent(context.Background(), opts)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("Updated calendar event %q.", event.Summary), map[string]interface{}{
		"type":  "update_calendar_event",
		"event": calendarEventPayload(event, "", ""),
	}, nil
}

func (a *App) executeCalendarDeleteAction(session store.ChatSession, action *SystemAction) (string, map[string]interface{}, error) {
	if a == nil || action == nil {
		return "", nil, fmt.Errorf("calendar delete action is required")
	}
	eventID := calendarOptionalParam(action.Params, "event_id")
	if eventID == "" {
		return "", nil, fmt.Errorf("event_id is required to delete a calendar event")
	}
	calendarID := calendarOptionalParam(action.Params, "calendar_id")
	if a.newGoogleCalendarClient == nil {
		return "", nil, fmt.Errorf("google calendar client is unavailable")
	}
	client, err := a.newGoogleCalendarClient(context.Background())
	if err != nil {
		return "", nil, err
	}
	if err := client.DeleteEvent(context.Background(), calendarID, eventID); err != nil {
		return "", nil, err
	}
	return "Deleted the calendar event.", map[string]interface{}{
		"type":     "delete_calendar_event",
		"event_id": eventID,
	}, nil
}

func (a *App) parseCalendarCreateRequest(action *SystemAction) (calendarCreateRequest, error) {
	req := calendarCreateRequest{
		CalendarID:  calendarOptionalParam(action.Params, "calendar_id"),
		Summary:     firstNonEmptyCalendarValue(calendarOptionalParam(action.Params, "summary"), calendarOptionalParam(action.Params, "title"), calendarOptionalParam(action.Params, "text")),
		Description: calendarOptionalParam(action.Params, "description"),
		Location:    calendarOptionalParam(action.Params, "location"),
		Attendees:   calendarStringListParam(action.Params, "attendees"),
		AllDay:      calendarBoolParam(action.Params, "all_day"),
	}
	if req.Summary == "" {
		return calendarCreateRequest{}, fmt.Errorf("calendar event summary is required")
	}
	startRaw := calendarOptionalParam(action.Params, "start")
	if startRaw == "" {
		startRaw = calendarStartFromDateAndTime(action.Params)
	}
	start, err := parseCalendarInputTime(startRaw)
	if err != nil {
		return calendarCreateRequest{}, fmt.Errorf("calendar event start must be RFC3339, YYYY-MM-DDTHH:MM, YYYY-MM-DD HH:MM, or YYYY-MM-DD")
	}
	if calendarTimeLooksDateOnly(startRaw) {
		req.AllDay = true
	}
	req.Start = start

	endRaw := calendarOptionalParam(action.Params, "end")
	if endRaw != "" {
		end, endErr := parseCalendarInputTime(endRaw)
		if endErr != nil {
			return calendarCreateRequest{}, fmt.Errorf("calendar event end must be RFC3339, YYYY-MM-DDTHH:MM, YYYY-MM-DD HH:MM, or YYYY-MM-DD")
		}
		req.End = end
		if calendarTimeLooksDateOnly(endRaw) {
			req.AllDay = true
		}
	}
	if req.End.IsZero() {
		durationMinutes := calendarIntParam(action.Params, "duration_minutes")
		if durationMinutes <= 0 {
			if req.AllDay {
				durationMinutes = 24 * 60
			} else {
				durationMinutes = calendarCreateDefaultMin
			}
		}
		req.End = req.Start.Add(time.Duration(durationMinutes) * time.Minute)
	}
	if !req.End.After(req.Start) {
		return calendarCreateRequest{}, fmt.Errorf("calendar event end must be after start")
	}
	return req, nil
}

func parseInlineCalendarCreateIntent(text string, now time.Time) *SystemAction {
	match := calendarCreatePattern.FindStringSubmatch(strings.TrimSpace(text))
	if len(match) != 5 {
		return nil
	}
	start, err := parseInlineCalendarCreateStart(now, match[1], match[2], match[3])
	if err != nil {
		return nil
	}
	summary := strings.TrimSpace(match[4])
	if summary == "" {
		return nil
	}
	return &SystemAction{
		Action: "create_calendar_event",
		Params: map[string]interface{}{
			"summary":          summary,
			"start":            start.Format(time.RFC3339),
			"duration_minutes": calendarCreateDefaultMin,
		},
	}
}

func parseInlineCalendarCreateStart(now time.Time, rawDate, rawHour, rawMinute string) (time.Time, error) {
	parts := strings.Split(strings.TrimSpace(rawDate), ".")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid date")
	}
	day, err := parseCalendarDatePart(parts[0], 31)
	if err != nil {
		return time.Time{}, err
	}
	month, err := parseCalendarDatePart(parts[1], 12)
	if err != nil {
		return time.Time{}, err
	}
	year := now.In(time.Local).Year()
	if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
		year, err = parseCalendarYearPart(parts[2])
		if err != nil {
			return time.Time{}, err
		}
	}
	hour, err := parseCalendarDatePart(rawHour, 23)
	if err != nil {
		return time.Time{}, err
	}
	minute := 0
	if strings.TrimSpace(rawMinute) != "" {
		minute, err = parseCalendarDatePart(rawMinute, 59)
		if err != nil {
			return time.Time{}, err
		}
	}
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local), nil
}

func parseCalendarDatePart(raw string, max int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("missing date value")
	}
	n, err := fmt.Sscanf(value, "%d", new(int))
	if err != nil || n != 1 {
		var parsed int
		if _, scanErr := fmt.Sscanf(value, "%d", &parsed); scanErr != nil {
			return 0, scanErr
		}
		if parsed < 0 || parsed > max {
			return 0, fmt.Errorf("date value out of range")
		}
		return parsed, nil
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return 0, err
	}
	if parsed < 0 || parsed > max {
		return 0, fmt.Errorf("date value out of range")
	}
	return parsed, nil
}

func parseCalendarYearPart(raw string) (int, error) {
	year, err := parseCalendarDatePart(raw, 9999)
	if err != nil {
		return 0, err
	}
	if year < 100 {
		return 2000 + year, nil
	}
	return year, nil
}

func parseCalendarInputTime(raw string) (time.Time, error) {
	clean := strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if layout == "2006-01-02" || layout == "2006-01-02T15:04" || layout == "2006-01-02 15:04" {
			if parsed, err := time.ParseInLocation(layout, clean, time.Local); err == nil {
				return parsed, nil
			}
			continue
		}
		if parsed, err := time.Parse(layout, clean); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time")
}

func parseCalendarInputTimeOrZero(raw string) time.Time {
	t, _ := parseCalendarInputTime(raw)
	return t
}

func calendarTimeLooksDateOnly(raw string) bool {
	clean := strings.TrimSpace(raw)
	return len(clean) == len("2006-01-02") && strings.Count(clean, "-") == 2
}

func calendarStartFromDateAndTime(params map[string]interface{}) string {
	dateValue := calendarOptionalParam(params, "date")
	if dateValue == "" {
		return ""
	}
	timeValue := calendarOptionalParam(params, "time")
	if timeValue == "" {
		return dateValue
	}
	return dateValue + " " + timeValue
}

func calendarStringListParam(params map[string]interface{}, key string) []string {
	value, ok := params[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if clean := strings.TrimSpace(item); clean != "" {
				out = append(out, clean)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if clean := strings.TrimSpace(fmt.Sprint(item)); clean != "" && clean != "<nil>" {
				out = append(out, clean)
			}
		}
		return out
	case string:
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if clean := strings.TrimSpace(part); clean != "" {
				out = append(out, clean)
			}
		}
		return out
	default:
		return nil
	}
}

func calendarBoolParam(params map[string]interface{}, key string) bool {
	switch typed := params[key].(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func calendarIntParam(params map[string]interface{}, key string) int {
	switch typed := params[key].(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func calendarConfirmationTimeLabel(start time.Time, allDay bool) string {
	if allDay {
		return start.In(time.Local).Format("2006-01-02")
	}
	return start.In(time.Local).Format("2006-01-02 15:04")
}

func calendarEventPayload(event providerdata.Event, calendarName, sphere string) map[string]interface{} {
	return map[string]interface{}{
		"id":            event.ID,
		"calendar_id":   event.CalendarID,
		"calendar_name": calendarName,
		"provider":      store.ExternalProviderGoogleCalendar,
		"sphere":        sphere,
		"summary":       event.Summary,
		"description":   event.Description,
		"location":      event.Location,
		"start":         event.Start.Format(time.RFC3339),
		"end":           event.End.Format(time.RFC3339),
		"all_day":       event.AllDay,
		"status":        event.Status,
		"organizer":     event.Organizer,
		"attendees":     append([]string(nil), event.Attendees...),
		"recurring":     event.Recurring,
	}
}

func (a *App) collectCalendarEvents(ctx context.Context, req calendarActionRequest, activeSphere string) ([]calendarEventEntry, []string, error) {
	timeMin, timeMax := calendarTimeRange(req)
	var (
		events   []calendarEventEntry
		warnings []string
	)
	googleEvents, googleWarnings, err := a.collectGoogleCalendarEvents(ctx, req, activeSphere, timeMin, timeMax)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, googleEvents...)
	warnings = append(warnings, googleWarnings...)
	exchangeEvents, exchangeWarnings, err := a.collectExchangeEWSEvents(ctx, req, activeSphere, timeMin, timeMax)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, exchangeEvents...)
	warnings = append(warnings, exchangeWarnings...)
	icsEvents, icsWarnings, err := a.collectICSEvents(req, activeSphere, timeMin, timeMax)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, icsEvents...)
	warnings = append(warnings, icsWarnings...)
	sort.Slice(events, func(i, j int) bool {
		if events[i].Start.Equal(events[j].Start) {
			if events[i].Sphere == events[j].Sphere {
				return strings.ToLower(events[i].Summary) < strings.ToLower(events[j].Summary)
			}
			return events[i].Sphere < events[j].Sphere
		}
		return events[i].Start.Before(events[j].Start)
	})
	return events, warnings, nil
}

func (a *App) collectExchangeEWSEvents(ctx context.Context, req calendarActionRequest, activeSphere string, timeMin, timeMax time.Time) ([]calendarEventEntry, []string, error) {
	accounts, err := a.store.ListExternalAccountsByProvider(store.ExternalProviderExchangeEWS)
	if err != nil {
		return nil, nil, err
	}
	var (
		events   []calendarEventEntry
		warnings []string
	)
	for _, account := range accounts {
		if !account.Enabled {
			continue
		}
		client, clientErr := a.exchangeEWSClientForAccount(ctx, account)
		if clientErr != nil {
			warnings = append(warnings, fmt.Sprintf("Exchange EWS %q unavailable: %v", account.AccountName, clientErr))
			continue
		}
		items, eventErr := client.GetCalendarEvents(ctx, "", 0, 250)
		_ = client.Close()
		if eventErr != nil {
			warnings = append(warnings, fmt.Sprintf("Exchange EWS %q failed: %v", account.AccountName, eventErr))
			continue
		}
		for _, item := range items {
			if item.End.Before(timeMin) || item.Start.After(timeMax) {
				continue
			}
			entry := calendarEventEntry{
				Summary:     strings.TrimSpace(item.Subject),
				Description: strings.TrimSpace(item.Body),
				Location:    strings.TrimSpace(item.Location),
				Source:      firstNonEmptyCalendarValue(account.AccountName, "Kalender", "Exchange EWS"),
				Provider:    store.ExternalProviderExchangeEWS,
				Sphere:      account.Sphere,
				Start:       item.Start.In(time.Local),
				End:         item.End.In(time.Local),
				AllDay:      item.IsAllDay,
			}
			if !matchesCalendarQuery(req.Query, entry, "") {
				continue
			}
			events = append(events, entry)
		}
	}
	return events, warnings, nil
}

func (a *App) collectGoogleCalendarEvents(ctx context.Context, req calendarActionRequest, activeSphere string, timeMin, timeMax time.Time) ([]calendarEventEntry, []string, error) {
	accounts, err := tabcalendar.GoogleCalendarAccounts(a.store)
	if err != nil {
		return nil, nil, err
	}
	if len(accounts) == 0 || a.newGoogleCalendarClient == nil {
		return nil, nil, nil
	}
	client, err := a.newGoogleCalendarClient(ctx)
	if err != nil {
		return nil, []string{fmt.Sprintf("Google Calendar unavailable: %v", err)}, nil
	}
	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		return nil, []string{fmt.Sprintf("Google Calendar list failed: %v", err)}, nil
	}
	var (
		events   []calendarEventEntry
		warnings []string
	)
	for _, cal := range calendars {
		providerSphere := tabcalendar.ResolveCalendarSphere(a.store, store.ExternalProviderGoogleCalendar, cal.ID, cal.Name, activeSphere, accounts)
		calEvents, eventErr := client.GetEvents(ctx, tabcalendar.GetEventsOptions{
			CalendarID: cal.ID,
			TimeMin:    timeMin,
			TimeMax:    timeMax,
			MaxResults: 250,
			Query:      strings.TrimSpace(req.Query),
		})
		if eventErr != nil {
			warnings = append(warnings, fmt.Sprintf("Google Calendar %q failed: %v", cal.Name, eventErr))
			continue
		}
		for _, event := range calEvents {
			entry := calendarEventEntry{
				Summary:     strings.TrimSpace(event.Summary),
				Description: strings.TrimSpace(event.Description),
				Location:    strings.TrimSpace(event.Location),
				Attendees:   append([]string(nil), event.Attendees...),
				Source:      firstNonEmptyCalendarValue(cal.Name, cal.ID, "Google Calendar"),
				Provider:    store.ExternalProviderGoogleCalendar,
				Sphere:      providerSphere,
				Start:       event.Start.In(time.Local),
				End:         event.End.In(time.Local),
				AllDay:      event.AllDay,
			}
			if !matchesCalendarQuery(req.Query, entry, "") {
				continue
			}
			events = append(events, entry)
		}
	}
	return events, warnings, nil
}

func (a *App) collectICSEvents(req calendarActionRequest, activeSphere string, timeMin, timeMax time.Time) ([]calendarEventEntry, []string, error) {
	if a == nil || a.newICSCalendarClient == nil {
		return nil, nil, nil
	}
	client, err := a.newICSCalendarClient()
	if err != nil {
		return nil, []string{fmt.Sprintf("ICS calendars unavailable: %v", err)}, nil
	}
	accounts, err := a.store.ListExternalAccountsByProvider(store.ExternalProviderICS)
	if err != nil {
		return nil, nil, err
	}
	var (
		events   []calendarEventEntry
		warnings []string
	)
	for _, cal := range client.ListCalendars() {
		providerSphere := tabcalendar.ResolveCalendarSphere(a.store, store.ExternalProviderICS, cal.ID, cal.Name, activeSphere, accounts)
		calEvents, eventErr := client.GetEvents(cal.Name, timeMin, timeMax)
		if eventErr != nil {
			warnings = append(warnings, fmt.Sprintf("ICS calendar %q failed: %v", cal.Name, eventErr))
			continue
		}
		for _, event := range calEvents {
			entry := calendarEventEntry{
				Summary:     strings.TrimSpace(event.Summary),
				Description: strings.TrimSpace(event.Description),
				Location:    strings.TrimSpace(event.Location),
				Source:      firstNonEmptyCalendarValue(cal.Name, cal.ID, "ICS"),
				Provider:    store.ExternalProviderICS,
				Sphere:      providerSphere,
				Start:       event.Start.In(time.Local),
				End:         event.End.In(time.Local),
				AllDay:      event.AllDay,
			}
			if !matchesCalendarQuery(req.Query, entry, "") {
				continue
			}
			events = append(events, entry)
		}
	}
	return events, warnings, nil
}

func (a *App) collectCalendarDeadlines(req calendarActionRequest) ([]calendarDeadlineEntry, error) {
	items, err := a.store.ListItems()
	if err != nil {
		return nil, err
	}
	timeMin, timeMax := calendarTimeRange(req)
	workspaceNames := map[int64]string{}
	projectNames := map[int64]string{}
	var deadlines []calendarDeadlineEntry
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.State), store.ItemStateDone) {
			continue
		}
		if item.FollowUpAt != nil {
			when, parseErr := parseCalendarTimestamp(*item.FollowUpAt)
			if parseErr == nil && !when.Before(timeMin) && when.Before(timeMax) {
				entry := calendarDeadlineEntry{
					Title:     item.Title,
					Sphere:    item.Sphere,
					Kind:      "Due",
					When:      when.In(time.Local),
					Workspace: calendarWorkspaceName(a, item.WorkspaceID, workspaceNames),
					Project:   calendarProjectName(a, item.WorkspaceID, projectNames),
				}
				if matchesCalendarQuery(req.Query, calendarEventEntry{}, calendarDeadlineSearchText(entry)) {
					deadlines = append(deadlines, entry)
				}
			}
		}
		if item.VisibleAfter != nil {
			when, parseErr := parseCalendarTimestamp(*item.VisibleAfter)
			if parseErr == nil && !when.Before(timeMin) && when.Before(timeMax) {
				entry := calendarDeadlineEntry{
					Title:     item.Title,
					Sphere:    item.Sphere,
					Kind:      "Resurface",
					When:      when.In(time.Local),
					Workspace: calendarWorkspaceName(a, item.WorkspaceID, workspaceNames),
					Project:   calendarProjectName(a, item.WorkspaceID, projectNames),
				}
				if matchesCalendarQuery(req.Query, calendarEventEntry{}, calendarDeadlineSearchText(entry)) {
					deadlines = append(deadlines, entry)
				}
			}
		}
	}
	sort.Slice(deadlines, func(i, j int) bool {
		if deadlines[i].When.Equal(deadlines[j].When) {
			if deadlines[i].Kind == deadlines[j].Kind {
				return strings.ToLower(deadlines[i].Title) < strings.ToLower(deadlines[j].Title)
			}
			return deadlines[i].Kind < deadlines[j].Kind
		}
		return deadlines[i].When.Before(deadlines[j].When)
	})
	return deadlines, nil
}

func sanitizeCalendarFileToken(raw string) string {
	clean := strings.TrimSpace(strings.ToLower(raw))
	if clean == "" {
		return "calendar"
	}
	clean = calendarTokenSanitizer.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-.")
	if clean == "" {
		return "calendar"
	}
	return clean
}

func parseCalendarTimestamp(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
}

func firstNonEmptyCalendarValue(values ...string) string {
	for _, value := range values {
		if clean := strings.TrimSpace(value); clean != "" {
			return clean
		}
	}
	return ""
}

func calendarOptionalParam(params map[string]interface{}, key string) string {
	clean := strings.TrimSpace(fmt.Sprint(params[key]))
	if clean == "" || clean == "<nil>" {
		return ""
	}
	return clean
}
