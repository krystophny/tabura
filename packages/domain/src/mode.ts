import type { CognitiveMode, ModeState } from "./types";

const ALLOWED_TRANSITIONS: Record<CognitiveMode, Set<CognitiveMode>> = {
  dialogue: new Set(["dialogue", "plan"]),
  plan: new Set(["plan", "dialogue", "execution"]),
  execution: new Set(["execution", "review"]),
  review: new Set(["review", "dialogue", "plan"])
};

export type ModeTransitionResult =
  | { ok: true; next: ModeState; changed: boolean }
  | { ok: false; errorCode: "invalid_mode_transition"; message: string; previous: ModeState };

export function transitionMode(previous: ModeState, requested: ModeState): ModeTransitionResult {
  const allowed = ALLOWED_TRANSITIONS[previous.cognitiveMode];
  if (!allowed.has(requested.cognitiveMode)) {
    return {
      ok: false,
      errorCode: "invalid_mode_transition",
      message: `invalid cognitive transition ${previous.cognitiveMode} -> ${requested.cognitiveMode}`,
      previous
    };
  }

  return {
    ok: true,
    next: requested,
    changed:
      previous.contextMode !== requested.contextMode || previous.cognitiveMode !== requested.cognitiveMode
  };
}
