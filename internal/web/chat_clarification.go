package web

import (
	"strings"

	"github.com/krystophny/tabura/internal/store"
)

func chatMessageText(msg store.ChatMessage) string {
	text := strings.TrimSpace(msg.ContentPlain)
	if text == "" {
		text = strings.TrimSpace(msg.ContentMarkdown)
	}
	return text
}

func recentConversationTail(messages []store.ChatMessage, limit int) []store.ChatMessage {
	if limit <= 0 {
		return nil
	}
	tail := make([]store.ChatMessage, 0, limit)
	for i := len(messages) - 1; i >= 0 && len(tail) < limit; i-- {
		role := strings.ToLower(strings.TrimSpace(messages[i].Role))
		if role != "user" && role != "assistant" {
			continue
		}
		if chatMessageText(messages[i]) == "" {
			continue
		}
		tail = append(tail, messages[i])
	}
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return tail
}

func looksLikeClarificationQuestion(text string) bool {
	return strings.Contains(strings.TrimSpace(text), "?")
}

func looksLikeStandaloneSystemRequest(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, prefix := range []string{
		"open ", "show ", "display ", "list ", "create ", "make ", "switch ", "toggle ",
		"sync ", "print ", "review ", "move ", "rename ", "delete ", "clear ", "capture ",
		"refine ", "promote ", "assign ", "link ", "map ", "triage ", "delegate ",
		"snooze ", "split ", "explain ", "summarize ", "write ", "draft ", "turn ",
		"stop ", "start ", "be ",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func contextualizeClarificationReply(messages []store.ChatMessage, latestUser string) string {
	trimmedLatest := strings.TrimSpace(latestUser)
	if trimmedLatest == "" {
		return ""
	}
	if looksLikeStandaloneSystemRequest(trimmedLatest) {
		return trimmedLatest
	}
	tail := recentConversationTail(messages, 4)
	if len(tail) == 0 ||
		!strings.EqualFold(strings.TrimSpace(tail[len(tail)-1].Role), "user") ||
		!strings.EqualFold(chatMessageText(tail[len(tail)-1]), trimmedLatest) {
		tail = append(tail, store.ChatMessage{Role: "user", ContentPlain: trimmedLatest})
	}
	if len(tail) < 3 {
		return trimmedLatest
	}
	priorUser := tail[len(tail)-3]
	assistant := tail[len(tail)-2]
	currentUser := tail[len(tail)-1]
	if !strings.EqualFold(strings.TrimSpace(priorUser.Role), "user") ||
		!strings.EqualFold(strings.TrimSpace(assistant.Role), "assistant") ||
		!strings.EqualFold(strings.TrimSpace(currentUser.Role), "user") {
		return trimmedLatest
	}
	priorUserText := chatMessageText(priorUser)
	assistantText := chatMessageText(assistant)
	if priorUserText == "" || assistantText == "" || strings.EqualFold(priorUserText, trimmedLatest) {
		return trimmedLatest
	}
	if !looksLikeClarificationQuestion(assistantText) {
		return trimmedLatest
	}
	return strings.TrimSpace(priorUserText + "\nAssistant asked: " + assistantText + "\nUser answered: " + trimmedLatest)
}

func (a *App) contextualizeClarificationReplyForSession(sessionID, latestUser string) string {
	trimmedLatest := strings.TrimSpace(latestUser)
	if a == nil || a.store == nil || strings.TrimSpace(sessionID) == "" {
		return trimmedLatest
	}
	messages, err := a.store.ListChatMessages(sessionID, 8)
	if err != nil {
		return trimmedLatest
	}
	return contextualizeClarificationReply(messages, trimmedLatest)
}
