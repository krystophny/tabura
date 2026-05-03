package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const timeEntriesTableSchema = `CREATE TABLE IF NOT EXISTS time_entries (
  id INTEGER PRIMARY KEY,
  workspace_id INTEGER REFERENCES workspaces(id) ON DELETE SET NULL,
  project_item_id INTEGER REFERENCES items(id) ON DELETE SET NULL,
  action_item_id INTEGER REFERENCES items(id) ON DELETE SET NULL,
  track TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  ended_at TEXT,
  activity TEXT NOT NULL DEFAULT '',
  notes TEXT
);`

func scanTimeEntry(scanner interface {
	Scan(dest ...any) error
}) (TimeEntry, error) {
	var (
		out         TimeEntry
		workspace   sql.NullInt64
		projectItem sql.NullInt64
		actionItem  sql.NullInt64
		endedAt     sql.NullString
		notes       sql.NullString
	)
	if err := scanner.Scan(
		&out.ID,
		&workspace,
		&projectItem,
		&actionItem,
		&out.Sphere,
		&out.Track,
		&out.StartedAt,
		&endedAt,
		&out.Activity,
		&notes,
	); err != nil {
		return TimeEntry{}, err
	}
	out.WorkspaceID = nullInt64Pointer(workspace)
	out.ProjectItemID = nullInt64Pointer(projectItem)
	out.ActionItemID = nullInt64Pointer(actionItem)
	out.EndedAt = nullStringPointer(endedAt)
	out.Notes = nullStringPointer(notes)
	out.Sphere = normalizeSphere(out.Sphere)
	out.Track = strings.TrimSpace(out.Track)
	out.Activity = strings.TrimSpace(out.Activity)
	return out, nil
}

func formatTimeEntryTimestamp(ts time.Time) string {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return ts.UTC().Format(time.RFC3339)
}

func normalizeTimeEntryFilter(filter TimeEntryListFilter) (TimeEntryListFilter, error) {
	normalized := TimeEntryListFilter{ActiveOnly: filter.ActiveOnly, Track: strings.TrimSpace(filter.Track)}
	sphere, err := normalizeOptionalSphereFilter(filter.Sphere)
	if err != nil {
		return TimeEntryListFilter{}, err
	}
	normalized.Sphere = sphere
	if filter.From != nil {
		from := filter.From.UTC()
		normalized.From = &from
	}
	if filter.To != nil {
		to := filter.To.UTC()
		normalized.To = &to
	}
	if normalized.From != nil && normalized.To != nil && !normalized.To.After(*normalized.From) {
		return TimeEntryListFilter{}, errors.New("time range end must be after start")
	}
	return normalized, nil
}

func timeEntryContextMatches(entry *TimeEntry, workspaceID *int64, sphere string) bool {
	return timeEntryContextMatchesFocus(entry, TimeEntryFocus{WorkspaceID: workspaceID, Sphere: sphere})
}

func timeEntryContextMatchesTrack(entry *TimeEntry, workspaceID *int64, sphere, track string) bool {
	return timeEntryContextMatchesFocus(entry, TimeEntryFocus{WorkspaceID: workspaceID, Sphere: sphere, Track: track})
}

func timeEntryContextMatchesFocus(entry *TimeEntry, focus TimeEntryFocus) bool {
	if entry == nil {
		return false
	}
	if normalizeSphere(entry.Sphere) != normalizeSphere(focus.Sphere) {
		return false
	}
	if strings.TrimSpace(entry.Track) != strings.TrimSpace(focus.Track) {
		return false
	}
	return sameOptionalID(entry.WorkspaceID, focus.WorkspaceID) &&
		sameOptionalID(entry.ProjectItemID, focus.ProjectItemID) &&
		sameOptionalID(entry.ActionItemID, focus.ActionItemID)
}

