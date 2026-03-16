package web

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krystophny/tabura/internal/store"
	tabsync "github.com/krystophny/tabura/internal/sync"
)

func TestSourcePushManagerStartsWatcherAndTriggersSync(t *testing.T) {
	app := newAuthedTestApp(t)
	_, err := app.store.CreateExternalAccount(store.SphereWork, store.ExternalProviderExchangeEWS, "TU Graz", map[string]any{
		"endpoint": "https://exchange.tugraz.at/EWS/Exchange.asmx",
		"username": "ert",
	})
	if err != nil {
		t.Fatalf("CreateExternalAccount(exchange_ews) error: %v", err)
	}

	var syncCalls atomic.Int32
	watchStarted := make(chan struct{}, 1)
	provider := &accountSyncProvider{
		name: store.ExternalProviderExchangeEWS,
		syncAccount: func(context.Context, store.ExternalAccount) (int, error) {
			syncCalls.Add(1)
			return 1, nil
		},
		syncPolicy: func(context.Context, store.ExternalAccount) (tabsync.SyncPolicy, error) {
			return tabsync.SyncPolicy{DisablePoll: true}, nil
		},
		watchAccount: func(ctx context.Context, _ store.ExternalAccount, trigger func()) error {
			select {
			case watchStarted <- struct{}{}:
			default:
			}
			trigger()
			<-ctx.Done()
			return ctx.Err()
		},
	}

	manager, ok := newSourcePushManager(app, []*accountSyncProvider{provider}).(*sourcePushManager)
	if !ok || manager == nil {
		t.Fatal("expected source push manager")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.reconcile(ctx)

	select {
	case <-watchStarted:
	case <-time.After(time.Second):
		t.Fatal("push watcher did not start")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if syncCalls.Load() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("syncCalls = %d, want at least 1", syncCalls.Load())
}
