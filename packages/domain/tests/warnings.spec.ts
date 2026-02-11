import { describe, expect, it } from "vitest";
import { evaluatePromptWarnings } from "../src/warnings";

describe("evaluatePromptWarnings", () => {
  it("emits advisory warning for project execution without lock", () => {
    const warnings = evaluatePromptWarnings({
      mode: { contextMode: "project", cognitiveMode: "execution" },
      prompt: "Run tests",
      planLockHash: null
    });

    expect(warnings.some((w) => w.code === "plan_lock_missing")).toBe(true);
    expect(warnings.every((w) => w.blocking === false)).toBe(true);
  });

  it("emits advisory warning for risky global prompt", () => {
    const warnings = evaluatePromptWarnings({
      mode: { contextMode: "global", cognitiveMode: "dialogue" },
      prompt: "sudo rm -rf /tmp/old-artifacts"
    });

    expect(warnings.some((w) => w.code === "global_write_risk")).toBe(true);
    expect(warnings.every((w) => w.blocking === false)).toBe(true);
  });
});
