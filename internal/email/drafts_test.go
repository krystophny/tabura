package email

import "testing"

func TestNormalizeDraftInputAllowsBlankDrafts(t *testing.T) {
	normalized, err := NormalizeDraftInput(DraftInput{
		Subject: "  Draft subject  ",
		Body:    "draft body\r\n",
	})
	if err != nil {
		t.Fatalf("NormalizeDraftInput() error: %v", err)
	}
	if normalized.Subject != "Draft subject" {
		t.Fatalf("subject = %q, want Draft subject", normalized.Subject)
	}
	if normalized.Body != "draft body" {
		t.Fatalf("body = %q, want draft body", normalized.Body)
	}
}

func TestNormalizeDraftSendInputRequiresRecipient(t *testing.T) {
	if _, err := NormalizeDraftSendInput(DraftInput{Subject: "Draft"}); err == nil {
		t.Fatal("NormalizeDraftSendInput() error = nil, want recipient validation")
	}
}
