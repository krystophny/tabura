package hotwordtrain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	officialCatalogSourceID     = "openwakeword"
	officialCatalogSourceLabel  = "Official openWakeWord"
	officialCatalogSourceURL    = "https://github.com/dscripka/openWakeWord"
	communityCatalogSourceID    = "home-assistant-community"
	communityCatalogSourceLabel = "Home Assistant Community"
	communityCatalogSourceURL   = "https://github.com/fwartner/home-assistant-wakewords-collection"
)

var catalogVersionRE = regexp.MustCompile(`^v[0-9]+$`)

type catalogSources struct {
	OfficialModelsURL string
	CommunityTreeURL  string
	CommunityRawBase  string
}

type catalogCache struct {
	entries   []CatalogEntry
	fetchedAt time.Time
}

type modelMetadata struct {
	CatalogKey      string `json:"catalog_key,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	Phrase          string `json:"phrase,omitempty"`
	Source          string `json:"source,omitempty"`
	SourceURL       string `json:"source_url,omitempty"`
	ReadmeURL       string `json:"readme_url,omitempty"`
	DownloadURL     string `json:"download_url,omitempty"`
	UpstreamFile    string `json:"upstream_file,omitempty"`
	HasExternalData bool   `json:"has_external_data,omitempty"`
}

func defaultCatalogSources() catalogSources {
	return catalogSources{
		OfficialModelsURL: "https://raw.githubusercontent.com/dscripka/openWakeWord/main/openwakeword/__init__.py",
		CommunityTreeURL:  "https://api.github.com/repos/fwartner/home-assistant-wakewords-collection/git/trees/main?recursive=1",
		CommunityRawBase:  "https://raw.githubusercontent.com/fwartner/home-assistant-wakewords-collection/main/",
	}
}

func (m *Manager) ListCatalog(ctx context.Context) ([]CatalogEntry, error) {
	m.mu.Lock()
	cache := m.catalogCache
	client := m.httpClient
	sources := m.catalog
	m.mu.Unlock()
	if len(cache.entries) > 0 && time.Since(cache.fetchedAt) < 10*time.Minute {
		return annotateCatalogEntries(cache.entries, m)
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	official, err := fetchOfficialCatalog(ctx, client, sources.OfficialModelsURL)
	if err != nil {
		if len(cache.entries) > 0 {
			return annotateCatalogEntries(cache.entries, m)
		}
		return nil, err
	}
	community, err := fetchCommunityCatalog(ctx, client, sources.CommunityTreeURL, sources.CommunityRawBase)
	if err != nil {
		if len(cache.entries) > 0 {
			return annotateCatalogEntries(cache.entries, m)
		}
		return nil, err
	}
	entries := append(official, community...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SourceLabel != entries[j].SourceLabel {
			return entries[i].SourceLabel < entries[j].SourceLabel
		}
		if entries[i].Phrase != entries[j].Phrase {
			return entries[i].Phrase < entries[j].Phrase
		}
		return entries[i].DisplayName < entries[j].DisplayName
	})

	m.mu.Lock()
	m.catalogCache = catalogCache{
		entries:   append([]CatalogEntry(nil), entries...),
		fetchedAt: time.Now(),
	}
	m.mu.Unlock()

	return annotateCatalogEntries(entries, m)
}

func (m *Manager) DownloadCatalogModel(ctx context.Context, key string) (Model, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Model{}, fmt.Errorf("missing catalog key")
	}
	entries, err := m.ListCatalog(ctx)
	if err != nil {
		return Model{}, err
	}
	var entry *CatalogEntry
	for i := range entries {
		if entries[i].Key == key {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		return Model{}, fmt.Errorf("catalog entry not found")
	}
	if entry.Installed && entry.InstalledModel != nil {
		return *entry.InstalledModel, nil
	}

	m.mu.Lock()
	client := m.httpClient
	m.mu.Unlock()
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.DownloadURL, nil)
	if err != nil {
		return Model{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Model{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Model{}, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Model{}, err
	}
	if len(data) == 0 {
		return Model{}, fmt.Errorf("download failed: empty body")
	}
	if err := m.ensureDir(m.modelsDir()); err != nil {
		return Model{}, err
	}
	stamp := time.Now().UTC().Format("2006-01-02_15-04-05Z")
	base := catalogFileSlug(entry.Source, entry.UpstreamFile)
	targetName := fmt.Sprintf("%s-%s.onnx", base, stamp)
	targetPath := filepath.Join(m.modelsDir(), targetName)
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return Model{}, err
	}
	meta := modelMetadata{
		CatalogKey:      entry.Key,
		DisplayName:     entry.DisplayName,
		Phrase:          entry.Phrase,
		Source:          entry.SourceLabel,
		SourceURL:       entry.SourceURL,
		ReadmeURL:       entry.ReadmeURL,
		DownloadURL:     entry.DownloadURL,
		UpstreamFile:    entry.UpstreamFile,
		HasExternalData: false,
	}
	if err := writeModelMetadata(targetPath, meta); err != nil {
		return Model{}, err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return Model{}, err
	}
	return modelFromPath(targetPath, info, false, meta), nil
}

func (m *Manager) ActiveModel() (Model, error) {
	vendorPath := m.vendorModelPath()
	info, err := os.Stat(vendorPath)
	if err != nil {
		return Model{}, err
	}
	meta, _ := readModelMetadata(m.activeModelMetadataPath())
	return modelFromPath(vendorPath, info, true, meta), nil
}

func annotateCatalogEntries(entries []CatalogEntry, m *Manager) ([]CatalogEntry, error) {
	models, err := m.ListModels()
	if err != nil {
		return nil, err
	}
	installed := make(map[string]Model)
	for _, model := range models {
		if model.CatalogKey == "" || model.Production {
			continue
		}
		existing, ok := installed[model.CatalogKey]
		if !ok || model.CreatedAt > existing.CreatedAt {
			installed[model.CatalogKey] = model
		}
	}
	active, _ := m.ActiveModel()
	out := make([]CatalogEntry, 0, len(entries))
	for _, entry := range entries {
		next := entry
		if model, ok := installed[entry.Key]; ok {
			modelCopy := model
			next.Installed = true
			next.InstalledModel = &modelCopy
		}
		next.Active = active.CatalogKey != "" && active.CatalogKey == entry.Key
		out = append(out, next)
	}
	return out, nil
}

func fetchOfficialCatalog(ctx context.Context, client *http.Client, sourceURL string) ([]CatalogEntry, error) {
	body, err := fetchRemoteText(ctx, client, sourceURL)
	if err != nil {
		return nil, err
	}
	blockStart := strings.Index(body, "MODELS = {")
	blockEnd := strings.Index(body, "model_class_mappings = {")
	if blockStart < 0 || blockEnd <= blockStart {
		return nil, fmt.Errorf("official catalog parse failed")
	}
	block := body[blockStart:blockEnd]
	re := regexp.MustCompile(`"([a-zA-Z0-9_]+)"\s*:\s*\{[^}]*"download_url"\s*:\s*"([^"]+)"`)
	matches := re.FindAllStringSubmatch(block, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("official catalog parse failed")
	}
	entries := make([]CatalogEntry, 0, len(matches))
	for _, match := range matches {
		key := strings.TrimSpace(match[1])
		downloadURL := strings.TrimSpace(match[2])
		if key == "" || downloadURL == "" {
			continue
		}
		onnxURL := strings.TrimSuffix(downloadURL, ".tflite") + ".onnx"
		display := displayNameForPhrase(key)
		entries = append(entries, CatalogEntry{
			Key:          officialCatalogSourceID + ":" + key,
			DisplayName:  display,
			Phrase:       strings.ReplaceAll(key, "_", " "),
			Source:       officialCatalogSourceID,
			SourceLabel:  officialCatalogSourceLabel,
			SourceURL:    officialCatalogSourceURL,
			ReadmeURL:    officialCatalogSourceURL,
			DownloadURL:  onnxURL,
			UpstreamFile: path.Base(onnxURL),
		})
	}
	return entries, nil
}

