export type ContextMode = "global" | "project";

export type CognitiveMode = "dialogue" | "plan" | "execution" | "review";

export type ModeState = {
  contextMode: ContextMode;
  cognitiveMode: CognitiveMode;
};

export type WarningCode = "plan_lock_missing" | "global_write_risk";

export type WarningSeverity = "info" | "warning";

export type WarningRecord = {
  code: WarningCode;
  message: string;
  severity: WarningSeverity;
  blocking: false;
};

export type PromptRoute = {
  sessionId: string;
  payload: string;
};

export type ArtifactKind = "text_response" | "diff_patch";

export type ArtifactRecord = {
  id: string;
  turnId: string;
  type: ArtifactKind;
  storagePath: string;
};

export type ArtifactBundleV1 = {
  turnId: string;
  primaryTextArtifactId: string;
  diffArtifactId?: string;
};
