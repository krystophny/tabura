import type { ModeState, WarningRecord } from "./types";

const GLOBAL_WRITE_RISK_RE = /\b(rm\s+-rf|sudo\b|chmod\b|chown\b|mv\b|cp\b|tee\b|truncate\b)\b/i;

export function evaluatePromptWarnings(input: {
  mode: ModeState;
  prompt: string;
  planLockHash?: string | null;
}): WarningRecord[] {
  const out: WarningRecord[] = [];

  if (
    input.mode.contextMode === "project" &&
    input.mode.cognitiveMode === "execution" &&
    !input.planLockHash?.trim()
  ) {
    out.push({
      code: "plan_lock_missing",
      message: "Execution mode is active without a plan lock hash. v1 continues with advisory warning.",
      severity: "warning",
      blocking: false
    });
  }

  if (input.mode.contextMode === "global" && GLOBAL_WRITE_RISK_RE.test(input.prompt)) {
    out.push({
      code: "global_write_risk",
      message: "Prompt appears to include risky global write/exec intent. v1 continues with advisory warning.",
      severity: "warning",
      blocking: false
    });
  }

  return out;
}
