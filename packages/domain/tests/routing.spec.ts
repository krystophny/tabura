import { describe, expect, it } from "vitest";
import { routePromptToSession } from "../src/promptRouting";

describe("routePromptToSession", () => {
  it("routes prompt to active session", () => {
    const result = routePromptToSession({ sessionId: "s1", prompt: "hello" });
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.route.payload).toBe("hello\n");
    }
  });

  it("fails when session missing", () => {
    const result = routePromptToSession({ sessionId: null, prompt: "hello" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.errorCode).toBe("session_missing");
    }
  });
});
