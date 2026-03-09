package web

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/krystophny/tabura/internal/email"
	"github.com/krystophny/tabura/internal/providerdata"
	"github.com/krystophny/tabura/internal/store"
)

type fakeEmailSyncProvider struct {
	listFunc  func(email.SearchOptions) ([]string, error)
	messages  map[string]*providerdata.EmailMessage
	listCalls []email.SearchOptions
}

func (f *fakeEmailSyncProvider) ListMessages(_ context.Context, opts email.SearchOptions) ([]string, error) {
	f.listCalls = append(f.listCalls, opts)
	if f.listFunc == nil {
		return nil, nil
	}
	return f.listFunc(opts)
}

func (f *fakeEmailSyncProvider) GetMessages(_ context.Context, messageIDs []string, _ string) ([]*providerdata.EmailMessage, error) {
	out := make([]*providerdata.EmailMessage, 0, len(messageIDs))
	for _, id := range messageIDs {
		if message, ok := f.messages[id]; ok {
			out = append(out, message)
		}
	}
	return out, nil
}

func (f *fakeEmailSyncProvider) Close() error {
	return nil
}

func TestSourceSyncRunnerPollsGmailAndIMAPAccounts(t *testing.T) {
	app := newAuthedTestApp(t)

	gmailAccount, err := app.store.CreateExternalAccount(store.SphereWork, store.ExternalProviderGmail, "Work Gmail", nil)
	if err != nil {
		t.Fatalf("CreateExternalAccount(gmail) error: %v", err)
	}
	imapAccount, err := app.store.CreateExternalAccount(store.SpherePrivate, store.ExternalProviderIMAP, "Private IMAP", nil)
	if err != nil {
		t.Fatalf("CreateExternalAccount(imap) error: %v", err)
	}

	gmailProvider := &fakeEmailSyncProvider{
		listFunc: func(opts email.SearchOptions) ([]string, error) {
			switch {
			case opts.IsRead != nil && !*opts.IsRead:
				return []string{"gmail-1"}, nil
			case opts.IsFlagged != nil && *opts.IsFlagged:
				return nil, nil
			case !opts.Since.IsZero():
				return []string{"gmail-1"}, nil
			default:
				return nil, nil
			}
		},
		messages: map[string]*providerdata.EmailMessage{
			"gmail-1": {
				ID:         "gmail-1",
				ThreadID:   "thread-gmail-1",
				Subject:    "Review release notes",
				Sender:     "Ada <ada@example.com>",
				Recipients: []string{"team@example.com"},
				Date:       time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC),
				Labels:     []string{"INBOX"},
			},
		},
	}
	imapProvider := &fakeEmailSyncProvider{
		listFunc: func(opts email.SearchOptions) ([]string, error) {
			switch {
			case opts.IsRead != nil && !*opts.IsRead:
				return []string{"INBOX:7"}, nil
			case opts.IsFlagged != nil && *opts.IsFlagged:
				return nil, nil
			case !opts.Since.IsZero():
				return []string{"INBOX:7"}, nil
			default:
				return nil, nil
			}
		},
		messages: map[string]*providerdata.EmailMessage{
			"INBOX:7": {
				ID:         "INBOX:7",
				ThreadID:   "thread-imap-7",
				Subject:    "Schedule site visit",
				Sender:     "Bob <bob@example.com>",
				Recipients: []string{"ops@example.com"},
				Date:       time.Date(2026, time.March, 9, 11, 0, 0, 0, time.UTC),
				Labels:     []string{"INBOX"},
			},
		},
	}
	app.newEmailSyncProvider = func(_ context.Context, account store.ExternalAccount) (emailSyncProvider, error) {
		switch account.ID {
		case gmailAccount.ID:
			return gmailProvider, nil
		case imapAccount.ID:
			return imapProvider, nil
		default:
			t.Fatalf("unexpected account id: %d", account.ID)
			return nil, nil
		}
	}
	app.sourceSync = app.newSourceSyncRunner()

	result, err := app.syncSourcesNow(context.Background())
	if err != nil {
		t.Fatalf("syncSourcesNow() error: %v", err)
	}
	if len(result.Accounts) != 2 {
		t.Fatalf("len(result.Accounts) = %d, want 2", len(result.Accounts))
	}
	for _, account := range result.Accounts {
		if account.Skipped {
			t.Fatalf("account %#v was skipped, want sync", account)
		}
		if account.Err != nil {
			t.Fatalf("account %#v returned error", account)
		}
	}

	gmailItem, err := app.store.GetItemBySource(store.ExternalProviderGmail, "message:gmail-1")
	if err != nil {
		t.Fatalf("GetItemBySource(gmail) error: %v", err)
	}
	if gmailItem.State != store.ItemStateInbox {
		t.Fatalf("gmail item state = %q, want inbox", gmailItem.State)
	}
	if gmailItem.Sphere != store.SphereWork {
		t.Fatalf("gmail item sphere = %q, want work", gmailItem.Sphere)
	}

	imapItem, err := app.store.GetItemBySource(store.ExternalProviderIMAP, "message:INBOX:7")
	if err != nil {
		t.Fatalf("GetItemBySource(imap) error: %v", err)
	}
	if imapItem.Sphere != store.SpherePrivate {
		t.Fatalf("imap item sphere = %q, want private", imapItem.Sphere)
	}

	artifacts, err := app.store.ListArtifactsByKind(store.ArtifactKindEmail)
	if err != nil {
		t.Fatalf("ListArtifactsByKind(email) error: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("len(email artifacts) = %d, want 2", len(artifacts))
	}

	itemArtifacts, err := app.store.ListItemArtifacts(gmailItem.ID)
	if err != nil {
		t.Fatalf("ListItemArtifacts(gmail) error: %v", err)
	}
	if len(itemArtifacts) != 1 {
		t.Fatalf("len(gmail item artifacts) = %d, want 1", len(itemArtifacts))
	}

	var gmailMeta map[string]any
	if err := json.Unmarshal([]byte(strFromPointer(itemArtifacts[0].Artifact.MetaJSON)), &gmailMeta); err != nil {
		t.Fatalf("Unmarshal(gmail meta) error: %v", err)
	}
	if got := strFromAny(gmailMeta["thread_id"]); got != "thread-gmail-1" {
		t.Fatalf("gmail thread_id = %q, want thread-gmail-1", got)
	}
	if got := strFromAny(gmailMeta["sender"]); got != "Ada <ada@example.com>" {
		t.Fatalf("gmail sender = %q, want Ada <ada@example.com>", got)
	}

	gmailBinding, err := app.store.GetBindingByRemote(gmailAccount.ID, store.ExternalProviderGmail, "email", "gmail-1")
	if err != nil {
		t.Fatalf("GetBindingByRemote(gmail) error: %v", err)
	}
	if gmailBinding.ItemID == nil || gmailBinding.ArtifactID == nil {
		t.Fatalf("gmail binding = %#v, want item and artifact ids", gmailBinding)
	}
}

