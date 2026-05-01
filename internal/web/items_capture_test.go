package web

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sloppy-org/slopshell/internal/store"
)

func TestItemCaptureCreatesActionInInbox(t *testing.T) {
	app := newAuthedTestApp(t)

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title": "Reply to grant request",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rr.Code, rr.Body.String())
	}
	data := decodeJSONDataResponse(t, rr)
	itemPayload, ok := data["item"].(map[string]any)
	if !ok {
		t.Fatalf("missing item payload: %#v", data)
	}
	if got := strFromAny(itemPayload["title"]); got != "Reply to grant request" {
		t.Fatalf("title = %q, want Reply to grant request", got)
	}
	if got := strFromAny(itemPayload["kind"]); got != store.ItemKindAction {
		t.Fatalf("kind = %q, want %s", got, store.ItemKindAction)
	}
	if got := strFromAny(itemPayload["state"]); got != store.ItemStateInbox {
		t.Fatalf("state = %q, want inbox", got)
	}
	stored := mustFirstItemByState(t, app, store.ItemStateInbox)
	if stored.Kind != store.ItemKindAction {
		t.Fatalf("stored kind = %q, want action", stored.Kind)
	}
}

func TestItemCaptureCreatesProjectItemWhenKindProject(t *testing.T) {
	app := newAuthedTestApp(t)

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title": "Ship dialog refresh",
		"kind":  store.ItemKindProject,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rr.Code, rr.Body.String())
	}
	stored := mustFirstItemByState(t, app, store.ItemStateInbox)
	if stored.Kind != store.ItemKindProject {
		t.Fatalf("stored kind = %q, want project", stored.Kind)
	}
}

func TestItemCaptureLinksUnderProjectItem(t *testing.T) {
	app := newAuthedTestApp(t)
	parent, err := app.store.CreateItem("Mentorship", store.ItemOptions{Kind: store.ItemKindProject})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title":             "Schedule first 1:1",
		"project_item_id":   parent.ID,
		"project_item_role": store.ItemLinkRoleNextAction,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rr.Code, rr.Body.String())
	}
	data := decodeJSONDataResponse(t, rr)
	link, ok := data["project_item"].(map[string]any)
	if !ok {
		t.Fatalf("missing project_item payload: %#v", data)
	}
	if int64(toFloat64(t, link["project_item_id"])) != parent.ID {
		t.Fatalf("project_item_id = %v, want %d", link["project_item_id"], parent.ID)
	}
	if got := strFromAny(link["role"]); got != store.ItemLinkRoleNextAction {
		t.Fatalf("role = %q, want %s", got, store.ItemLinkRoleNextAction)
	}
	links, err := app.store.ListItemChildLinks(parent.ID)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("links len = %d, want 1", len(links))
	}
}

func TestItemCaptureRejectsProjectLinkOnProjectKind(t *testing.T) {
	app := newAuthedTestApp(t)
	parent, err := app.store.CreateItem("Outcome", store.ItemOptions{Kind: store.ItemKindProject})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title":           "Sub-outcome",
		"kind":            store.ItemKindProject,
		"project_item_id": parent.ID,
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "project_item_id cannot be set") {
		t.Fatalf("body missing diagnostic: %s", rr.Body.String())
	}
}

func TestItemCaptureRejectsEmptyTitle(t *testing.T) {
	app := newAuthedTestApp(t)
	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title": "   ",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "title is required") {
		t.Fatalf("body missing diagnostic: %s", rr.Body.String())
	}
}

func TestItemCaptureRejectsUnknownKind(t *testing.T) {
	app := newAuthedTestApp(t)
	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title": "Plan retreat",
		"kind":  "epic",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "kind must be") {
		t.Fatalf("body missing diagnostic: %s", rr.Body.String())
	}
}

func TestItemCaptureRoutesSphereByWorkspace(t *testing.T) {
	app := newAuthedTestApp(t)
	workspace, err := app.store.CreateWorkspace("Plasma research", filepath.Join(t.TempDir(), "plasma"))
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := app.store.SetWorkspaceSphere(workspace.ID, store.SphereWork); err != nil {
		t.Fatalf("set workspace sphere: %v", err)
	}

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title":        "Wire diagnostic loop",
		"workspace_id": workspace.ID,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rr.Code, rr.Body.String())
	}
	stored := mustFirstItemByState(t, app, store.ItemStateInbox)
	if stored.Sphere != store.SphereWork {
		t.Fatalf("sphere = %q, want work", stored.Sphere)
	}
	if stored.WorkspaceID == nil || *stored.WorkspaceID != workspace.ID {
		t.Fatalf("workspace_id = %v, want %d", stored.WorkspaceID, workspace.ID)
	}
}

func TestItemCaptureCreatesAndLinksLabelByName(t *testing.T) {
	app := newAuthedTestApp(t)
	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title": "Read deferred docs",
		"label": "reading",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rr.Code, rr.Body.String())
	}
	data := decodeJSONDataResponse(t, rr)
	label, ok := data["label"].(map[string]any)
	if !ok {
		t.Fatalf("missing label payload: %#v", data)
	}
	if got := strFromAny(label["name"]); got != "reading" {
		t.Fatalf("label name = %q, want reading", got)
	}
}

func TestItemCaptureRejectsLabelIdAndLabelTogether(t *testing.T) {
	app := newAuthedTestApp(t)
	label, err := app.store.CreateLabel("focus", nil)
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title":    "Plan retreat",
		"label":    "focus",
		"label_id": label.ID,
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
	}
}

func TestItemCaptureRejectsMissingProject(t *testing.T) {
	app := newAuthedTestApp(t)
	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/capture", map[string]any{
		"title":           "Stranded",
		"project_item_id": 999,
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
	}
	items, err := app.store.ListItemsByState(store.ItemStateInbox)
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("inbox len = %d, want 0 (rolled back)", len(items))
	}
}

func toFloat64(t *testing.T, value any) float64 {
	t.Helper()
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		t.Fatalf("expected numeric value, got %T (%v)", value, value)
		return 0
	}
}
