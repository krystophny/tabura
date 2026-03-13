import * as context from './app-context.js';

const { refs, state } = context;
const parseOptionalBoolean = (...args) => refs.parseOptionalBoolean(...args);

export function applyTurnRuntimePreferences(source) {
  state.turnPolicyProfile = String(source?.turn_policy_profile || state.turnPolicyProfile || 'balanced').trim().toLowerCase() || 'balanced';
  const turnEvalLoggingEnabled = parseOptionalBoolean(source?.turn_eval_logging_enabled);
  state.turnEvalLoggingEnabled = turnEvalLoggingEnabled !== null
    ? turnEvalLoggingEnabled
    : state.turnEvalLoggingEnabled !== false;
  if (state.dialogueDiagnostics) {
    state.dialogueDiagnostics.profile = state.turnPolicyProfile;
    state.dialogueDiagnostics.evalLoggingEnabled = state.turnEvalLoggingEnabled;
  }
}
