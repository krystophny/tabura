import { refs, state } from './app-context.js';
import { isTurnIntelligenceConnected, sendTurnConfig } from './turn-client.js';

const DIALOGUE_DIAGNOSTIC_EVENT_LIMIT = 120;

function cloneDialogueDiagnostics() {
  const diagnostics: any = state.dialogueDiagnostics || {};
  return {
    ...diagnostics,
    recentEvents: Array.isArray(diagnostics.recentEvents) ? diagnostics.recentEvents.slice() : [],
    lastAction: diagnostics.lastAction ? { ...diagnostics.lastAction } : null,
    lastMetrics: diagnostics.lastMetrics ? JSON.parse(JSON.stringify(diagnostics.lastMetrics)) : null,
  };
}

export function pushDialogueDiagnosticEvent(kind, payload: Record<string, any> = {}) {
  if (!state.dialogueDiagnostics) {
    state.dialogueDiagnostics = {
      connected: false,
      sessionId: '',
      profile: state.turnPolicyProfile || 'balanced',
      evalLoggingEnabled: state.turnEvalLoggingEnabled !== false,
      readyAt: 0,
      lastAction: null,
      lastMetrics: null,
      recentEvents: [],
    };
  }
  const entry = {
    ts: Date.now(),
    kind: String(kind || '').trim() || 'event',
    ...payload,
  };
  const events = Array.isArray(state.dialogueDiagnostics.recentEvents)
    ? state.dialogueDiagnostics.recentEvents
    : [];
  events.push(entry);
  while (events.length > DIALOGUE_DIAGNOSTIC_EVENT_LIMIT) {
    events.shift();
  }
  state.dialogueDiagnostics.recentEvents = events;
}

export function getDialogueDiagnostics() {
  return cloneDialogueDiagnostics();
}

export function clearDialogueDiagnostics() {
  state.dialogueDiagnostics = {
    connected: isTurnIntelligenceConnected(),
    sessionId: state.chatSessionId || '',
    profile: state.turnPolicyProfile || 'balanced',
    evalLoggingEnabled: state.turnEvalLoggingEnabled !== false,
    readyAt: 0,
    lastAction: null,
    lastMetrics: null,
    recentEvents: [],
  };
}

export async function setDialogueTurnProfile(profile) {
  const nextProfile = String(profile || '').trim().toLowerCase() || 'balanced';
  const payload = await refs.updateRuntimePreferences({
    turn_policy_profile: nextProfile,
  });
  state.turnPolicyProfile = String(payload?.turn_policy_profile || nextProfile).trim().toLowerCase() || 'balanced';
  if (state.dialogueDiagnostics) {
    state.dialogueDiagnostics.profile = state.turnPolicyProfile;
  }
  sendTurnConfig(state.turnPolicyProfile, state.turnEvalLoggingEnabled !== false);
  pushDialogueDiagnosticEvent('profile_updated', { profile: state.turnPolicyProfile });
  return state.turnPolicyProfile;
}

export async function setDialogueEvalLogging(enabled) {
  const nextEnabled = Boolean(enabled);
  const payload = await refs.updateRuntimePreferences({
    turn_eval_logging_enabled: nextEnabled,
  });
  state.turnEvalLoggingEnabled = payload?.turn_eval_logging_enabled !== false;
  if (state.dialogueDiagnostics) {
    state.dialogueDiagnostics.evalLoggingEnabled = state.turnEvalLoggingEnabled;
  }
  sendTurnConfig(state.turnPolicyProfile, state.turnEvalLoggingEnabled);
  pushDialogueDiagnosticEvent('eval_logging_updated', { enabled: state.turnEvalLoggingEnabled });
  return state.turnEvalLoggingEnabled;
}

clearDialogueDiagnostics();
