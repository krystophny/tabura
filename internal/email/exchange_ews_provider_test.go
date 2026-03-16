package email

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExchangeEWSDraftLifecycle(t *testing.T) {
	t.Helper()
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		action := strings.Trim(r.Header.Get("SOAPAction"), `"`)
		actions = append(actions, action)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		switch {
		case strings.HasSuffix(action, "/CreateItem"):
			if !strings.Contains(string(body), "MimeContent") {
				t.Fatalf("CreateItem body missing MimeContent: %s", string(body))
			}
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages" xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Body>
    <m:CreateItemResponse>
      <m:ResponseMessages>
        <m:CreateItemResponseMessage ResponseClass="Success">
          <m:ResponseCode>NoError</m:ResponseCode>
          <m:Items>
            <t:Message>
              <t:ItemId Id="draft-1" ChangeKey="ck1" />
              <t:ConversationId Id="thread-1" />
              <t:Subject>Hello</t:Subject>
            </t:Message>
          </m:Items>
        </m:CreateItemResponseMessage>
      </m:ResponseMessages>
    </m:CreateItemResponse>
  </soap:Body>
</soap:Envelope>`)
		case strings.HasSuffix(action, "/UpdateItem"):
			if !strings.Contains(string(body), "MimeContent") {
				t.Fatalf("UpdateItem body missing MimeContent: %s", string(body))
			}
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Body>
    <m:UpdateItemResponse>
      <m:ResponseMessages>
        <m:UpdateItemResponseMessage ResponseClass="Success">
          <m:ResponseCode>NoError</m:ResponseCode>
        </m:UpdateItemResponseMessage>
      </m:ResponseMessages>
    </m:UpdateItemResponse>
  </soap:Body>
</soap:Envelope>`)
		case strings.HasSuffix(action, "/GetItem"):
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages" xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Body>
    <m:GetItemResponse>
      <m:ResponseMessages>
        <m:GetItemResponseMessage ResponseClass="Success">
          <m:ResponseCode>NoError</m:ResponseCode>
          <m:Items>
            <t:Message>
              <t:ItemId Id="draft-1" ChangeKey="ck2" />
              <t:ConversationId Id="thread-1" />
              <t:Subject>Hello updated</t:Subject>
            </t:Message>
          </m:Items>
        </m:GetItemResponseMessage>
      </m:ResponseMessages>
    </m:GetItemResponse>
  </soap:Body>
</soap:Envelope>`)
		case strings.HasSuffix(action, "/SendItem"):
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Body>
    <m:SendItemResponse>
      <m:ResponseMessages>
        <m:SendItemResponseMessage ResponseClass="Success">
          <m:ResponseCode>NoError</m:ResponseCode>
        </m:SendItemResponseMessage>
      </m:ResponseMessages>
    </m:SendItemResponse>
  </soap:Body>
</soap:Envelope>`)
		default:
			t.Fatalf("unexpected SOAP action %q body=%s", action, string(body))
		}
	}))
	defer server.Close()

	provider, err := NewExchangeEWSMailProvider(ExchangeEWSConfig{
		Endpoint: server.URL,
		Username: "ert",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("NewExchangeEWSMailProvider() error: %v", err)
	}
	defer provider.Close()

	draft, err := provider.CreateDraft(t.Context(), DraftInput{
		From:    "ert@example.test",
		To:      []string{"alice@example.test"},
		Subject: "Hello",
		Body:    "Body",
	})
	if err != nil {
		t.Fatalf("CreateDraft() error: %v", err)
	}
	if draft.ID != "draft-1" || draft.ThreadID != "thread-1" {
		t.Fatalf("draft = %#v", draft)
	}

	updated, err := provider.UpdateDraft(t.Context(), draft.ID, DraftInput{
		From:    "ert@example.test",
		To:      []string{"alice@example.test"},
		Subject: "Hello updated",
		Body:    "Body updated",
	})
	if err != nil {
		t.Fatalf("UpdateDraft() error: %v", err)
	}
	if updated.ID != "draft-1" || updated.ThreadID != "thread-1" {
		t.Fatalf("updated draft = %#v", updated)
	}

	if err := provider.SendDraft(t.Context(), draft.ID, DraftInput{
		From:    "ert@example.test",
		To:      []string{"alice@example.test"},
		Subject: "Hello updated",
		Body:    "Body updated",
	}); err != nil {
		t.Fatalf("SendDraft() error: %v", err)
	}

	want := []string{
		"http://schemas.microsoft.com/exchange/services/2006/messages/CreateItem",
		"http://schemas.microsoft.com/exchange/services/2006/messages/UpdateItem",
		"http://schemas.microsoft.com/exchange/services/2006/messages/GetItem",
		"http://schemas.microsoft.com/exchange/services/2006/messages/UpdateItem",
		"http://schemas.microsoft.com/exchange/services/2006/messages/GetItem",
		"http://schemas.microsoft.com/exchange/services/2006/messages/SendItem",
	}
	if strings.Join(actions, "\n") != strings.Join(want, "\n") {
		t.Fatalf("actions = %#v, want %#v", actions, want)
	}
}

func TestExchangeEWSDisplayFolderNameNormalizesArchiveDisplay(t *testing.T) {
	tests := map[string]string{
		"Archive":               "",
		"Archive/padova2023":    "padova2023",
		`Archive\simons24`:      "simons24",
		"Posteingang":           "Posteingang",
		"Archive/work/projectA": "projectA",
	}
	for input, want := range tests {
		if got := exchangeEWSDisplayFolderName(input); got != want {
			t.Fatalf("exchangeEWSDisplayFolderName(%q) = %q, want %q", input, got, want)
		}
	}
}
