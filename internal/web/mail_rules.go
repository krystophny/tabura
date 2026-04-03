package web

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/krystophny/slopshell/internal/ews"
	"github.com/krystophny/slopshell/internal/store"
)

type mailRuleUpsertRequest struct {
	Rule ews.Rule `json:"rule"`
}

func (a *App) handleMailRulesList(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}
	account, client, err := a.exchangeEWSClientForRoute(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer client.Close()
	rules, err := client.GetInboxRules(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeAPIData(w, http.StatusOK, map[string]any{
		"account":                account,
		"rules":                  rules,
		"server_rules_supported": true,
	})
}

func (a *App) handleMailRuleCreate(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}
	_, client, err := a.exchangeEWSClientForRoute(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer client.Close()
	var req mailRuleUpsertRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.Rule.Name) == "" {
		writeAPIError(w, http.StatusBadRequest, "rule name is required")
		return
	}
	if err := client.UpdateInboxRules(r.Context(), []ews.RuleOperation{{Kind: ews.RuleOperationCreate, Rule: req.Rule}}); err != nil {
		writeAPIError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (a *App) handleMailRuleUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}
	_, client, err := a.exchangeEWSClientForRoute(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer client.Close()
	var req mailRuleUpsertRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	ruleID := strings.TrimSpace(chi.URLParam(r, "rule_id"))
	if ruleID == "" {
		writeAPIError(w, http.StatusBadRequest, "rule_id is required")
		return
	}
	req.Rule.ID = ruleID
	if strings.TrimSpace(req.Rule.Name) == "" {
		writeAPIError(w, http.StatusBadRequest, "rule name is required")
		return
	}
	if err := client.UpdateInboxRules(r.Context(), []ews.RuleOperation{{Kind: ews.RuleOperationSet, Rule: req.Rule}}); err != nil {
		writeAPIError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (a *App) handleMailRuleDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}
	_, client, err := a.exchangeEWSClientForRoute(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer client.Close()
	ruleID := strings.TrimSpace(chi.URLParam(r, "rule_id"))
	if ruleID == "" {
		writeAPIError(w, http.StatusBadRequest, "rule_id is required")
		return
	}
	if err := client.UpdateInboxRules(r.Context(), []ews.RuleOperation{{Kind: ews.RuleOperationDelete, Rule: ews.Rule{ID: ruleID}}}); err != nil {
		writeAPIError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeNoContent(w)
}

func (a *App) exchangeEWSClientForRoute(r *http.Request) (store.ExternalAccount, *ews.Client, error) {
	accountID, err := parseURLInt64Param(r, "account_id")
	if err != nil {
		return store.ExternalAccount{}, nil, err
	}
	account, err := a.store.GetExternalAccount(accountID)
	if err != nil {
		return store.ExternalAccount{}, nil, err
	}
	if account.Provider != store.ExternalProviderExchangeEWS {
		return store.ExternalAccount{}, nil, errBadRequest("mail rules are only supported for exchange_ews accounts")
	}
	client, err := a.exchangeEWSClientForAccount(r.Context(), account)
	if err != nil {
		return store.ExternalAccount{}, nil, err
	}
	return account, client, nil
}

func errBadRequest(message string) error {
	return &requestError{message: message}
}

type requestError struct {
	message string
}

func (e *requestError) Error() string {
	return e.message
}
