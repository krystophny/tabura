package web

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/krystophny/sloppad/internal/store"
)

type helpyEmailProvidersFile struct {
	DefaultProvider string                           `json:"default_provider"`
	Providers       map[string]helpyEmailProviderRow `json:"providers"`
}

type helpyEmailProviderRow struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	TLS      bool   `json:"tls"`
	StartTLS bool   `json:"starttls"`
}

func (a *App) migrateLegacyExternalAccounts() error {
	if a == nil || a.store == nil {
		return nil
	}
	if !shouldRunLegacyExternalAccountMigration(a.dataDir) {
		return nil
	}
	if err := a.ensureGmailAccountsPrivate(); err != nil {
		return err
	}
	return a.importLegacyHelpyExchangeEWSAccount()
}

func shouldRunLegacyExternalAccountMigration(dataDir string) bool {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return false
	}
	cleanHome := filepath.Clean(home)
	cleanDataDir := filepath.Clean(strings.TrimSpace(dataDir))
	if cleanDataDir == "" {
		return false
	}
	return cleanDataDir == cleanHome || strings.HasPrefix(cleanDataDir, cleanHome+string(os.PathSeparator))
}

func (a *App) ensureGmailAccountsPrivate() error {
	accounts, err := a.store.ListExternalAccountsByProvider(store.ExternalProviderGmail)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.Sphere == store.SpherePrivate {
			continue
		}
		sphere := store.SpherePrivate
		if err := a.store.UpdateExternalAccount(account.ID, store.ExternalAccountUpdate{Sphere: &sphere}); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) importLegacyHelpyExchangeEWSAccount() error {
	existing, err := a.store.ListExternalAccountsByProvider(store.ExternalProviderExchangeEWS)
	if err == nil && len(existing) > 0 {
		return nil
	}
	if err != nil {
		return err
	}
	row, providerName, err := loadLegacyHelpyExchangeProvider()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(row.Type), store.ExternalProviderIMAP) {
		return nil
	}
	host := strings.TrimSpace(row.Host)
	username := strings.TrimSpace(row.Username)
	if host == "" || username == "" {
		return nil
	}
	config := map[string]any{
		"endpoint":             "https://" + host + "/EWS/Exchange.asmx",
		"username":             username,
		"archive_folder":       "Archive",
		"legacy_helpy_env_var": "HELPY_IMAP_PASSWORD_" + strings.ToUpper(strings.TrimSpace(providerName)),
		"batch_size":           50,
	}
	_, err = a.store.CreateExternalAccount(store.SphereWork, store.ExternalProviderExchangeEWS, "TU Graz Exchange", config)
	return err
}

func loadLegacyHelpyExchangeProvider() (helpyEmailProviderRow, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return helpyEmailProviderRow{}, "", err
	}
	path := filepath.Join(home, ".config", "helpy", "email_providers.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return helpyEmailProviderRow{}, "", err
	}
	var file helpyEmailProvidersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return helpyEmailProviderRow{}, "", err
	}
	if providerName := strings.TrimSpace(file.DefaultProvider); providerName != "" {
		if row, ok := file.Providers[providerName]; ok {
			return row, providerName, nil
		}
	}
	for name, row := range file.Providers {
		if strings.Contains(strings.ToLower(strings.TrimSpace(row.Host)), "exchange") {
			return row, name, nil
		}
	}
	return helpyEmailProviderRow{}, "", os.ErrNotExist
}
