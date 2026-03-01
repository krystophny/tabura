package web

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krystophny/tabura/internal/appserver"
)

func (a *App) registerActiveChatTurn(sessionID, runID string, cancel context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chatTurnCancel[sessionID] == nil {
		a.chatTurnCancel[sessionID] = map[string]context.CancelFunc{}
	}
	a.chatTurnCancel[sessionID][runID] = cancel
}

func (a *App) unregisterActiveChatTurn(sessionID, runID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	runs := a.chatTurnCancel[sessionID]
	if runs == nil {
		return
	}
	delete(runs, runID)
	if len(runs) == 0 {
		delete(a.chatTurnCancel, sessionID)
	}
}

func (a *App) cancelActiveChatTurns(sessionID string) int {
	a.mu.Lock()
	runs := a.chatTurnCancel[sessionID]
	if len(runs) == 0 {
		a.mu.Unlock()
		return 0
	}
	cancels := make([]context.CancelFunc, 0, len(runs))
	for _, cancel := range runs {
		cancels = append(cancels, cancel)
	}
	delete(a.chatTurnCancel, sessionID)
	a.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	return len(cancels)
}

func (a *App) clearQueuedChatTurns(sessionID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	queued := a.chatTurnQueue[sessionID]
	delete(a.chatTurnQueue, sessionID)
	delete(a.chatTurnOutputMode, sessionID)
	return queued
}

func (a *App) cancelChatWork(sessionID string) (int, int) {
	activeCanceled := a.cancelActiveChatTurns(sessionID)
	queuedCanceled := a.clearQueuedChatTurns(sessionID)
	if queuedCanceled > 0 {
		a.broadcastChatEvent(sessionID, map[string]interface{}{
			"type":  "turn_queue_cleared",
			"count": queuedCanceled,
		})
	}
	return activeCanceled, queuedCanceled
}

type clearAllReport struct {
	ActiveCanceled   int
	QueuedCanceled   int
	DelegateCanceled int
	SessionsClosed   int
	TempFilesCleared int
}

func (a *App) clearCanvasForProject(projectKey string) {
	canvasSessionID := strings.TrimSpace(a.resolveCanvasSessionID(projectKey))
	if canvasSessionID == "" {
		return
	}
	a.mu.Lock()
	port, ok := a.tunnelPorts[canvasSessionID]
	a.mu.Unlock()
	if !ok {
		return
	}
	_, _ = a.mcpToolsCall(port, "canvas_clear", map[string]interface{}{
		"session_id": canvasSessionID,
		"reason":     "context reset",
	})
}

func (a *App) clearProjectTempCanvasFiles(projectKey string) int {
	cwd := strings.TrimSpace(a.cwdForProjectKey(projectKey))
	if cwd == "" {
		return 0
	}
	tmpDir := filepath.Join(cwd, ".tabura", "artifacts", "tmp")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return 0
	}
	cleared := 0
	for _, entry := range entries {
		target := filepath.Join(tmpDir, entry.Name())
		if err := os.RemoveAll(target); err == nil {
			cleared++
		}
	}
	return cleared
}

func (a *App) clearAllAgentsAndContexts(currentSessionID string) (clearAllReport, error) {
	report := clearAllReport{}
	sessions, err := a.store.ListChatSessions()
	if err != nil {
		return report, err
	}
	for _, session := range sessions {
		activeCanceled, queuedCanceled := a.cancelChatWork(session.ID)
		report.ActiveCanceled += activeCanceled
		report.QueuedCanceled += queuedCanceled
		report.DelegateCanceled += a.cancelDelegatedJobsForProject(session.ProjectKey)
		report.TempFilesCleared += a.clearProjectTempCanvasFiles(session.ProjectKey)
		a.clearCanvasForProject(session.ProjectKey)
		a.broadcastChatEvent(session.ID, map[string]interface{}{
			"type": "chat_cleared",
		})
	}
	closed := 0
	a.mu.Lock()
	appSessions := a.chatAppSessions
	a.chatAppSessions = map[string]*appserver.Session{}
	a.mu.Unlock()
	for _, s := range appSessions {
		if s == nil {
			continue
		}
		_ = s.Close()
		closed++
	}
	report.SessionsClosed = closed
	if err := a.store.ClearAllChatMessages(); err != nil {
		return report, err
	}
	if err := a.store.ClearAllChatEvents(); err != nil {
		return report, err
	}
	if err := a.store.ResetAllChatSessionThreads(); err != nil {
		return report, err
	}
	if strings.TrimSpace(currentSessionID) != "" {
		a.closeAppSession(currentSessionID)
	}
	return report, nil
}

func (a *App) delegateActiveJobsForProject(projectKey string) int {
	cwd := strings.TrimSpace(a.cwdForProjectKey(projectKey))
	if cwd == "" {
		return 0
	}
	canvasSessionID := strings.TrimSpace(a.resolveCanvasSessionID(projectKey))
	if canvasSessionID == "" {
		return 0
	}
	a.mu.Lock()
	port, ok := a.tunnelPorts[canvasSessionID]
	a.mu.Unlock()
	if !ok {
		return 0
	}
	status, err := a.mcpToolsCall(port, "delegate_to_model_active_count", map[string]interface{}{"cwd_prefix": cwd})
	if err != nil {
		log.Printf("delegate activity probe failed for project=%q cwd=%q: %v", projectKey, cwd, err)
		return 0
	}
	return intFromAny(status["active"], 0)
}

