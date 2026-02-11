export type ContextMode = "global" | "project";
export type CognitiveMode = "dialogue" | "plan" | "execution" | "review";

export type ModeState = {
  context_mode: ContextMode;
  cognitive_mode: CognitiveMode;
};

export type ArtifactSummary = {
  id: string;
  turn_id: string;
  artifact_type: "text_response" | "diff_patch";
  storage_path: string;
  created_at: string;
};

export type ArtifactDetail = ArtifactSummary & {
  body: string;
};

export type WarningRecord = {
  id: string;
  turn_id: string;
  code: string;
  message: string;
  severity: "info" | "warning";
  created_at: string;
};

export type StartSessionResponse = {
  session_id: string;
  profile: "codex" | "shell";
};

export type SubmitPromptResponse = {
  turn_id: string;
  artifact_ids: string[];
  warning_ids: string[];
};

export type TermOutputEvent = {
  session_id: string;
  chunk: string;
};

export type TermStatusEvent = {
  session_id: string;
  running: boolean;
  exit_code: number | null;
};

export type TurnCompletedEvent = {
  turn_id: string;
  artifact_ids: string[];
  warning_ids: string[];
};

export type WarningEmittedEvent = {
  turn_id: string;
  code: string;
  message: string;
  severity: "info" | "warning";
};