func fetchCommunityCatalog(ctx context.Context, client *http.Client, treeURL, rawBase string) ([]CatalogEntry, error) {
	body, err := fetchRemoteText(ctx, client, treeURL)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}
	entries := make([]CatalogEntry, 0)
	seen := make(map[string]struct{})
	re := regexp.MustCompile(`^en/([^/]+)/([^/]+)\.onnx$`)
	for _, item := range payload.Tree {
		if item.Type != "blob" {
			continue
		}
		match := re.FindStringSubmatch(item.Path)
		if len(match) != 3 {
			continue
		}
		dirName := match[1]
		stem := match[2]
		key := communityCatalogSourceID + ":" + item.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		readmeURL := rawBase + "en/" + dirName + "/README.md"
		entries = append(entries, CatalogEntry{
			Key:          key,
			DisplayName:  displayNameForPhrase(stem),
			Phrase:       strings.ReplaceAll(dirName, "_", " "),
			Source:       communityCatalogSourceID,
			SourceLabel:  communityCatalogSourceLabel,
			SourceURL:    communityCatalogSourceURL,
			ReadmeURL:    readmeURL,
			DownloadURL:  rawBase + item.Path,
			UpstreamFile: path.Base(item.Path),
		})
	}
	return entries, nil
}

func fetchRemoteText(ctx context.Context, client *http.Client, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json, text/plain;q=0.9, */*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("catalog fetch failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func displayNameForPhrase(value string) string {
	parts := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "_", " "), "-", " "))
	for i := range parts {
		word := parts[i]
		switch {
		case word == "":
		case len(word) <= 3 && strings.ToUpper(word) == word:
			parts[i] = word
		case catalogVersionRE.MatchString(strings.ToLower(word)):
			parts[i] = strings.ToUpper(word[:1]) + word[1:]
		default:
			parts[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(parts, " ")
}

func catalogFileSlug(sourceID, upstreamFile string) string {
	base := strings.TrimSuffix(strings.TrimSpace(upstreamFile), path.Ext(upstreamFile))
	base = strings.ReplaceAll(base, ".", "-")
	base = strings.ReplaceAll(base, "_", "-")
	sourceID = strings.ReplaceAll(strings.TrimSpace(sourceID), "_", "-")
	return strings.ToLower(sourceID + "-" + base)
}
