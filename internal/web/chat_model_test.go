package web

import (
	"testing"

	"github.com/krystophny/tabura/internal/modelprofile"
	"github.com/krystophny/tabura/internal/store"
)

func TestEffectiveProjectChatModelFallsBackToLocal(t *testing.T) {
	app := newAuthedTestApp(t)
	app.appServerModel = modelprofile.ModelSpark

	project := store.Workspace{}
	if got := app.effectiveWorkspaceChatModelAlias(project); got != modelprofile.AliasLocal {
		t.Fatalf("effectiveWorkspaceChatModelAlias() = %q, want %q", got, modelprofile.AliasLocal)
	}
	if got := app.effectiveWorkspaceChatModelReasoningEffort(project); got != modelprofile.ReasoningNone {
		t.Fatalf("effectiveWorkspaceChatModelReasoningEffort() = %q, want %q", got, modelprofile.ReasoningNone)
	}
}

func TestEnforceLocalWorkspaceModelDefaultsMigratesPersistedRemoteDefaults(t *testing.T) {
	app := newAuthedTestApp(t)

	defaultProject, err := app.ensureDefaultWorkspace()
	if err != nil {
		t.Fatalf("ensureDefaultWorkspace: %v", err)
	}
	if err := app.store.UpdateEnrichedWorkspaceChatModel(workspaceIDStr(defaultProject.ID), modelprofile.AliasSpark); err != nil {
		t.Fatalf("UpdateEnrichedWorkspaceChatModel(default): %v", err)
	}
	if err := app.store.UpdateEnrichedWorkspaceChatModelReasoningEffort(workspaceIDStr(defaultProject.ID), modelprofile.ReasoningLow); err != nil {
		t.Fatalf("UpdateEnrichedWorkspaceChatModelReasoningEffort(default): %v", err)
	}
	other, err := app.store.CreateEnrichedWorkspace("Other", "/tmp/other", "/tmp/other", "managed", "", "other", false)
	if err != nil {
		t.Fatalf("CreateEnrichedWorkspace: %v", err)
	}
	if err := app.store.UpdateEnrichedWorkspaceChatModel(workspaceIDStr(other.ID), modelprofile.AliasGPT); err != nil {
		t.Fatalf("UpdateEnrichedWorkspaceChatModel(other): %v", err)
	}
	if err := app.store.UpdateEnrichedWorkspaceChatModelReasoningEffort(workspaceIDStr(other.ID), modelprofile.ReasoningHigh); err != nil {
		t.Fatalf("UpdateEnrichedWorkspaceChatModelReasoningEffort(other): %v", err)
	}
	if err := app.store.SetAppState(appStateDefaultChatModelKey, modelprofile.AliasSpark); err != nil {
		t.Fatalf("SetAppState(default_chat_model): %v", err)
	}

	if err := enforceLocalWorkspaceModelDefaults(app.store); err != nil {
		t.Fatalf("enforceLocalWorkspaceModelDefaults(): %v", err)
	}

	defaultUpdated, err := app.store.GetEnrichedWorkspace(workspaceIDStr(defaultProject.ID))
	if err != nil {
		t.Fatalf("GetEnrichedWorkspace(default): %v", err)
	}
	if got := defaultUpdated.ChatModel; got != modelprofile.AliasLocal {
		t.Fatalf("default project chat_model = %q, want %q", got, modelprofile.AliasLocal)
	}
	if got := defaultUpdated.ChatModelReasoningEffort; got != modelprofile.ReasoningNone {
		t.Fatalf("default project chat_model_reasoning_effort = %q, want %q", got, modelprofile.ReasoningNone)
	}

	otherUpdated, err := app.store.GetEnrichedWorkspace(workspaceIDStr(other.ID))
	if err != nil {
		t.Fatalf("GetEnrichedWorkspace(other): %v", err)
	}
	if got := otherUpdated.ChatModel; got != modelprofile.AliasLocal {
		t.Fatalf("other project chat_model = %q, want %q", got, modelprofile.AliasLocal)
	}
	if got := otherUpdated.ChatModelReasoningEffort; got != modelprofile.ReasoningNone {
		t.Fatalf("other project chat_model_reasoning_effort = %q, want %q", got, modelprofile.ReasoningNone)
	}

	defaultAlias, err := app.store.AppState(appStateDefaultChatModelKey)
	if err != nil {
		t.Fatalf("AppState(default_chat_model): %v", err)
	}
	if defaultAlias != modelprofile.AliasLocal {
		t.Fatalf("default_chat_model = %q, want %q", defaultAlias, modelprofile.AliasLocal)
	}
}