func sameOptionalID(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (s *Store) validateTimeEntryContext(workspaceID *int64, sphere string) error {
	return s.validateTimeEntryFocus(TimeEntryFocus{WorkspaceID: workspaceID, Sphere: sphere})
}

func (s *Store) validateTimeEntryFocus(focus TimeEntryFocus) error {
	if normalizeRequiredSphere(focus.Sphere) == "" {
		return errors.New("sphere must be work or private")
	}
	if focus.WorkspaceID != nil {
		if *focus.WorkspaceID <= 0 {
			return errors.New("workspace_id must be a positive integer")
		}
		if _, err := s.GetWorkspace(*focus.WorkspaceID); err != nil {
			return err
		}
	}
	if focus.ProjectItemID != nil {
		if _, err := s.validateFocusItem(*focus.ProjectItemID, ItemKindProject); err != nil {
			return err
		}
	}
	if focus.ActionItemID != nil {
		action, err := s.validateFocusItem(*focus.ActionItemID, ItemKindAction)
		if err != nil {
			return err
		}
		if focus.ProjectItemID == nil {
			return nil
		}
		if !s.itemChildLinkExists(*focus.ProjectItemID, action.ID) {
			return errors.New("action item is not linked to project item")
		}
	}
	return nil
}

func (s *Store) validateFocusItem(id int64, kind ItemKind) (Item, error) {
	if id <= 0 {
		return Item{}, errors.New("item_id must be a positive integer")
	}
	item, err := s.GetItem(id)
	if err != nil {
		return Item{}, err
	}
	if item.Kind != string(kind) {
		return Item{}, fmt.Errorf("item %d is not a %s", id, kind)
	}
	return item, nil
}

func (s *Store) itemChildLinkExists(parentID, childID int64) bool {
	var found int
	err := s.db.QueryRow(`SELECT 1 FROM item_children WHERE parent_item_id = ? AND child_item_id = ? LIMIT 1`, parentID, childID).Scan(&found)
	return err == nil && found == 1
}

func (s *Store) ActiveWorkspace() (Workspace, error) {
	return scanWorkspace(s.db.QueryRow(
		`SELECT id, name, dir_path, ` + scopedContextSelect("context_workspaces", "workspace_id", "workspaces.id") + ` AS sphere, source_workspace_id, source_path, is_active, is_daily, daily_date, mcp_url, canvas_session_id, chat_model, chat_model_reasoning_effort, companion_config_json, created_at, updated_at
		 FROM workspaces
		 WHERE is_active <> 0
		 ORDER BY updated_at DESC, id DESC
		 LIMIT 1`,
	))
}

func (s *Store) ActiveTimeEntry() (*TimeEntry, error) {
	entry, err := scanTimeEntry(s.db.QueryRow(
		`SELECT id, workspace_id, project_item_id, action_item_id, ` + scopedContextSelect("context_time_entries", "time_entry_id", "time_entries.id") + ` AS sphere, track, started_at, ended_at, activity, notes
		 FROM time_entries
		 WHERE ended_at IS NULL
		 ORDER BY started_at DESC, id DESC
		 LIMIT 1`,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (s *Store) StartTimeEntry(at time.Time, workspaceID *int64, sphere, activity string, notes *string) (TimeEntry, error) {
	return s.StartTimeEntryWithTrack(at, workspaceID, sphere, "", activity, notes)
}

func (s *Store) StartTimeEntryWithTrack(at time.Time, workspaceID *int64, sphere, track, activity string, notes *string) (TimeEntry, error) {
	return s.StartTimeEntryWithFocus(at, TimeEntryFocus{WorkspaceID: workspaceID, Sphere: sphere, Track: track}, activity, notes)
}

func (s *Store) StartTimeEntryWithFocus(at time.Time, focus TimeEntryFocus, activity string, notes *string) (TimeEntry, error) {
	focus.Sphere = normalizeRequiredSphere(focus.Sphere)
	focus.Track = strings.TrimSpace(focus.Track)
	if err := s.validateTimeEntryFocus(focus); err != nil {
		return TimeEntry{}, err
	}
	startedAt := formatTimeEntryTimestamp(at)
	cleanActivity := strings.TrimSpace(activity)
	if cleanActivity == "" {
		cleanActivity = "context_switch"
	}
	res, err := s.db.Exec(
		`INSERT INTO time_entries (workspace_id, project_item_id, action_item_id, track, started_at, activity, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		nullablePositiveID(derefInt64(focus.WorkspaceID)),
		nullablePositiveID(derefInt64(focus.ProjectItemID)),
		nullablePositiveID(derefInt64(focus.ActionItemID)),
		focus.Track,
		startedAt,
		cleanActivity,
		normalizeOptionalString(notes),
	)
	if err != nil {
		return TimeEntry{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return TimeEntry{}, err
	}
	if err := s.syncScopedContextLink("context_time_entries", "time_entry_id", id, focus.Sphere); err != nil {
		return TimeEntry{}, err
	}
	return s.GetTimeEntry(id)
}

func (s *Store) SwitchActiveTimeEntry(at time.Time, workspaceID *int64, sphere, activity string, notes *string) (TimeEntry, bool, error) {
	return s.SwitchActiveTimeEntryWithTrack(at, workspaceID, sphere, "", activity, notes)
}

func (s *Store) SwitchActiveTimeEntryWithTrack(at time.Time, workspaceID *int64, sphere, track, activity string, notes *string) (TimeEntry, bool, error) {
	return s.SwitchActiveTimeEntryWithFocus(at, TimeEntryFocus{WorkspaceID: workspaceID, Sphere: sphere, Track: track}, activity, notes)
}

func (s *Store) SwitchActiveTimeEntryWithFocus(at time.Time, focus TimeEntryFocus, activity string, notes *string) (TimeEntry, bool, error) {
	focus.Sphere = normalizeRequiredSphere(focus.Sphere)
	focus.Track = strings.TrimSpace(focus.Track)
	if err := s.validateTimeEntryFocus(focus); err != nil {
		return TimeEntry{}, false, err
	}
	active, err := s.ActiveTimeEntry()
	if err != nil {
		return TimeEntry{}, false, err
	}
	if timeEntryContextMatchesFocus(active, focus) {
		return *active, false, nil
	}
	if _, err := s.StopActiveTimeEntries(at); err != nil {
		return TimeEntry{}, false, err
	}
	entry, err := s.StartTimeEntryWithFocus(at, focus, activity, notes)
	if err != nil {
		return TimeEntry{}, false, err
	}
	return entry, true, nil
}

func (s *Store) StopActiveTimeEntries(at time.Time) (int64, error) {
	res, err := s.db.Exec(
		`UPDATE time_entries
		 SET ended_at = ?
		 WHERE ended_at IS NULL`,
		formatTimeEntryTimestamp(at),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) GetTimeEntry(id int64) (TimeEntry, error) {
	return scanTimeEntry(s.db.QueryRow(
		`SELECT id, workspace_id, project_item_id, action_item_id, `+scopedContextSelect("context_time_entries", "time_entry_id", "time_entries.id")+` AS sphere, track, started_at, ended_at, activity, notes
		 FROM time_entries
		 WHERE id = ?`,
		id,
	))
}

func (s *Store) ListTimeEntries(filter TimeEntryListFilter) ([]TimeEntry, error) {
	normalized, err := normalizeTimeEntryFilter(filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, workspace_id, project_item_id, action_item_id, ` + scopedContextSelect("context_time_entries", "time_entry_id", "time_entries.id") + ` AS sphere, track, started_at, ended_at, activity, notes
		FROM time_entries`
	parts := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if normalized.Sphere != "" {
		parts = append(parts, scopedContextFilter("context_time_entries", "time_entry_id", "time_entries.id"))
		args = append(args, normalized.Sphere)
	}
	if normalized.ActiveOnly {
		parts = append(parts, "ended_at IS NULL")
	}
	if normalized.Track != "" {
		parts = append(parts, "lower(trim(track)) = lower(trim(?))")
		args = append(args, normalized.Track)
	}
	if normalized.From != nil {
		parts = append(parts, "(ended_at IS NULL OR ended_at >= ?)")
		args = append(args, formatTimeEntryTimestamp(*normalized.From))
	}
	if normalized.To != nil {
		parts = append(parts, "started_at < ?")
		args = append(args, formatTimeEntryTimestamp(*normalized.To))
	}
	if len(parts) > 0 {
		query += " WHERE " + strings.Join(parts, " AND ")
	}
	query += " ORDER BY started_at ASC, id ASC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []TimeEntry{}
	for rows.Next() {
		entry, err := scanTimeEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) SummarizeTimeEntries(filter TimeEntryListFilter, groupBy string, now time.Time) ([]TimeEntrySummary, error) {
	normalized, err := normalizeTimeEntryFilter(filter)
	if err != nil {
		return nil, err
	}
	cleanGroupBy := strings.ToLower(strings.TrimSpace(groupBy))
	switch cleanGroupBy {
	case "workspace", "project", "action", "sphere", "track":
	default:
		return nil, errors.New("group_by must be workspace, project, action, sphere or track")
	}
	entries, err := s.ListTimeEntries(normalized)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	type key struct {
		value string
	}
	summaries := map[key]*TimeEntrySummary{}
	itemLabels := map[int64]string{}
	for _, entry := range entries {
		startedAt, err := time.Parse(time.RFC3339, entry.StartedAt)
		if err != nil {
			return nil, fmt.Errorf("parse started_at for time entry %d: %w", entry.ID, err)
		}
		endedAt := now
		if entry.EndedAt != nil {
			endedAt, err = time.Parse(time.RFC3339, *entry.EndedAt)
			if err != nil {
				return nil, fmt.Errorf("parse ended_at for time entry %d: %w", entry.ID, err)
			}
		}
		if normalized.From != nil && startedAt.Before(*normalized.From) {
			startedAt = *normalized.From
		}
		if normalized.To != nil && endedAt.After(*normalized.To) {
			endedAt = *normalized.To
		}
		if !endedAt.After(startedAt) {
			continue
		}
		seconds := int64(endedAt.Sub(startedAt).Seconds())
		if seconds <= 0 {
			continue
		}
		summaryKey, summary := summarizeTimeEntry(entry, cleanGroupBy)
		if needsTimeEntrySummaryItemLabel(cleanGroupBy, entry) {
			itemID := timeEntrySummaryItemID(cleanGroupBy, entry)
			if _, ok := itemLabels[itemID]; !ok {
				item, err := s.GetItem(itemID)
				if err != nil {
					return nil, err
				}
				itemLabels[itemID] = item.Title
			}
			summary.Label = itemLabels[itemID]
		}
		if cleanGroupBy == "workspace" && entry.WorkspaceID != nil {
			workspace, err := s.GetWorkspace(*entry.WorkspaceID)
			if err != nil {
				return nil, err
			}
			summary.Label = workspace.Name
		}
		current := summaries[key{value: summaryKey}]
		if current == nil {
			copySummary := summary
			summaries[key{value: summaryKey}] = &copySummary
			current = &copySummary
		}
		current.Seconds += seconds
		current.EntryCount++
		current.Duration = formatDurationSeconds(current.Seconds)
	}
	rows := make([]TimeEntrySummary, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, *summary)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Seconds != rows[j].Seconds {
			return rows[i].Seconds > rows[j].Seconds
		}
		return rows[i].Label < rows[j].Label
	})
	return rows, nil
}

func summarizeTimeEntry(entry TimeEntry, groupBy string) (string, TimeEntrySummary) {
	switch groupBy {
	case "workspace":
		if entry.WorkspaceID == nil {
			return "workspace:none", TimeEntrySummary{
				Key:    "workspace:none",
				Label:  "No workspace",
				Sphere: entry.Sphere,
			}
		}
		return fmt.Sprintf("workspace:%d", *entry.WorkspaceID), TimeEntrySummary{
			Key:         fmt.Sprintf("workspace:%d", *entry.WorkspaceID),
			Label:       fmt.Sprintf("Workspace %d", *entry.WorkspaceID),
			WorkspaceID: entry.WorkspaceID,
			Sphere:      entry.Sphere,
		}
	case "project":
		if entry.ProjectItemID == nil {
			return "project:none", TimeEntrySummary{Key: "project:none", Label: "No project item", Sphere: entry.Sphere, Track: entry.Track}
		}
		return fmt.Sprintf("project:%d", *entry.ProjectItemID), TimeEntrySummary{
			Key:           fmt.Sprintf("project:%d", *entry.ProjectItemID),
			Label:         fmt.Sprintf("Project item %d", *entry.ProjectItemID),
			ProjectItemID: entry.ProjectItemID,
			Sphere:        entry.Sphere,
			Track:         entry.Track,
		}
	case "action":
		if entry.ActionItemID == nil {
			return "action:none", TimeEntrySummary{Key: "action:none", Label: "No action item", Sphere: entry.Sphere, Track: entry.Track}
		}
		return fmt.Sprintf("action:%d", *entry.ActionItemID), TimeEntrySummary{
			Key:          fmt.Sprintf("action:%d", *entry.ActionItemID),
			Label:        fmt.Sprintf("Action item %d", *entry.ActionItemID),
			ActionItemID: entry.ActionItemID,
			Sphere:       entry.Sphere,
			Track:        entry.Track,
		}
	case "track":
		if strings.TrimSpace(entry.Track) == "" {
			return "track:none", TimeEntrySummary{
				Key:    "track:none",
				Label:  "No track",
				Sphere: entry.Sphere,
			}
		}
		return "track:" + entry.Track, TimeEntrySummary{
			Key:    "track:" + entry.Track,
			Label:  entry.Track,
			Sphere: entry.Sphere,
			Track:  entry.Track,
		}
	default:
		return "sphere:" + entry.Sphere, TimeEntrySummary{
			Key:    "sphere:" + entry.Sphere,
			Label:  entry.Sphere,
			Sphere: entry.Sphere,
		}
	}
}

func needsTimeEntrySummaryItemLabel(groupBy string, entry TimeEntry) bool {
	return timeEntrySummaryItemID(groupBy, entry) > 0
}

func timeEntrySummaryItemID(groupBy string, entry TimeEntry) int64 {
	switch groupBy {
	case "project":
		return derefInt64(entry.ProjectItemID)
	case "action":
		return derefInt64(entry.ActionItemID)
	default:
		return 0
	}
}

func formatDurationSeconds(total int64) string {
	if total < 0 {
		total = 0
	}
	hours := total / 3600
	minutes := (total % 3600) / 60
	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