func (a *App) cancelDelegatedJobsForProject(projectKey string) int {
	cwd := strings.TrimSpace(a.cwdForProjectKey(projectKey))
	if cwd == "" {
		return 0
	}
	canvasSessionID := strings.TrimSpace(a.resolveCanvasSessionID(projectKey))
	if canvasSessionID == "" {
		return 0
	}
	a.mu.Lock()
	port, ok := a.tunnelPorts[canvasSessionID]
	a.mu.Unlock()
	if !ok {
		return 0
	}
	status, err := a.mcpToolsCall(port, "delegate_to_model_cancel_all", map[string]interface{}{"cwd_prefix": cwd})
	if err != nil {
		log.Printf("delegate cancel-all failed for project=%q cwd=%q: %v", projectKey, cwd, err)
		return 0
	}
	return intFromAny(status["canceled"], 0)
}

func (a *App) activeChatTurnCount(sessionID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.chatTurnCancel[sessionID])
}

func (a *App) queuedChatTurnCount(sessionID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.chatTurnQueue[sessionID]
}

func (a *App) enqueueAssistantTurn(sessionID, outputMode string, opts ...bool) int {
	mode := normalizeTurnOutputMode(outputMode)
	localOnly := len(opts) > 0 && opts[0]
	a.mu.Lock()
	a.chatTurnOutputMode[sessionID] = append(a.chatTurnOutputMode[sessionID], mode)
	a.chatTurnLocalOnly[sessionID] = append(a.chatTurnLocalOnly[sessionID], localOnly)
	a.chatTurnQueue[sessionID] = a.chatTurnQueue[sessionID] + 1
	queued := a.chatTurnQueue[sessionID]
	workerRunning := a.chatTurnWorker[sessionID]
	if !workerRunning {
		a.chatTurnWorker[sessionID] = true
	}
	a.mu.Unlock()
	if !workerRunning {
		go a.runAssistantTurnQueue(sessionID)
	}
	return queued
}

type dequeuedTurn struct {
	outputMode string
	localOnly  bool
}

func (a *App) dequeueAssistantTurn(sessionID string) (dequeuedTurn, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	queued := a.chatTurnQueue[sessionID]
	if queued <= 0 {
		return dequeuedTurn{}, false
	}
	modes := a.chatTurnOutputMode[sessionID]
	mode := turnOutputModeVoice
	if len(modes) > 0 {
		mode = normalizeTurnOutputMode(modes[0])
		modes = modes[1:]
		if len(modes) == 0 {
			delete(a.chatTurnOutputMode, sessionID)
		} else {
			a.chatTurnOutputMode[sessionID] = modes
		}
	}
	localFlags := a.chatTurnLocalOnly[sessionID]
	localOnly := false
	if len(localFlags) > 0 {
		localOnly = localFlags[0]
		localFlags = localFlags[1:]
		if len(localFlags) == 0 {
			delete(a.chatTurnLocalOnly, sessionID)
		} else {
			a.chatTurnLocalOnly[sessionID] = localFlags
		}
	}
	queued--
	if queued <= 0 {
		delete(a.chatTurnQueue, sessionID)
		delete(a.chatTurnOutputMode, sessionID)
		delete(a.chatTurnLocalOnly, sessionID)
		return dequeuedTurn{outputMode: mode, localOnly: localOnly}, true
	}
	a.chatTurnQueue[sessionID] = queued
	return dequeuedTurn{outputMode: mode, localOnly: localOnly}, true
}

func (a *App) markAssistantWorkerIdleIfQueueEmpty(sessionID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chatTurnQueue[sessionID] > 0 {
		return false
	}
	delete(a.chatTurnWorker, sessionID)
	return true
}

func (a *App) runAssistantTurnQueue(sessionID string) {
	for {
		turn, ok := a.dequeueAssistantTurn(sessionID)
		if !ok {
			if a.markAssistantWorkerIdleIfQueueEmpty(sessionID) {
				return
			}
			continue
		}
		a.runAssistantTurn(sessionID, turn.outputMode, turn.localOnly)
	}
}

func (a *App) getOrCreateAppSession(sessionID string, cwd string, profile appServerModelProfile) (*appserver.Session, bool, error) {
	a.mu.Lock()
	s := a.chatAppSessions[sessionID]
	a.mu.Unlock()
	if s != nil && s.IsOpen() {
		return s, true, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var existingThreadID string
	if sess, err := a.store.GetChatSession(sessionID); err == nil {
		existingThreadID = strings.TrimSpace(sess.AppThreadID)
	}
	var newSess *appserver.Session
	var resumed bool
	if existingThreadID != "" {
		rs, ok, err := a.appServerClient.ResumeSessionWithParams(ctx, cwd, profile.Model, profile.ThreadParams, existingThreadID)
		if err != nil {
			return nil, false, err
		}
		newSess = rs
		resumed = ok
	} else {
		rs, err := a.appServerClient.OpenSessionWithParams(ctx, cwd, profile.Model, profile.ThreadParams)
		if err != nil {
			return nil, false, err
		}
		newSess = rs
	}
	a.mu.Lock()
	if old := a.chatAppSessions[sessionID]; old != nil {
		_ = old.Close()
	}
	a.chatAppSessions[sessionID] = newSess
	a.mu.Unlock()
	return newSess, resumed, nil
}

func (a *App) closeAppSession(sessionID string) {
	a.mu.Lock()
	s := a.chatAppSessions[sessionID]
	delete(a.chatAppSessions, sessionID)
	a.mu.Unlock()
	if s != nil {
		_ = s.Close()
	}
}
