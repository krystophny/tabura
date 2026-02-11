import type { ArtifactBundleV1, ArtifactRecord } from "./types";

export function buildArtifactBundleV1(turnId: string, artifacts: ArtifactRecord[]): ArtifactBundleV1 {
  const candidates = artifacts.filter((a) => a.turnId === turnId);
  const text = candidates.find((a) => a.type === "text_response");
  if (!text) {
    throw new Error(`turn ${turnId} has no canonical text artifact`);
  }

  const diff = candidates.find((a) => a.type === "diff_patch");
  return {
    turnId,
    primaryTextArtifactId: text.id,
    diffArtifactId: diff?.id
  };
}

export function isPatchLikeContent(content: string): boolean {
  const normalized = content.trimStart();
  return (
    normalized.startsWith("diff --git") ||
    normalized.startsWith("--- ") ||
    normalized.includes("\n+++ ") ||
    normalized.includes("@@")
  );
}
