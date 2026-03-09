package web

import (
	"context"
	"net/http"
	"testing"
	"time"

	tabsync "github.com/krystophny/tabura/internal/sync"
)

type stubSourceSyncRunner struct {
	runOnceCount  int
	runNowCount   int
	runOnceFn     func()
	runOnceResult tabsync.RunResult
	runNowResult  tabsync.RunResult
	runOnceErr    error
	runNowErr     error
}

func (s *stubSourceSyncRunner) RunOnce(context.Context) (tabsync.RunResult, error) {
	s.runOnceCount++
	if s.runOnceFn != nil {
		s.runOnceFn()
	}
	return s.runOnceResult, s.runOnceErr
}

func (s *stubSourceSyncRunner) RunNow(context.Context) (tabsync.RunResult, error) {
	s.runNowCount++
	return s.runNowResult, s.runNowErr
}

func TestSourcePollerLoopRunsRunnerUntilCanceled(t *testing.T) {
	app := newAuthedTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := &stubSourceSyncRunner{
		runOnceResult: tabsync.RunResult{NextDelay: 10 * time.Millisecond},
	}
	runner.runOnceFn = func() {
		if runner.runOnceCount >= 2 {
			cancel()
		}
	}
	app.sourceSync = runner

	done := make(chan struct{})
	go func() {
		defer close(done)
		app.runSourcePoller(ctx)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runSourcePoller did not stop after cancel")
	}
	if runner.runOnceCount < 2 {
		t.Fatalf("runOnceCount = %d, want at least 2", runner.runOnceCount)
	}
}

func TestSyncNowCommandForcesImmediateRun(t *testing.T) {
	app := newAuthedTestApp(t)
	project, err := app.ensureDefaultProjectRecord()
	if err != nil {
		t.Fatalf("ensureDefaultProjectRecord: %v", err)
	}
	session, err := app.store.GetOrCreateChatSession(project.ProjectKey)
	if err != nil {
		t.Fatalf("GetOrCreateChatSession: %v", err)
	}
	app.sourceSync = &stubSourceSyncRunner{
		runNowResult: tabsync.RunResult{
			Accounts: []tabsync.AccountResult{
				{AccountID: 1, Provider: "todoist", Label: "main"},
				{AccountID: 2, Provider: "bear", Label: "notes", Skipped: true, Reason: "interval"},
			},
		},
	}

	message, payloads, handled := app.classifyAndExecuteSystemAction(context.Background(), session.ID, session, "sync now")
	if !handled {
		t.Fatal("expected sync now to be handled")
	}
	if message != "Polled 2 external source account(s); 1 synced, 1 skipped." {
		t.Fatalf("message = %q, want poll summary", message)
	}
	if len(payloads) != 1 {
		t.Fatalf("payload count = %d, want 1", len(payloads))
	}
	if got := strFromAny(payloads[0]["type"]); got != "sync_sources" {
		t.Fatalf("payload type = %q, want sync_sources", got)
	}
	if got := intFromAny(payloads[0]["synced_accounts"], 0); got != 1 {
		t.Fatalf("synced_accounts = %d, want 1", got)
	}
	if got := intFromAny(payloads[0]["skipped_accounts"], 0); got != 1 {
		t.Fatalf("skipped_accounts = %d, want 1", got)
	}
	runner, _ := app.sourceSync.(*stubSourceSyncRunner)
	if runner.runNowCount != 1 {
		t.Fatalf("runNowCount = %d, want 1", runner.runNowCount)
	}
}

func TestBroadcastItemsIngestedWebsocketNotification(t *testing.T) {
	app := newAuthedTestApp(t)

	project, err := app.ensureDefaultProjectRecord()
	if err != nil {
		t.Fatalf("ensureDefaultProjectRecord: %v", err)
	}
	session, err := app.store.GetOrCreateChatSession(project.ProjectKey)
	if err != nil {
		t.Fatalf("GetOrCreateChatSession() error: %v", err)
	}
	conn, clientConn, cleanup := newParticipantTestWSConn(t)
	defer cleanup()
	app.hub.registerChat(session.ID, conn)
	defer app.hub.unregisterChat(session.ID, conn)

	app.broadcastItemsIngested(2, "todoist")

	payload := waitForWSJSONMessageType(t, clientConn, 2*time.Second, "items_ingested")
	if got, ok := payload["count"].(float64); !ok || int(got) != 2 {
		t.Fatalf("items_ingested count = %#v, want 2", payload["count"])
	}
	if got := strFromAny(payload["source"]); got != "todoist" {
		t.Fatalf("items_ingested source = %q, want todoist", got)
	}
}

func TestRouterRejectsIngestEndpoint(t *testing.T) {
	app := newAuthedTestApp(t)

	rr := doAuthedJSONRequest(t, app.Router(), http.MethodPost, "/api/ingest", map[string]any{"items": []any{}})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("POST /api/ingest status = %d, want 404", rr.Code)
	}
}