func TestEffectiveProjectChatModelDefaultsToLocal(t *testing.T) {
	app := newAuthedTestApp(t)
	app.appServerModel = ""

	project := store.Workspace{}
	if got := app.effectiveWorkspaceChatModelAlias(project); got != modelprofile.AliasLocal {
		t.Fatalf("effectiveWorkspaceChatModelAlias() = %q, want %q", got, modelprofile.AliasLocal)
	}
	if got := app.effectiveWorkspaceChatModelReasoningEffort(project); got != modelprofile.ReasoningNone {
		t.Fatalf("effectiveWorkspaceChatModelReasoningEffort() = %q, want %q", got, modelprofile.ReasoningNone)
	}
}

func TestAppServerModelProfileForProjectUsesStoredAliasAndNormalizesReasoning(t *testing.T) {
	app := newAuthedTestApp(t)

	profile := app.appServerModelProfileForWorkspace(store.Workspace{
		ChatModel:                modelprofile.AliasGPT,
		ChatModelReasoningEffort: "minimal",
	})
	if profile.Alias != modelprofile.AliasGPT {
		t.Fatalf("profile.Alias = %q, want %q", profile.Alias, modelprofile.AliasGPT)
	}
	if profile.Model != modelprofile.ModelGPT {
		t.Fatalf("profile.Model = %q, want %q", profile.Model, modelprofile.ModelGPT)
	}
	if profile.ThreadParams != nil {
		t.Fatalf("profile.ThreadParams = %#v, want nil", profile.ThreadParams)
	}
	if got := profile.TurnParams["effort"]; got != modelprofile.ReasoningHigh {
		t.Fatalf("profile.TurnParams[effort] = %#v, want %q", got, modelprofile.ReasoningHigh)
	}
}

func TestAppServerModelProfileForWorkspacePathFallsBackWhenProjectMissing(t *testing.T) {
	app := newAuthedTestApp(t)
	app.appServerModel = modelprofile.ModelSpark

	profile := app.appServerModelProfileForWorkspacePath("missing-project")
	if profile.Alias != modelprofile.AliasLocal {
		t.Fatalf("profile.Alias = %q, want %q", profile.Alias, modelprofile.AliasLocal)
	}
	if profile.Model != modelprofile.ModelSpark {
		t.Fatalf("profile.Model = %q, want %q", profile.Model, modelprofile.ModelSpark)
	}
	if got := profile.TurnParams["effort"]; got != modelprofile.ReasoningNone {
		t.Fatalf("profile.TurnParams[effort] = %#v, want %q", got, modelprofile.ReasoningNone)
	}
}

func TestAssistantTurnModeHonorsConfiguredRouting(t *testing.T) {
	app := newAuthedTestApp(t)

	app.assistantMode = assistantModeLocal
	if got := app.assistantTurnMode(false); got != assistantModeLocal {
		t.Fatalf("assistantTurnMode(local) = %q, want %q", got, assistantModeLocal)
	}

	app.assistantMode = assistantModeCodex
	if got := app.assistantTurnMode(false); got != assistantModeCodex {
		t.Fatalf("assistantTurnMode(codex) = %q, want %q", got, assistantModeCodex)
	}

	app.assistantMode = assistantModeAuto
	app.appServerClient = nil
	if got := app.assistantTurnMode(false); got != assistantModeLocal {
		t.Fatalf("assistantTurnMode(auto, no app-server) = %q, want %q", got, assistantModeLocal)
	}
	if got := app.assistantTurnMode(true); got != assistantModeLocal {
		t.Fatalf("assistantTurnMode(localOnly) = %q, want %q", got, assistantModeLocal)
	}
}
