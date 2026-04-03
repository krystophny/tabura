package web

import (
	"fmt"
	"path/filepath"
	"strings"
)

func isResearchQuery(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	strongSignals := []string{
		"research ",
		" research",
		"find and summarize",
		"find recent work",
		"recent work",
		"literature review",
		"survey the literature",
		"compare ",
		"papers on",
		"paper about",
		"citations",
	}
	for _, signal := range strongSignals {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

func researchArtifactRoot(sessionID string) string {
	stem := strings.TrimSpace(sessionID)
	if stem == "" {
		return ""
	}
	stem = canvasTempFileStemRe.ReplaceAllString(stem, "-")
	stem = strings.Trim(stem, "-.")
	if stem == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(".sloppad", "artifacts", "research", stem))
}

func appendResearchArtifactPrompt(b *strings.Builder, outputMode, userText, researchRoot string) {
	if b == nil || normalizeTurnOutputMode(outputMode) != turnOutputModeSilent || !isResearchQuery(userText) {
		return
	}
	root := strings.TrimSpace(researchRoot)
	if root == "" {
		root = filepath.ToSlash(filepath.Join(".sloppad", "artifacts", "research", "current"))
	}
	b.WriteString("## Research Artifact Output\n")
	b.WriteString("This is a research request. Produce multiple file-backed canvas artifacts instead of one long chat reply.\n")
	fmt.Fprintf(b, "- Write every :::file block under %q.\n", root)
	fmt.Fprintf(b, "- Include %q with the structured summary and citations.\n", filepath.ToSlash(filepath.Join(root, "summary.md")))
	b.WriteString("- Add supporting artifacts such as sources.md, comparison.md, notes.md, or extracted snippets when useful.\n")
	b.WriteString("- Keep any companion chat outside :::file blocks to one short sentence.\n\n")
}

func normalizeResearchFileBlocks(blocks []fileBlock, researchRoot string) []fileBlock {
	root := filepath.ToSlash(strings.TrimSpace(researchRoot))
	if root == "" || len(blocks) == 0 {
		return blocks
	}
	out := make([]fileBlock, 0, len(blocks))
	for i, block := range blocks {
		path := filepath.ToSlash(strings.TrimSpace(block.Path))
		if filepath.IsAbs(path) {
			path = filepath.Base(path)
		}
		path = strings.TrimPrefix(path, "./")
		path = filepath.ToSlash(filepath.Clean(path))
		if path == "." || path == "" {
			path = defaultResearchArtifactName(i)
		}
		if strings.HasPrefix(path, "../") {
			path = defaultResearchArtifactName(i)
		}
		if strings.HasPrefix(path, ".sloppad/") && !strings.HasPrefix(path, root+"/") {
			path = filepath.Base(path)
		}
		if !strings.HasPrefix(path, root+"/") && path != root {
			path = filepath.ToSlash(filepath.Join(root, path))
		}
		block.Path = path
		out = append(out, block)
	}
	return out
}

func defaultResearchArtifactName(index int) string {
	switch index {
	case 0:
		return "summary.md"
	case 1:
		return "sources.md"
	case 2:
		return "comparison.md"
	default:
		return fmt.Sprintf("artifact-%d.md", index+1)
	}
}

func (a *App) isResearchTurn(sessionID string) bool {
	if a == nil || strings.TrimSpace(sessionID) == "" {
		return false
	}
	messages, err := a.store.ListChatMessages(sessionID, 12)
	if err != nil {
		return false
	}
	return isResearchQuery(latestUserMessage(messages))
}
