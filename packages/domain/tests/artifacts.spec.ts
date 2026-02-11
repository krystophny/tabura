import { describe, expect, it } from "vitest";
import { buildArtifactBundleV1, isPatchLikeContent } from "../src/artifacts";

describe("artifact bundle", () => {
  it("builds bundle with required text and optional diff", () => {
    const bundle = buildArtifactBundleV1("turn-1", [
      { id: "a1", turnId: "turn-1", type: "text_response", storagePath: "canonical-text.md" },
      { id: "a2", turnId: "turn-1", type: "diff_patch", storagePath: "latest.patch" }
    ]);

    expect(bundle.primaryTextArtifactId).toBe("a1");
    expect(bundle.diffArtifactId).toBe("a2");
  });

  it("throws when canonical text is missing", () => {
    expect(() =>
      buildArtifactBundleV1("turn-1", [
        { id: "a2", turnId: "turn-1", type: "diff_patch", storagePath: "latest.patch" }
      ])
    ).toThrow(/no canonical text artifact/);
  });

  it("detects patch-like content", () => {
    expect(isPatchLikeContent("diff --git a/file b/file\n@@")).toBe(true);
    expect(isPatchLikeContent("plain response text")).toBe(false);
  });
});
