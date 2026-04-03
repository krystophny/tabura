package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type localRevisionEntry struct {
	ID          string            `json:"id"`
	CreatedAt   string            `json:"created_at"`
	Kind        string            `json:"kind"`
	Title       string            `json:"title"`
	Path        string            `json:"path,omitempty"`
	Files       map[string]string `json:"files,omitempty"`
	Description string            `json:"description,omitempty"`
}

type localRevisionManifest struct {
	DocumentID string               `json:"document_id"`
	Title      string               `json:"title"`
	Path       string               `json:"path,omitempty"`
	UpdatedAt  string               `json:"updated_at"`
	Entries    []localRevisionEntry `json:"entries"`
}

func sanitizeRevisionSlug(raw string) string {
	input := strings.TrimSpace(raw)
	if input == "" {
		return "canvas"
	}
	input = filepath.ToSlash(input)
	input = strings.Trim(input, "/")
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(input) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "canvas"
	}
	return slug
}

func revisionDocumentSlug(title, path string) string {
	if strings.TrimSpace(path) != "" {
		return sanitizeRevisionSlug(path)
	}
	return sanitizeRevisionSlug(title)
}

func writeLocalRevisionHistory(projectRoot string, manifest localRevisionManifest) (string, error) {
	revisionsDir := filepath.Join(projectRoot, ".sloppad", "revisions", manifest.DocumentID)
	if err := os.MkdirAll(revisionsDir, 0o755); err != nil {
		return "", err
	}
	historyPath := filepath.Join(revisionsDir, "history.md")
	var b strings.Builder
	title := strings.TrimSpace(manifest.Title)
	if title == "" {
		title = manifest.DocumentID
	}
	fmt.Fprintf(&b, "# Local Revision History: %s\n\n", title)
	if path := strings.TrimSpace(manifest.Path); path != "" {
		fmt.Fprintf(&b, "- Source path: `%s`\n", path)
	}
	fmt.Fprintf(&b, "- Updated at: `%s`\n\n", manifest.UpdatedAt)
	if len(manifest.Entries) == 0 {
		b.WriteString("No revisions recorded yet.\n")
	} else {
		for i, entry := range manifest.Entries {
			fmt.Fprintf(&b, "## %d. %s\n\n", i+1, entry.ID)
			fmt.Fprintf(&b, "- Created at: `%s`\n", entry.CreatedAt)
			fmt.Fprintf(&b, "- Kind: `%s`\n", entry.Kind)
			if desc := strings.TrimSpace(entry.Description); desc != "" {
				fmt.Fprintf(&b, "- Note: %s\n", desc)
			}
			if len(entry.Files) > 0 {
				keys := make([]string, 0, len(entry.Files))
				for key := range entry.Files {
					keys = append(keys, key)
				}
				slices.Sort(keys)
				for _, key := range keys {
					fmt.Fprintf(&b, "- %s: `%s`\n", key, entry.Files[key])
				}
			}
			b.WriteString("\n")
		}
	}
	if err := os.WriteFile(historyPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(projectRoot, historyPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relPath), nil
}

func appendLocalRevision(projectRoot, title, path, kind, description string, files map[string]string) (string, string, error) {
	docID := revisionDocumentSlug(title, path)
	revisionsDir := filepath.Join(projectRoot, ".sloppad", "revisions", docID)
	if err := os.MkdirAll(revisionsDir, 0o755); err != nil {
		return "", "", err
	}
	manifestPath := filepath.Join(revisionsDir, "manifest.json")

	manifest := localRevisionManifest{
		DocumentID: docID,
		Title:      strings.TrimSpace(title),
		Path:       strings.TrimSpace(path),
		Entries:    []localRevisionEntry{},
	}
	if raw, err := os.ReadFile(manifestPath); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &manifest)
		if manifest.DocumentID == "" {
			manifest.DocumentID = docID
		}
		if manifest.Title == "" {
			manifest.Title = strings.TrimSpace(title)
		}
		if manifest.Path == "" {
			manifest.Path = strings.TrimSpace(path)
		}
	}

	now := time.Now().UTC()
	entry := localRevisionEntry{
		ID:          now.Format("20060102-150405"),
		CreatedAt:   now.Format(time.RFC3339),
		Kind:        strings.TrimSpace(kind),
		Title:       strings.TrimSpace(title),
		Path:        strings.TrimSpace(path),
		Files:       files,
		Description: strings.TrimSpace(description),
	}
	manifest.UpdatedAt = entry.CreatedAt
	manifest.Entries = append([]localRevisionEntry{entry}, manifest.Entries...)
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(manifestPath, append(encoded, '\n'), 0o644); err != nil {
		return "", "", err
	}
	historyRel, err := writeLocalRevisionHistory(projectRoot, manifest)
	if err != nil {
		return "", "", err
	}
	manifestRel, err := filepath.Rel(projectRoot, manifestPath)
	if err != nil {
		return "", "", err
	}
	return filepath.ToSlash(manifestRel), historyRel, nil
}
