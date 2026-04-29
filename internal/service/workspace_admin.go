package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	v1 "ruleGoKratos/api/rulego/v1"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	codeWorkspaceRootDir = "/app/code_workspace"
	repoSyncTimeout      = 10 * time.Minute
)

type workspaceWriteReq struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	RepositoryURLs []string `json:"repositoryUrls"`
}

type workspaceDTO struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	RootDir        string              `json:"rootDir"`
	ConfigFile     string              `json:"configFile"`
	RepositoryURLs []string            `json:"repositoryUrls"`
	Repositories   []workspaceRepoItem `json:"repositories"`
	CreatedAt      string              `json:"createdAt"`
	UpdatedAt      string              `json:"updatedAt"`
}

type workspaceRepoItem struct {
	URL string `json:"url"`
	Dir string `json:"dir"`
}

type workspaceFile struct {
	Folders    []workspaceFolder      `json:"folders"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	Meta       workspaceMeta          `json:"ruleGoWorkspace"`
}

type workspaceFolder struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

type workspaceMeta struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	RootDir      string              `json:"rootDir"`
	Repositories []workspaceRepoItem `json:"repositories"`
	CreatedAt    string              `json:"createdAt"`
	UpdatedAt    string              `json:"updatedAt"`
}

func (s *AdminService) ListWorkspaces(ctx context.Context, _ *v1.ListWorkspacesRequest) (*v1.ListWorkspacesReply, error) {
	items, err := listAllWorkspaceDTOs()
	if err != nil {
		return nil, err
	}
	resp := &v1.ListWorkspacesReply{
		Items: make([]*v1.WorkspaceItem, 0, len(items)),
	}
	for _, item := range items {
		resp.Items = append(resp.Items, workspaceToProto(item))
	}
	return resp, nil
}

func (s *AdminService) GetWorkspace(ctx context.Context, req *v1.GetWorkspaceRequest) (*v1.GetWorkspaceReply, error) {
	item, err := loadWorkspaceDTO(strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	return &v1.GetWorkspaceReply{Item: workspaceToProto(*item)}, nil
}

func (s *AdminService) CreateWorkspace(ctx context.Context, req *v1.CreateWorkspaceRequest) (*v1.CreateWorkspaceReply, error) {
	writeReq := workspaceWriteReq{
		ID:             req.GetId(),
		Name:           req.GetName(),
		Description:    req.GetDescription(),
		RepositoryURLs: req.GetRepositoryUrls(),
	}
	if strings.TrimSpace(writeReq.ID) == "" {
		writeReq.ID = uuid.NewString()
	}
	payload, err := normalizeWorkspaceReq(writeReq, true)
	if err != nil {
		return nil, err
	}
	if _, err := loadWorkspaceDTO(payload.ID); err == nil {
		return nil, errors.New("工作区 id 已存在")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := syncWorkspaceRepos(ctx, payload.ID, payload.Repositories); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := writeWorkspaceFile(payload, now, now); err != nil {
		return nil, err
	}
	item, err := loadWorkspaceDTO(payload.ID)
	if err != nil {
		return nil, err
	}
	return &v1.CreateWorkspaceReply{Item: workspaceToProto(*item)}, nil
}

func (s *AdminService) UpdateWorkspace(ctx context.Context, req *v1.UpdateWorkspaceRequest) (*v1.UpdateWorkspaceReply, error) {
	writeReq := workspaceWriteReq{
		ID:             req.GetId(),
		Name:           req.GetName(),
		Description:    req.GetDescription(),
		RepositoryURLs: req.GetRepositoryUrls(),
	}
	payload, err := normalizeWorkspaceReq(writeReq, false)
	if err != nil {
		return nil, err
	}
	prev, err := loadWorkspaceDTO(payload.ID)
	if err != nil {
		return nil, err
	}
	if err := syncWorkspaceRepos(ctx, payload.ID, payload.Repositories); err != nil {
		return nil, err
	}
	createdAt := prev.CreatedAt
	if strings.TrimSpace(createdAt) == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	if err := writeWorkspaceFile(payload, createdAt, updatedAt); err != nil {
		return nil, err
	}
	item, err := loadWorkspaceDTO(payload.ID)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateWorkspaceReply{Item: workspaceToProto(*item)}, nil
}

// RunWorkspaceRepoSync 与工作区管理中「同步仓库」及 SyncWorkspace RPC 行为一致：对磁盘上的 git 仓库执行 pull/clone，并更新 .code-workspace 的 updatedAt。
func RunWorkspaceRepoSync(ctx context.Context, workspaceID string) error {
	id, err := normalizeWorkspaceID(workspaceID)
	if err != nil {
		return err
	}
	item, err := loadWorkspaceDTO(id)
	if err != nil {
		return err
	}
	if err := syncWorkspaceRepos(ctx, id, item.Repositories); err != nil {
		return err
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	payload := &normalizedWorkspacePayload{
		ID:           item.ID,
		Name:         item.Name,
		Description:  item.Description,
		Repositories: item.Repositories,
	}
	return writeWorkspaceFile(payload, item.CreatedAt, updatedAt)
}

func (s *AdminService) SyncWorkspace(ctx context.Context, req *v1.SyncWorkspaceRequest) (*v1.SyncWorkspaceReply, error) {
	if err := RunWorkspaceRepoSync(ctx, req.GetId()); err != nil {
		return nil, err
	}
	id, err := normalizeWorkspaceID(req.GetId())
	if err != nil {
		return nil, err
	}
	newItem, err := loadWorkspaceDTO(id)
	if err != nil {
		return nil, err
	}
	return &v1.SyncWorkspaceReply{
		Item:    workspaceToProto(*newItem),
		Message: "仓库同步完成",
	}, nil
}

func (s *AdminService) DeleteWorkspace(ctx context.Context, req *v1.DeleteWorkspaceRequest) (*v1.DeleteWorkspaceReply, error) {
	id, err := normalizeWorkspaceID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := os.Remove(workspaceConfigPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.RemoveAll(workspaceRootPath(id)); err != nil {
		return nil, err
	}
	return &v1.DeleteWorkspaceReply{Ok: true}, nil
}

type normalizedWorkspacePayload struct {
	ID           string
	Name         string
	Description  string
	Repositories []workspaceRepoItem
}

func normalizeWorkspaceReq(req workspaceWriteReq, requireID bool) (*normalizedWorkspacePayload, error) {
	id := strings.TrimSpace(req.ID)
	if requireID || id != "" {
		nid, err := normalizeWorkspaceID(id)
		if err != nil {
			return nil, err
		}
		id = nid
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name 不能为空")
	}
	repos, err := normalizeRepositories(req.RepositoryURLs)
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		return nil, errors.New("至少配置一个 git 仓库地址")
	}
	return &normalizedWorkspacePayload{
		ID:           id,
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		Repositories: repos,
	}, nil
}

func normalizeWorkspaceID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("id 不能为空")
	}
	for _, ch := range id {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.'
		if !ok {
			return "", errors.New("id 仅支持字母、数字、-、_、.")
		}
	}
	return id, nil
}

func normalizeRepositories(urls []string) ([]workspaceRepoItem, error) {
	seenURL := make(map[string]struct{})
	usedDir := make(map[string]struct{})
	out := make([]workspaceRepoItem, 0, len(urls))
	for _, raw := range urls {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if _, ok := seenURL[u]; ok {
			continue
		}
		seenURL[u] = struct{}{}
		dir := uniqueRepoDirName(repoDirNameFromURL(u), usedDir)
		usedDir[dir] = struct{}{}
		out = append(out, workspaceRepoItem{URL: u, Dir: dir})
	}
	return out, nil
}

func repoDirNameFromURL(raw string) string {
	u := strings.TrimSpace(raw)
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")
	parts := strings.Split(u, "/")
	last := parts[len(parts)-1]
	if i := strings.LastIndex(last, ":"); i >= 0 {
		last = last[i+1:]
	}
	last = strings.TrimSpace(last)
	if last == "" {
		return "repo"
	}
	var b strings.Builder
	for _, ch := range last {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.'
		if ok {
			b.WriteRune(ch)
		} else {
			b.WriteByte('-')
		}
	}
	name := strings.Trim(strings.TrimSpace(b.String()), "-")
	if name == "" {
		return "repo"
	}
	return name
}

func uniqueRepoDirName(base string, used map[string]struct{}) string {
	if _, ok := used[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
}

func ensureWorkspaceRootDir() error {
	return os.MkdirAll(codeWorkspaceRootDir, 0o755)
}

func workspaceConfigPath(id string) string {
	return filepath.Join(codeWorkspaceRootDir, id+".code-workspace")
}

func workspaceRootPath(id string) string {
	return filepath.Join(codeWorkspaceRootDir, id)
}

func syncWorkspaceRepos(ctx context.Context, workspaceID string, repos []workspaceRepoItem) error {
	if err := ensureWorkspaceRootDir(); err != nil {
		return err
	}
	root := workspaceRootPath(workspaceID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	for _, repo := range repos {
		repoDir := filepath.Join(root, repo.Dir)
		gitDir := filepath.Join(repoDir, ".git")
		if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
			if _, err := runGitCommand(ctx, []string{"-C", repoDir, "pull", "--ff-only"}...); err != nil {
				return fmt.Errorf("更新仓库失败（%s）: %w", repo.URL, err)
			}
			continue
		}
		if st, err := os.Stat(repoDir); err == nil && st.IsDir() {
			entries, _ := os.ReadDir(repoDir)
			if len(entries) > 0 {
				return fmt.Errorf("目录已存在且非 git 仓库：%s", repoDir)
			}
		}
		if _, err := runGitCommand(ctx, "clone", repo.URL, repoDir); err != nil {
			return fmt.Errorf("拉取仓库失败（%s）: %w", repo.URL, err)
		}
	}
	return nil
}

func runGitCommand(ctx context.Context, args ...string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, repoSyncTimeout)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, "git", args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("容器内未安装 git，请在运行镜像中安装 git 后重试")
		}
		if text == "" {
			return "", err
		}
		return text, errors.New(text)
	}
	return text, nil
}

func writeWorkspaceFile(payload *normalizedWorkspacePayload, createdAt, updatedAt string) error {
	if err := ensureWorkspaceRootDir(); err != nil {
		return err
	}
	root := workspaceRootPath(payload.ID)
	cfg := workspaceFile{
		Folders: make([]workspaceFolder, 0, len(payload.Repositories)),
		Settings: map[string]interface{}{
			"rulego.workspace.id":   payload.ID,
			"rulego.workspace.name": payload.Name,
		},
		Extensions: map[string]interface{}{
			"recommendations": []string{},
		},
		Meta: workspaceMeta{
			ID:           payload.ID,
			Name:         payload.Name,
			Description:  payload.Description,
			RootDir:      root,
			Repositories: payload.Repositories,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		},
	}
	for _, repo := range payload.Repositories {
		cfg.Folders = append(cfg.Folders, workspaceFolder{
			Path: filepath.ToSlash(filepath.Join(payload.ID, repo.Dir)),
			Name: repo.Dir,
		})
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(workspaceConfigPath(payload.ID), b, 0o644)
}

func listAllWorkspaceDTOs() ([]workspaceDTO, error) {
	if err := ensureWorkspaceRootDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(codeWorkspaceRootDir)
	if err != nil {
		return nil, err
	}
	out := make([]workspaceDTO, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".code-workspace") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".code-workspace")
		item, err := loadWorkspaceDTO(id)
		if err != nil {
			continue
		}
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out, nil
}

func loadWorkspaceDTO(id string) (*workspaceDTO, error) {
	id, err := normalizeWorkspaceID(id)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(workspaceConfigPath(id))
	if err != nil {
		return nil, err
	}
	var cfg workspaceFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Meta.ID == "" {
		cfg.Meta.ID = id
	}
	root := cfg.Meta.RootDir
	if strings.TrimSpace(root) == "" {
		root = workspaceRootPath(id)
	}
	repoURLs := make([]string, 0, len(cfg.Meta.Repositories))
	for _, r := range cfg.Meta.Repositories {
		repoURLs = append(repoURLs, r.URL)
	}
	return &workspaceDTO{
		ID:             cfg.Meta.ID,
		Name:           cfg.Meta.Name,
		Description:    cfg.Meta.Description,
		RootDir:        root,
		ConfigFile:     workspaceConfigPath(id),
		RepositoryURLs: repoURLs,
		Repositories:   cfg.Meta.Repositories,
		CreatedAt:      cfg.Meta.CreatedAt,
		UpdatedAt:      cfg.Meta.UpdatedAt,
	}, nil
}

func workspaceToProto(in workspaceDTO) *v1.WorkspaceItem {
	repoItems := make([]*v1.WorkspaceRepoItem, 0, len(in.Repositories))
	for _, repo := range in.Repositories {
		repoItems = append(repoItems, &v1.WorkspaceRepoItem{
			Url: repo.URL,
			Dir: repo.Dir,
		})
	}
	return &v1.WorkspaceItem{
		Id:             in.ID,
		Name:           in.Name,
		Description:    in.Description,
		RootDir:        in.RootDir,
		ConfigFile:     in.ConfigFile,
		RepositoryUrls: in.RepositoryURLs,
		Repositories:   repoItems,
		CreatedAt:      in.CreatedAt,
		UpdatedAt:      in.UpdatedAt,
	}
}
