package web

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type itemDedupActionRequest struct {
	CanonicalItemID *int64 `json:"canonical_item_id,omitempty"`
}

func (a *App) handleItemDedupReview(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}
	if !a.resurfaceDueItemsForRead(w) {
		return
	}
	filter, err := parseItemListFilterQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	groups, err := a.store.ListItemDedupCandidatesFiltered(r.URL.Query().Get("kind"), filter)
	if err != nil {
		writeItemStoreError(w, err)
		return
	}
	writeAPIData(w, http.StatusOK, map[string]any{
		"groups": groups,
		"total":  len(groups),
	})
}

func (a *App) handleItemDedupAction(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}
	candidateID, err := parseURLInt64Param(r, "candidate_id")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req itemDedupActionRequest
	if r.Body != nil && strings.EqualFold(r.Method, http.MethodPost) {
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
	}
	action := strings.TrimSpace(chi.URLParam(r, "action"))
	group, err := a.store.ApplyItemDedupDecision(candidateID, action, req.CanonicalItemID)
	if err != nil {
		writeItemStoreError(w, err)
		return
	}
	writeAPIData(w, http.StatusOK, map[string]any{
		"group":  group,
		"action": action,
	})
}
