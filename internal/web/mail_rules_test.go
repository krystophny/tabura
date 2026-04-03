package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krystophny/sloppad/internal/store"
)

func TestMailRulesAPIListsAndUpdatesExchangeEWSRules(t *testing.T) {
	var updateSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		switch r.Header.Get("SOAPAction") {
		case `"http://schemas.microsoft.com/exchange/services/2006/messages/GetInboxRules"`:
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages" xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Body>
    <m:GetInboxRulesResponse>
      <m:ResponseCode>NoError</m:ResponseCode>
      <m:InboxRules>
        <t:Rule>
          <t:RuleId>rule-1</t:RuleId>
          <t:DisplayName>Project Mail</t:DisplayName>
          <t:Priority>1</t:Priority>
          <t:IsEnabled>true</t:IsEnabled>
          <t:Conditions>
            <t:ContainsSubjectStrings><t:String>project</t:String></t:ContainsSubjectStrings>
          </t:Conditions>
          <t:Actions>
            <t:StopProcessingRules>true</t:StopProcessingRules>
          </t:Actions>
        </t:Rule>
      </m:InboxRules>
    </m:GetInboxRulesResponse>
  </soap:Body>
</soap:Envelope>`)
		case `"http://schemas.microsoft.com/exchange/services/2006/messages/UpdateInboxRules"`:
			updateSeen = true
			if string(body) == "" {
				t.Fatal("UpdateInboxRules request body was empty")
			}
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Body>
    <m:UpdateInboxRulesResponse>
      <m:ResponseCode>NoError</m:ResponseCode>
    </m:UpdateInboxRulesResponse>
  </soap:Body>
</soap:Envelope>`)
		default:
			t.Fatalf("unexpected SOAPAction %q", r.Header.Get("SOAPAction"))
		}
	}))
	defer server.Close()

	app := newAuthedTestApp(t)
	account, err := app.store.CreateExternalAccount(store.SphereWork, store.ExternalProviderExchangeEWS, "TU Graz Exchange", map[string]any{
		"endpoint": server.URL,
		"username": "ert",
	})
	if err != nil {
		t.Fatalf("CreateExternalAccount() error: %v", err)
	}
	t.Setenv(store.ExternalAccountPasswordEnvVar(account.Provider, account.AccountName), "secret")

	rrList := doAuthedJSONRequest(t, app.Router(), http.MethodGet, "/api/external-accounts/"+itoa(account.ID)+"/mail-rules", nil)
	if rrList.Code != http.StatusOK {
		t.Fatalf("GET mail-rules status = %d: %s", rrList.Code, rrList.Body.String())
	}
	listPayload := decodeJSONDataResponse(t, rrList)
	rules, ok := listPayload["rules"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("rules payload = %#v", listPayload)
	}

	rrUpdate := doAuthedJSONRequest(t, app.Router(), http.MethodPut, "/api/external-accounts/"+itoa(account.ID)+"/mail-rules/rule-1", map[string]any{
		"rule": map[string]any{
			"name":     "Project Mail",
			"priority": 1,
			"enabled":  true,
			"conditions": map[string]any{
				"contains_subject_strings": []string{"project"},
			},
			"actions": map[string]any{
				"stop_processing_rules": true,
			},
		},
	})
	if rrUpdate.Code != http.StatusOK {
		t.Fatalf("PUT mail-rules status = %d: %s", rrUpdate.Code, rrUpdate.Body.String())
	}
	if !updateSeen {
		t.Fatal("expected UpdateInboxRules to be called")
	}
}
