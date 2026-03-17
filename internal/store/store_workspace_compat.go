package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	appStateActiveProjectIDKey = "active_project_id"
	projectNameStatePrefix     = "project_name:"
	projectWorkspacePathPrefix = "project_workspace_path:"
	projectRootPathPrefix      = "project_root_path:"
	projectKindStatePrefix     = "project_kind:"
)

type Project struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	WorkspacePath            string `json:"workspace_path"`
	RootPath                 string `json:"root_path"`
	Kind                     string `json:"kind"`
	MCPURL                   string `json:"mcp_url,omitempty"`
	CanvasSessionID          string `json:"canvas_session_id"`
	ChatModel                string `json:"chat_model"`
	ChatModelReasoningEffort string `json:"chat_model_reasoning_effort"`
	CompanionConfigJSON      string `json:"-"`
	IsDefault                bool   `json:"is_default"`
	CreatedAt                int64  `json:"created_at"`
	UpdatedAt                int64  `json:"updated_at"`
	LastOpenedAt             int64  `json:"last_opened_at"`
}

func workspaceIDString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func parseWorkspaceIDString(id string) (int64, error) {
	workspaceID, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil || workspaceID <= 0 {
		return 0, sql.ErrNoRows
	}
	return workspaceID, nil
}

func parseWorkspaceTimestamp(value string) int64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.Unix()
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed.Unix()
	}
	return 0
}

func projectKindStateKey(id int64) string {
	return projectKindStatePrefix + strconv.FormatInt(id, 10)
}

func projectNameStateKey(id int64) string {
	return projectNameStatePrefix + strconv.FormatInt(id, 10)
}

func projectWorkspacePathStateKey(id int64) string {
	return projectWorkspacePathPrefix + strconv.FormatInt(id, 10)
}

func projectRootPathStateKey(id int64) string {
	return projectRootPathPrefix + strconv.FormatInt(id, 10)
}

func (s *Store) activeProjectID() (string, error) {
	return s.AppState(appStateActiveProjectIDKey)
}

func (s *Store) compatibilityWorkspacePath(workspaceID int64, defaultPath string) string {
	if workspaceID <= 0 {
		return normalizeWorkspacePath(defaultPath)
	}
	if path, err := s.AppState(projectWorkspacePathStateKey(workspaceID)); err == nil && strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path)
	}
	return normalizeWorkspacePath(defaultPath)
}

func (s *Store) projectKindForWorkspace(workspace Workspace) string {
	if kind, err := s.AppState(projectKindStateKey(workspace.ID)); err == nil {
		switch clean := strings.ToLower(strings.TrimSpace(kind)); clean {
		case "managed", "linked", "meeting", "task":
			return clean
		}
	}
	return "workspace"
}

func (s *Store) projectNameForWorkspace(workspace Workspace) string {
	if name, err := s.AppState(projectNameStateKey(workspace.ID)); err == nil && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return workspace.Name
}

func (s *Store) projectWorkspacePathForWorkspace(workspace Workspace) string {
	if path, err := s.AppState(projectWorkspacePathStateKey(workspace.ID)); err == nil && strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path)
	}
	return workspace.DirPath
}

func (s *Store) projectRootPathForWorkspace(workspace Workspace) string {
	if path, err := s.AppState(projectRootPathStateKey(workspace.ID)); err == nil && strings.TrimSpace(path) != "" {
		return normalizeWorkspacePath(path)
	}
	return workspace.DirPath
}

