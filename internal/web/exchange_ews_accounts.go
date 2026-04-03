package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/krystophny/sloppad/internal/email"
	"github.com/krystophny/sloppad/internal/ews"
	"github.com/krystophny/sloppad/internal/providerdata"
	"github.com/krystophny/sloppad/internal/store"
)

func decodeExchangeEWSAccountConfig(account store.ExternalAccount) (email.ExchangeEWSConfig, error) {
	config := map[string]any{}
	raw := strings.TrimSpace(account.ConfigJSON)
	if raw != "" && raw != "{}" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			return email.ExchangeEWSConfig{}, fmt.Errorf("decode exchange ews account config: %w", err)
		}
	}
	return email.ExchangeEWSConfigFromMap(account.AccountName, config)
}

func (a *App) exchangeEWSMailProviderForAccount(ctx context.Context, account store.ExternalAccount) (*email.ExchangeEWSMailProvider, error) {
	cfg, err := decodeExchangeEWSAccountConfig(account)
	if err != nil {
		return nil, err
	}
	password, _, err := a.store.ResolveExternalAccountPasswordForAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	cfg.Password = password
	return email.NewExchangeEWSMailProvider(cfg)
}

func (a *App) exchangeEWSClientForAccount(ctx context.Context, account store.ExternalAccount) (*ews.Client, error) {
	cfg, err := decodeExchangeEWSAccountConfig(account)
	if err != nil {
		return nil, err
	}
	password, _, err := a.store.ResolveExternalAccountPasswordForAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	return ews.NewClient(ews.Config{
		Endpoint:      cfg.Endpoint,
		Username:      cfg.Username,
		Password:      password,
		ServerVersion: cfg.ServerVersion,
		BatchSize:     cfg.BatchSize,
		InsecureTLS:   cfg.InsecureTLS,
	})
}

type exchangeEWSContactSyncProvider struct {
	client *ews.Client
}

func (p *exchangeEWSContactSyncProvider) ListContacts(ctx context.Context) ([]providerdata.Contact, error) {
	if p == nil || p.client == nil {
		return nil, nil
	}
	contacts, err := p.client.GetContacts(ctx, "", 0, 500)
	if err != nil {
		return nil, err
	}
	out := make([]providerdata.Contact, 0, len(contacts))
	for _, contact := range contacts {
		out = append(out, providerdata.Contact{
			ProviderRef:  contact.ID,
			Name:         contact.DisplayName,
			Email:        contact.Email,
			Organization: contact.CompanyName,
			Phones:       append([]string(nil), contact.Phones...),
		})
	}
	return out, nil
}

func (p *exchangeEWSContactSyncProvider) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}