func TestSyncEmailAccountCreatesFollowUpItemsFromRules(t *testing.T) {
	app := newAuthedTestApp(t)

	account, err := app.store.CreateExternalAccount(store.SphereWork, store.ExternalProviderGmail, "Legal Gmail", map[string]any{
		"follow_up_rules": []any{
			map[string]any{"subject": "contract"},
		},
	})
	if err != nil {
		t.Fatalf("CreateExternalAccount() error: %v", err)
	}

	provider := &fakeEmailSyncProvider{
		listFunc: func(opts email.SearchOptions) ([]string, error) {
			switch {
			case opts.Subject == "contract":
				return []string{"gmail-contract"}, nil
			case opts.IsRead != nil && !*opts.IsRead:
				return nil, nil
			case opts.IsFlagged != nil && *opts.IsFlagged:
				return nil, nil
			case !opts.Since.IsZero():
				return []string{"gmail-contract"}, nil
			default:
				return nil, nil
			}
		},
		messages: map[string]*providerdata.EmailMessage{
			"gmail-contract": {
				ID:         "gmail-contract",
				ThreadID:   "thread-contract",
				Subject:    "contract review needed",
				Sender:     "Counsel <counsel@example.com>",
				Recipients: []string{"legal@example.com"},
				Date:       time.Date(2026, time.March, 8, 16, 0, 0, 0, time.UTC),
				Labels:     []string{"Archive"},
				IsRead:     true,
			},
		},
	}
	app.newEmailSyncProvider = func(context.Context, store.ExternalAccount) (emailSyncProvider, error) {
		return provider, nil
	}

	if _, err := app.syncEmailAccount(context.Background(), account); err != nil {
		t.Fatalf("syncEmailAccount() error: %v", err)
	}

	item, err := app.store.GetItemBySource(store.ExternalProviderGmail, "message:gmail-contract")
	if err != nil {
		t.Fatalf("GetItemBySource(rule) error: %v", err)
	}
	if item.Title != "contract review needed" {
		t.Fatalf("rule item title = %q, want subject", item.Title)
	}
}

func TestSyncEmailAccountLeavesDoneItemsClosed(t *testing.T) {
	app := newAuthedTestApp(t)

	account, err := app.store.CreateExternalAccount(store.SphereWork, store.ExternalProviderGmail, "Done Gmail", nil)
	if err != nil {
		t.Fatalf("CreateExternalAccount() error: %v", err)
	}

	provider := &fakeEmailSyncProvider{
		listFunc: func(opts email.SearchOptions) ([]string, error) {
			switch {
			case opts.IsRead != nil && !*opts.IsRead:
				return []string{"gmail-done"}, nil
			case opts.IsFlagged != nil && *opts.IsFlagged:
				return nil, nil
			case !opts.Since.IsZero():
				return []string{"gmail-done"}, nil
			default:
				return nil, nil
			}
		},
		messages: map[string]*providerdata.EmailMessage{
			"gmail-done": {
				ID:         "gmail-done",
				ThreadID:   "thread-done",
				Subject:    "Already handled",
				Sender:     "Ops <ops@example.com>",
				Recipients: []string{"team@example.com"},
				Date:       time.Date(2026, time.March, 9, 9, 0, 0, 0, time.UTC),
				Labels:     []string{"INBOX"},
			},
		},
	}
	app.newEmailSyncProvider = func(context.Context, store.ExternalAccount) (emailSyncProvider, error) {
		return provider, nil
	}

	if _, err := app.syncEmailAccount(context.Background(), account); err != nil {
		t.Fatalf("first syncEmailAccount() error: %v", err)
	}
	item, err := app.store.GetItemBySource(store.ExternalProviderGmail, "message:gmail-done")
	if err != nil {
		t.Fatalf("GetItemBySource(first) error: %v", err)
	}
	if err := app.store.UpdateItemState(item.ID, store.ItemStateDone); err != nil {
		t.Fatalf("UpdateItemState(done) error: %v", err)
	}

	if _, err := app.syncEmailAccount(context.Background(), account); err != nil {
		t.Fatalf("second syncEmailAccount() error: %v", err)
	}
	item, err = app.store.GetItem(item.ID)
	if err != nil {
		t.Fatalf("GetItem(second) error: %v", err)
	}
	if item.State != store.ItemStateDone {
		t.Fatalf("item state after resync = %q, want done", item.State)
	}
}

func strFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
