package web

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestHandleChatSessionCancelStopsActiveTurn(t *testing.T) {
	app, err := New(t.TempDir(), "", "", "", false)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	if err := app.store.AddAuthSession("token-test"); err != nil {
		t.Fatalf("add auth session: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Shutdown(context.Background())
	})

	session, err := app.store.GetOrCreateChatSession("cancel-test-project")
	if err != nil {
		t.Fatalf("create chat session: %v", err)
	}

	cancelCalled := make(chan struct{}, 1)
	app.registerActiveChatTurn(session.ID, "run-1", func() {
		select {
		case cancelCalled <- struct{}{}:
		default:
		}
	})

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/chat/sessions/"+session.ID+"/cancel", map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	select {
	case <-cancelCalled:
	default:
		t.Fatalf("expected active chat turn cancel func to be invoked")
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := intFromAny(payload["canceled"], -1); got != 1 {
		t.Fatalf("expected canceled=1, got %v", payload["canceled"])
	}
}
