package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// gtdCaptureFlags is the parsed form of `sls gtd capture <title> [flags]`.
//
// Capture deliberately stays terse: title is positional, every other field
// is an opt-in flag. The endpoint owns validation, so the CLI's job is
// translation, not policy.
type gtdCaptureFlags struct {
	title           string
	kind            string
	sphere          string
	workspaceID     string
	actorID         string
	label           string
	labelID         string
	projectItemID   string
	projectRole     string
	source          string
	sourceRef       string
}

type gtdCaptureResponse struct {
	Item        gtdItem `json:"item"`
	Label       *struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"label"`
	ProjectItem *struct {
		ProjectItemID int64  `json:"project_item_id"`
		Role          string `json:"role"`
	} `json:"project_item"`
}

func (c *gtdMutationClient) capture(args []string) (string, error) {
	flags, err := parseGtdCaptureArgs(args)
	if err != nil {
		return "", err
	}
	body, err := captureRequestBody(flags)
	if err != nil {
		return "", err
	}
	var resp gtdCaptureResponse
	if err := c.request(http.MethodPost, "/api/items/capture", body, &resp); err != nil {
		return "", err
	}
	return formatGtdCaptureResult(resp), nil
}

func parseGtdCaptureArgs(args []string) (gtdCaptureFlags, error) {
	flagArgs, titleParts := splitGtdCaptureArgs(args)
	fs := flag.NewFlagSet("sls gtd capture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var f gtdCaptureFlags
	fs.StringVar(&f.kind, "kind", "", "item kind: action (default) or project")
	fs.StringVar(&f.sphere, "vault", "", "sphere: work or private (defaults to active sphere or workspace sphere)")
	fs.StringVar(&f.sphere, "sphere", "", "alias for --vault")
	fs.StringVar(&f.workspaceID, "workspace", "", "Slopshell workspace id")
	fs.StringVar(&f.actorID, "actor-id", "", "actor id to associate with the capture")
	fs.StringVar(&f.label, "label", "", "label name (created if missing)")
	fs.StringVar(&f.labelID, "label-id", "", "label id (alternative to --label)")
	fs.StringVar(&f.projectItemID, "project-item-id", "", "link the new action under this project item")
	fs.StringVar(&f.projectItemID, "project-item", "", "alias for --project-item-id")
	fs.StringVar(&f.projectItemID, "project", "", "alias for --project-item-id")
	fs.StringVar(&f.projectRole, "role", "", "project link role: next_action (default), support, blocked_by")
	fs.StringVar(&f.source, "source", "", "source backend (todoist, markdown, github, ...)")
	fs.StringVar(&f.sourceRef, "source-ref", "", "stable upstream identifier; required for non-todoist sources")
	if err := fs.Parse(flagArgs); err != nil {
		return gtdCaptureFlags{}, fmt.Errorf("%w: %v", errGtdUsage, err)
	}
	if rest := fs.Args(); len(rest) > 0 {
		titleParts = append(titleParts, rest...)
	}
	f.title = strings.TrimSpace(strings.Join(titleParts, " "))
	if f.title == "" {
		return gtdCaptureFlags{}, fmt.Errorf("%w: title is required", errGtdUsage)
	}
	return f, nil
}

// splitGtdCaptureArgs separates flag tokens from title tokens so the user can
// place the title before, after, or between flags. The flag package alone
// stops parsing at the first positional, which would force `--flag` after a
// quoted multi-word title — awkward for a quick-capture CLI.
func splitGtdCaptureArgs(args []string) ([]string, []string) {
	flagArgs := make([]string, 0, len(args))
	title := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			title = append(title, arg)
			continue
		}
		flagArgs = append(flagArgs, arg)
		if strings.Contains(arg, "=") {
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			flagArgs = append(flagArgs, args[i+1])
			i++
		}
	}
	return flagArgs, title
}

func captureRequestBody(f gtdCaptureFlags) (map[string]any, error) {
	body := map[string]any{"title": f.title}
	if v := strings.TrimSpace(f.kind); v != "" {
		body["kind"] = v
	}
	if v := strings.TrimSpace(f.sphere); v != "" {
		body["sphere"] = v
	}
	if err := assignPositiveInt64Body(body, "workspace_id", "workspace", f.workspaceID); err != nil {
		return nil, err
	}
	if err := assignPositiveInt64Body(body, "actor_id", "actor-id", f.actorID); err != nil {
		return nil, err
	}
	if err := assignPositiveInt64Body(body, "label_id", "label-id", f.labelID); err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(f.label); v != "" {
		body["label"] = v
	}
	if err := assignPositiveInt64Body(body, "project_item_id", "project-item-id", f.projectItemID); err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(f.projectRole); v != "" {
		body["project_item_role"] = v
	}
	if v := strings.TrimSpace(f.source); v != "" {
		body["source"] = v
	}
	if v := strings.TrimSpace(f.sourceRef); v != "" {
		body["source_ref"] = v
	}
	return body, nil
}

func assignPositiveInt64Body(body map[string]any, key, flagName, raw string) error {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil
	}
	value, err := parsePositiveInt64(clean, flagName)
	if err != nil {
		return err
	}
	body[key] = value
	return nil
}

func formatGtdCaptureResult(resp gtdCaptureResponse) string {
	parts := []string{
		fmt.Sprintf("captured #%d", resp.Item.ID),
		fmt.Sprintf("kind=%s", resp.Item.Kind),
		fmt.Sprintf("state=%s", resp.Item.State),
	}
	if resp.Item.Sphere != "" {
		parts = append(parts, fmt.Sprintf("sphere=%s", resp.Item.Sphere))
	}
	if src := optionalStringPtr(resp.Item.Source); src != "" {
		parts = append(parts, fmt.Sprintf("source=%s", src))
	}
	if resp.Label != nil {
		parts = append(parts, fmt.Sprintf("label=%s", resp.Label.Name))
	}
	if resp.ProjectItem != nil {
		parts = append(parts, fmt.Sprintf("project_item=%d", resp.ProjectItem.ProjectItemID))
		if resp.ProjectItem.Role != "" {
			parts = append(parts, fmt.Sprintf("role=%s", resp.ProjectItem.Role))
		}
	}
	parts = append(parts, fmt.Sprintf("title=%q", resp.Item.Title))
	return strings.Join(parts, " ")
}