func (s *Store) projectFromWorkspace(workspace Workspace) (Project, error) {
	activeProjectID, err := s.activeProjectID()
	if err != nil {
		return Project{}, err
	}
	return Project{
		ID:                       workspaceIDString(workspace.ID),
		Name:                     s.projectNameForWorkspace(workspace),
		WorkspacePath:            s.projectWorkspacePathForWorkspace(workspace),
		RootPath:                 s.projectRootPathForWorkspace(workspace),
		Kind:                     s.projectKindForWorkspace(workspace),
		MCPURL:                   workspace.MCPURL,
		CanvasSessionID:          workspace.CanvasSessionID,
		ChatModel:                normalizeWorkspaceChatModel(workspace.ChatModel),
		ChatModelReasoningEffort: normalizeWorkspaceChatModelReasoningEffort(workspace.ChatModelReasoningEffort),
		CompanionConfigJSON:      workspace.CompanionConfigJSON,
		IsDefault:                strings.TrimSpace(activeProjectID) == workspaceIDString(workspace.ID) || strings.EqualFold(strings.TrimSpace(workspace.CanvasSessionID), "local"),
		CreatedAt:                parseWorkspaceTimestamp(workspace.CreatedAt),
		UpdatedAt:                parseWorkspaceTimestamp(workspace.UpdatedAt),
		LastOpenedAt:             parseWorkspaceTimestamp(workspace.UpdatedAt),
	}, nil
}

