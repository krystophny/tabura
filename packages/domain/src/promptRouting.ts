import type { PromptRoute } from "./types";

export function routePromptToSession(input: {
  sessionId: string | null;
  prompt: string;
}):
  | { ok: true; route: PromptRoute }
  | { ok: false; errorCode: "session_missing" | "prompt_empty"; message: string } {
  const prompt = input.prompt.trim();
  if (!prompt) {
    return { ok: false, errorCode: "prompt_empty", message: "Prompt must not be empty." };
  }

  if (!input.sessionId) {
    return {
      ok: false,
      errorCode: "session_missing",
      message: "No active session. Start or attach a local session first."
    };
  }

  return {
    ok: true,
    route: {
      sessionId: input.sessionId,
      payload: `${prompt}\n`
    }
  };
}
