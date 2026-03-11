package web

import "github.com/krystophny/tabura/internal/store"

func isExplicitWorkspace(workspace store.Workspace) bool {
	return !workspace.IsDaily
}

func filterExplicitWorkspaces(workspaces []store.Workspace) []store.Workspace {
	filtered := make([]store.Workspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		if isExplicitWorkspace(workspace) {
			filtered = append(filtered, workspace)
		}
	}
	return filtered
}

func activeExplicitWorkspace(workspaces []store.Workspace) *store.Workspace {
	for i := range workspaces {
		if workspaces[i].IsActive && isExplicitWorkspace(workspaces[i]) {
			return &workspaces[i]
		}
	}
	return nil
}