func (s *Store) ListProjects() ([]Project, error) {
	workspaces, err := s.ListWorkspaces()
	if err != nil {
		return nil, err
	}
	out := make([]Project, 0, len(workspaces))
	for _, workspace := range workspaces {
		if workspace.IsDaily {
			continue
		}
		project, err := s.projectFromWorkspace(workspace)
		if err != nil {
			return nil, err
		}
		out = append(out, project)
	}
	if len(out) == 0 {
		if workspace, err := s.ActiveWorkspace(); err == nil {
			project, projectErr := s.projectFromWorkspace(workspace)
			if projectErr != nil {
				return nil, projectErr
			}
			out = append(out, project)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) GetProject(id string) (Project, error) {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return Project{}, err
	}
	workspace, err := s.GetWorkspace(workspaceID)
	if err != nil {
		return Project{}, err
	}
	return s.projectFromWorkspace(workspace)
}

func (s *Store) GetProjectByWorkspacePath(workspacePath string) (Project, error) {
	rawPath := strings.TrimSpace(workspacePath)
	cleanPath := normalizeWorkspacePath(workspacePath)
	workspace, err := s.GetWorkspaceByPath(cleanPath)
	if err == nil {
		return s.projectFromWorkspace(workspace)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Project{}, err
	}
	workspaces, err := s.ListWorkspaces()
	if err != nil {
		return Project{}, err
	}
	for _, workspace := range workspaces {
		if workspace.IsDaily {
			continue
		}
		projectPath := strings.TrimSpace(s.projectWorkspacePathForWorkspace(workspace))
		switch {
		case projectPath != "" && projectPath == rawPath:
			return s.projectFromWorkspace(workspace)
		case filepath.IsAbs(projectPath) && projectPath == cleanPath:
			return s.projectFromWorkspace(workspace)
		}
	}
	return Project{}, sql.ErrNoRows
}

func (s *Store) GetProjectByRootPath(rootPath string) (Project, error) {
	cleanPath := normalizeWorkspacePath(rootPath)
	workspaces, err := s.ListWorkspaces()
	if err != nil {
		return Project{}, err
	}
	for _, workspace := range workspaces {
		if workspace.IsDaily {
			continue
		}
		if strings.TrimSpace(s.projectRootPathForWorkspace(workspace)) == cleanPath {
			return s.projectFromWorkspace(workspace)
		}
	}
	return s.GetProjectByWorkspacePath(cleanPath)
}

func (s *Store) GetProjectByCanvasSession(canvasSessionID string) (Project, error) {
	workspaces, err := s.ListWorkspaces()
	if err != nil {
		return Project{}, err
	}
	clean := strings.TrimSpace(canvasSessionID)
	for _, workspace := range workspaces {
		if strings.TrimSpace(workspace.CanvasSessionID) == clean {
			return s.projectFromWorkspace(workspace)
		}
	}
	return Project{}, sql.ErrNoRows
}

func (s *Store) CreateProject(name, workspacePath, rootPath, kind, mcpURL, canvasSessionID string, isDefault bool) (Project, error) {
	sphere := SpherePrivate
	cleanRootPath := normalizeWorkspacePath(rootPath)
	cleanWorkspacePath := strings.TrimSpace(workspacePath)
	if cleanWorkspacePath == "" {
		cleanWorkspacePath = cleanRootPath
	}
	targetPath := cleanRootPath
	if targetPath == "" {
		targetPath = normalizeWorkspacePath(cleanWorkspacePath)
	}
	if targetPath != "" {
		if workspace, err := s.GetWorkspaceByPath(targetPath); err == nil && strings.TrimSpace(workspace.Sphere) != "" {
			sphere = workspace.Sphere
		} else if !errors.Is(err, sql.ErrNoRows) && err != nil {
			return Project{}, err
		} else if workspaceID, findErr := s.FindWorkspaceContainingPath(targetPath); findErr == nil && workspaceID != nil {
			workspace, getErr := s.GetWorkspace(*workspaceID)
			if getErr != nil {
				return Project{}, getErr
			}
			if strings.TrimSpace(workspace.Sphere) != "" {
				sphere = workspace.Sphere
			}
		} else if findErr != nil {
			return Project{}, findErr
		}
	}
	if sphere == SpherePrivate {
		if activeSphere, err := s.ActiveSphere(); err == nil && strings.TrimSpace(activeSphere) != "" {
			sphere = activeSphere
		}
	}
	workspace, err := s.CreateWorkspace(name, cleanRootPath, sphere)
	if err != nil {
		return Project{}, err
	}
	if err := s.SetAppState(projectNameStateKey(workspace.ID), strings.TrimSpace(name)); err != nil {
		return Project{}, err
	}
	if err := s.SetAppState(projectWorkspacePathStateKey(workspace.ID), cleanWorkspacePath); err != nil {
		return Project{}, err
	}
	if err := s.SetAppState(projectRootPathStateKey(workspace.ID), cleanRootPath); err != nil {
		return Project{}, err
	}
	if cleanKind := strings.ToLower(strings.TrimSpace(kind)); cleanKind != "" {
		if err := s.SetAppState(projectKindStateKey(workspace.ID), cleanKind); err != nil {
			return Project{}, err
		}
	}
	if strings.TrimSpace(mcpURL) != "" {
		if updated, updateErr := s.UpdateWorkspaceMCPURL(workspace.ID, mcpURL); updateErr == nil {
			workspace = updated
		} else {
			return Project{}, updateErr
		}
	}
	if strings.TrimSpace(canvasSessionID) != "" {
		if updated, updateErr := s.UpdateWorkspaceCanvasSession(workspace.ID, canvasSessionID); updateErr == nil {
			workspace = updated
		} else {
			return Project{}, updateErr
		}
	}
	if isDefault {
		if err := s.SetAppState(appStateActiveProjectIDKey, workspaceIDString(workspace.ID)); err != nil {
			return Project{}, err
		}
		if err := s.SetActiveWorkspace(workspace.ID); err != nil {
			return Project{}, err
		}
		workspace, err = s.GetWorkspace(workspace.ID)
		if err != nil {
			return Project{}, err
		}
	}
	return s.projectFromWorkspace(workspace)
}

func (s *Store) UpdateWorkspaceMCPURL(id int64, mcpURL string) (Workspace, error) {
	_, err := s.db.Exec(`UPDATE workspaces SET mcp_url = ?, updated_at = datetime('now') WHERE id = ?`, strings.TrimSpace(mcpURL), id)
	if err != nil {
		return Workspace{}, err
	}
	return s.GetWorkspace(id)
}

func (s *Store) UpdateWorkspaceCanvasSession(id int64, canvasSessionID string) (Workspace, error) {
	_, err := s.db.Exec(`UPDATE workspaces SET canvas_session_id = ?, updated_at = datetime('now') WHERE id = ?`, strings.TrimSpace(canvasSessionID), id)
	if err != nil {
		return Workspace{}, err
	}
	return s.GetWorkspace(id)
}

func (s *Store) SetActiveWorkspaceID(workspaceID string) error {
	workspaceNumericID, err := parseWorkspaceIDString(workspaceID)
	if err != nil {
		return errors.New("workspace id is required")
	}
	if _, err := s.GetWorkspace(workspaceNumericID); err != nil {
		return err
	}
	return s.SetAppState(appStateActiveProjectIDKey, workspaceIDString(workspaceNumericID))
}

func (s *Store) ActiveWorkspaceID() (string, error) {
	if activeProjectID, err := s.activeProjectID(); err == nil && strings.TrimSpace(activeProjectID) != "" {
		return strings.TrimSpace(activeProjectID), nil
	} else if err != nil {
		return "", err
	}
	workspace, err := s.ActiveWorkspace()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if workspace.IsDaily {
		return "", nil
	}
	return workspaceIDString(workspace.ID), nil
}

func (s *Store) TouchProject(id string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE workspaces SET updated_at = datetime('now') WHERE id = ?`, workspaceID)
	return err
}

func (s *Store) UpdateProjectTransport(id, mcpURL, canvasSessionID string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	if _, err := s.UpdateWorkspaceMCPURL(workspaceID, mcpURL); err != nil {
		return err
	}
	_, err = s.UpdateWorkspaceCanvasSession(workspaceID, canvasSessionID)
	return err
}

func (s *Store) UpdateProjectRuntime(id, mcpURL, canvasSessionID string) error {
	return s.UpdateProjectTransport(id, mcpURL, canvasSessionID)
}

func (s *Store) UpdateProjectChatModel(id, model string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	return s.UpdateWorkspaceChatModel(workspaceID, model)
}

func (s *Store) UpdateProjectChatModelReasoningEffort(id, effort string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	return s.UpdateWorkspaceChatModelReasoningEffort(workspaceID, effort)
}

func (s *Store) UpdateProjectKind(id, kind string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	return s.SetAppState(projectKindStateKey(workspaceID), strings.ToLower(strings.TrimSpace(kind)))
}

func (s *Store) RenameProject(id, name, workspacePath, rootPath, kind string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	if cleanName := strings.TrimSpace(name); cleanName != "" {
		if err := s.SetAppState(projectNameStateKey(workspaceID), cleanName); err != nil {
			return err
		}
	}
	if cleanWorkspacePath := strings.TrimSpace(workspacePath); cleanWorkspacePath != "" {
		if err := s.SetAppState(projectWorkspacePathStateKey(workspaceID), cleanWorkspacePath); err != nil {
			return err
		}
	}
	if cleanRootPath := normalizeWorkspacePath(rootPath); cleanRootPath != "" {
		if err := s.SetAppState(projectRootPathStateKey(workspaceID), cleanRootPath); err != nil {
			return err
		}
	}
	if cleanKind := strings.ToLower(strings.TrimSpace(kind)); cleanKind != "" {
		if err := s.SetAppState(projectKindStateKey(workspaceID), cleanKind); err != nil {
			return err
		}
	}
	if cleanRootPath := normalizeWorkspacePath(rootPath); cleanRootPath != "" {
		_, err = s.UpdateWorkspaceLocation(workspaceID, name, cleanRootPath)
		return err
	}
	_, err = s.UpdateWorkspaceName(workspaceID, name)
	return err
}

func (s *Store) UpdateProjectLocation(id, name, workspacePath, rootPath, kind string) error {
	return s.RenameProject(id, name, workspacePath, rootPath, kind)
}

func (s *Store) DeleteProject(workspaceID string) error {
	workspaceNumericID, err := parseWorkspaceIDString(workspaceID)
	if err != nil {
		return err
	}
	if activeProjectID, err := s.activeProjectID(); err == nil && strings.TrimSpace(activeProjectID) == workspaceIDString(workspaceNumericID) {
		if err := s.SetAppState(appStateActiveProjectIDKey, ""); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := s.SetAppState(projectKindStateKey(workspaceNumericID), ""); err != nil {
		return err
	}
	if err := s.SetAppState(projectNameStateKey(workspaceNumericID), ""); err != nil {
		return err
	}
	if err := s.SetAppState(projectWorkspacePathStateKey(workspaceNumericID), ""); err != nil {
		return err
	}
	if err := s.SetAppState(projectRootPathStateKey(workspaceNumericID), ""); err != nil {
		return err
	}
	return s.DeleteWorkspace(workspaceNumericID)
}

func (s *Store) UpdateProjectCompanionConfig(id, configJSON string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	return s.UpdateWorkspaceCompanionConfig(workspaceID, configJSON)
}

func normalizeProjectName(name string) string {
	return normalizeWorkspaceName(name)
}

func normalizeProjectPath(path string) string {
	return normalizeWorkspacePath(path)
}

func normalizeProjectChatModel(raw string) string {
	return normalizeWorkspaceChatModel(raw)
}

func normalizeProjectChatModelReasoningEffort(raw string) string {
	return normalizeWorkspaceChatModelReasoningEffort(raw)
}

func (s *Store) workspaceForProject(project Project) (Workspace, error) {
	workspaceID, err := parseWorkspaceIDString(project.ID)
	if err != nil {
		return Workspace{}, err
	}
	return s.GetWorkspace(workspaceID)
}

func (s *Store) projectForWorkspace(workspace Workspace) (*Project, error) {
	project, err := s.projectFromWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *Store) ensureWorkspaceForLegacyProject(project Project) (Workspace, error) {
	return s.workspaceForProject(project)
}

func (s *Store) ensureWorkspaceForProject(project Project) (Workspace, error) {
	return s.workspaceForProject(project)
}

func (s *Store) ListWorkspacesForProject(workspaceID string) ([]Workspace, error) {
	project, err := s.GetProject(workspaceID)
	if err != nil {
		return nil, err
	}
	workspace, err := s.workspaceForProject(project)
	if err != nil {
		return nil, err
	}
	return []Workspace{workspace}, nil
}

func (s *Store) SetWorkspaceProject(id int64, _ *string) (Workspace, error) {
	return s.GetWorkspace(id)
}

func (s *Store) FindWorkspaceByProjectPath(path string) (*int64, error) {
	return s.FindWorkspaceContainingPath(path)
}

func (s *Store) inferWorkspaceIDForWorkspacePath(dirPath string) (*string, error) {
	if workspace, err := s.GetWorkspaceByPath(dirPath); err == nil {
		id := workspaceIDString(workspace.ID)
		return &id, nil
	}
	return nil, nil
}

func (s *Store) migrateLegacyProjectData() error {
	return nil
}

func (s *Store) purgeLegacyHubData() error {
	return nil
}

func sameWorkspaceID(current *string, want string) bool {
	return current != nil && strings.TrimSpace(*current) == strings.TrimSpace(want)
}

func (s *Store) copyLegacyProjectRuntimeConfigToWorkspace(Project) error {
	return nil
}

func (s *Store) linkContextToLegacyProject(int64, Project) error {
	return nil
}

func (s *Store) projectForWorkspaceID(workspaceID int64) (Project, error) {
	workspace, err := s.GetWorkspace(workspaceID)
	if err != nil {
		return Project{}, err
	}
	return s.projectFromWorkspace(workspace)
}

func (s *Store) appServerModelProfileForProject(project Project) string {
	return normalizeWorkspaceChatModel(project.ChatModel)
}

func (s *Store) appServerModelProfileForWorkspacePath(workspacePath string) string {
	project, err := s.GetProjectByWorkspacePath(workspacePath)
	if err != nil {
		return ""
	}
	return s.appServerModelProfileForProject(project)
}

func (s *Store) UpdateProjectCanvasSession(id, canvasSessionID string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	_, err = s.UpdateWorkspaceCanvasSession(workspaceID, canvasSessionID)
	return err
}

func (s *Store) UpdateProjectMCPURL(id, mcpURL string) error {
	workspaceID, err := parseWorkspaceIDString(id)
	if err != nil {
		return err
	}
	_, err = s.UpdateWorkspaceMCPURL(workspaceID, mcpURL)
	return err
}

func (s *Store) projectByPath(path string) (Project, error) {
	return s.GetProjectByWorkspacePath(path)
}

func (s *Store) activeProject() (Project, error) {
	id, err := s.ActiveWorkspaceID()
	if err != nil {
		return Project{}, err
	}
	return s.GetProject(id)
}

func (s *Store) workspaceIDForWorkspace(workspaceID int64) string {
	return workspaceIDString(workspaceID)
}

func invalidWorkspaceIDError(id string) error {
	return fmt.Errorf("invalid workspace-backed project id: %s", strings.TrimSpace(id))
}
