package web

import (
	"context"
	"strings"
	"time"

	"github.com/krystophny/tabura/internal/ews"
	"github.com/krystophny/tabura/internal/store"
	tabsync "github.com/krystophny/tabura/internal/sync"
)

func (a *App) exchangeEWSSourceSyncPolicy(context.Context, store.ExternalAccount) (tabsync.SyncPolicy, error) {
	return tabsync.SyncPolicy{
		DisablePoll:      true,
		FallbackInterval: 30 * time.Minute,
	}, nil
}

func (a *App) watchExchangeEWSSourceAccount(ctx context.Context, account store.ExternalAccount, trigger func()) error {
	if a == nil {
		return nil
	}
	client, err := a.exchangeEWSClientForAccount(ctx, account)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Watch(ctx, ews.WatchOptions{
		SubscribeToAllFolders: true,
		ConnectionTimeout:     29 * time.Minute,
	}, func(batch ews.StreamBatch) error {
		if len(batch.Events) == 0 {
			return nil
		}
		if a.emailRefreshes != nil {
			ids := make([]string, 0, len(batch.Events))
			for _, event := range batch.Events {
				if itemID := strings.TrimSpace(event.ItemID); itemID != "" {
					ids = append(ids, itemID)
					continue
				}
				if oldItemID := strings.TrimSpace(event.OldItemID); oldItemID != "" {
					ids = append(ids, oldItemID)
				}
			}
			a.emailRefreshes.add(account.ID, ids...)
		}
		trigger()
		return nil
	})
}
