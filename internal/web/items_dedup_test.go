package web

import (
	"net/http"
	"testing"

	"github.com/sloppy-org/slopshell/internal/store"
)

func TestItemDedupReviewAPIListsCandidateGroups(t *testing.T) {
	app := newAuthedTestApp(t)
	first, second := seedWebDedupItems(t, app)
	if _, err := app.store.CreateItemDedupCandidate(store.ItemDedupCandidateOptions{
		Kind:       store.ItemKindAction,
		Confidence: 0.88,
		Outcome:    "Finish review queue",
		Reasoning:  "local LLM matched source context",
		Detector:   "local-llm",
		Items: []store.ItemDedupCandidateItemInput{
			{ItemID: first.ID, Outcome: "Finish the review queue"},
			{ItemID: second.ID, Outcome: "Complete review queue"},
		},
	}); err != nil {
		t.Fatalf("CreateItemDedupCandidate() error: %v", err)
	}

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodGet, "/api/items/dedup?kind=action", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	payload := decodeJSONResponse(t, rr)
	groups, ok := payload["groups"].([]any)
	if !ok || len(groups) != 1 {
		t.Fatalf("groups = %#v, want one group", payload["groups"])
	}
	group := groups[0].(map[string]any)
	if group["reasoning"] != "local LLM matched source context" {
		t.Fatalf("reasoning = %v", group["reasoning"])
	}
	members := group["items"].([]any)
	for _, raw := range members {
		member := raw.(map[string]any)
		if len(member["source_bindings"].([]any)) != 1 {
			t.Fatalf("member bindings = %#v, want source binding", member)
		}
	}
}

func TestItemDedupReviewAPIActions(t *testing.T) {
	app := newAuthedTestApp(t)
	first, second := seedWebDedupItems(t, app)
	group, err := app.store.CreateItemDedupCandidate(store.ItemDedupCandidateOptions{
		Kind:  store.ItemKindAction,
		Items: []store.ItemDedupCandidateItemInput{{ItemID: first.ID}, {ItemID: second.ID}},
	})
	if err != nil {
		t.Fatalf("CreateItemDedupCandidate() error: %v", err)
	}

	later := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/dedup/"+itoa(group.ID)+"/review_later", map[string]any{})
	if later.Code != http.StatusOK {
		t.Fatalf("review_later status = %d, want 200: %s", later.Code, later.Body.String())
	}
	list := doAuthedJSONRequest(t, app.Router(), http.MethodGet, "/api/items/dedup", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", list.Code, list.Body.String())
	}
	groups := decodeJSONResponse(t, list)["groups"].([]any)
	if len(groups) != 1 || groups[0].(map[string]any)["state"] != store.ItemDedupStateReviewLater {
		t.Fatalf("groups after review_later = %#v", groups)
	}

	keep := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/items/dedup/"+itoa(group.ID)+"/keep_separate", map[string]any{})
	if keep.Code != http.StatusOK {
		t.Fatalf("keep_separate status = %d, want 200: %s", keep.Code, keep.Body.String())
	}
	empty := doAuthedJSONRequest(t, app.Router(), http.MethodGet, "/api/items/dedup", nil)
	if empty.Code != http.StatusOK {
		t.Fatalf("empty status = %d, want 200: %s", empty.Code, empty.Body.String())
	}
	emptyPayload := decodeJSONResponse(t, empty)
	groupsAfterKeep, ok := emptyPayload["groups"].([]any)
	if !ok {
		t.Fatalf("groups after keep_separate missing: %#v", emptyPayload)
	}
	if got := len(groupsAfterKeep); got != 0 {
		t.Fatalf("open groups after keep_separate = %d, want 0", got)
	}
}

func seedWebDedupItems(t *testing.T, app *App) (store.Item, store.Item) {
	t.Helper()
	first := seedWebDedupItem(t, app, "Review candidate group", store.ExternalProviderTodoist, "task-1", "Inbox")
	second := seedWebDedupItem(t, app, "Review candidate duplicate", store.ExternalProviderGmail, "msg-2", "Follow-up")
	return first, second
}

func seedWebDedupItem(t *testing.T, app *App, title, provider, remoteID, container string) store.Item {
	t.Helper()
	item, err := app.store.CreateItem(title, store.ItemOptions{State: store.ItemStateNext})
	if err != nil {
		t.Fatalf("CreateItem(%q) error: %v", title, err)
	}
	account, err := app.store.CreateExternalAccount(store.SphereWork, provider, provider+" "+remoteID, map[string]any{})
	if err != nil {
		t.Fatalf("CreateExternalAccount(%s) error: %v", provider, err)
	}
	if _, err := app.store.UpsertExternalBinding(store.ExternalBinding{
		AccountID:    account.ID,
		Provider:     account.Provider,
		ObjectType:   "task",
		RemoteID:     remoteID,
		ItemID:       &item.ID,
		ContainerRef: &container,
	}); err != nil {
		t.Fatalf("UpsertExternalBinding(%s) error: %v", remoteID, err)
	}
	return item
}
