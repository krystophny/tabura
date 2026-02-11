import { describe, expect, it } from "vitest";
import { transitionMode } from "../src/mode";

describe("transitionMode", () => {
  it("accepts valid transition", () => {
    const result = transitionMode(
      { contextMode: "project", cognitiveMode: "plan" },
      { contextMode: "project", cognitiveMode: "execution" }
    );
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.next.cognitiveMode).toBe("execution");
    }
  });

  it("rejects invalid transition", () => {
    const result = transitionMode(
      { contextMode: "global", cognitiveMode: "dialogue" },
      { contextMode: "global", cognitiveMode: "review" }
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.errorCode).toBe("invalid_mode_transition");
    }
  });
});
