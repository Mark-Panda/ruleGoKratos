package biz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	einoskill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/components/tool"
)

type localSkillFilesystemBackend struct{}

func (localSkillFilesystemBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	return nil, errors.New("LsInfo is not supported by skill runtime")
}

func (localSkillFilesystemBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	if req == nil || strings.TrimSpace(req.FilePath) == "" {
		return nil, errors.New("file path is required")
	}
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, err
	}
	return &filesystem.FileContent{Content: string(data)}, nil
}

func (localSkillFilesystemBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	return nil, errors.New("GrepRaw is not supported by skill runtime")
}

func (localSkillFilesystemBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	if req == nil {
		return nil, errors.New("glob request is required")
	}
	base := strings.TrimSpace(req.Path)
	if base == "" {
		return nil, errors.New("glob base path is required")
	}
	matches, err := filepath.Glob(filepath.Join(base, req.Pattern))
	if err != nil {
		return nil, err
	}
	out := make([]filesystem.FileInfo, 0, len(matches))
	for _, p := range matches {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		out = append(out, filesystem.FileInfo{
			Path:       p,
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (localSkillFilesystemBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	return errors.New("Write is not supported by skill runtime")
}

func (localSkillFilesystemBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	return errors.New("Edit is not supported by skill runtime")
}

type multiSkillBackend struct {
	backends []einoskill.Backend
}

func (b multiSkillBackend) List(ctx context.Context) ([]einoskill.FrontMatter, error) {
	seen := make(map[string]struct{})
	out := make([]einoskill.FrontMatter, 0)
	for _, backend := range b.backends {
		items, err := backend.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

func (b multiSkillBackend) Get(ctx context.Context, name string) (einoskill.Skill, error) {
	name = strings.TrimSpace(name)
	for _, backend := range b.backends {
		s, err := backend.Get(ctx, name)
		if err == nil {
			return s, nil
		}
	}
	return einoskill.Skill{}, fmt.Errorf("skill not found: %s", name)
}

type filteredSkillBackend struct {
	base  einoskill.Backend
	allow map[string]struct{}
}

func (b filteredSkillBackend) List(ctx context.Context) ([]einoskill.FrontMatter, error) {
	items, err := b.base.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(b.allow) == 0 {
		return items, nil
	}
	out := make([]einoskill.FrontMatter, 0, len(items))
	for _, item := range items {
		if _, ok := b.allow[item.Name]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (b filteredSkillBackend) Get(ctx context.Context, name string) (einoskill.Skill, error) {
	if len(b.allow) > 0 {
		if _, ok := b.allow[name]; !ok {
			return einoskill.Skill{}, fmt.Errorf("skill not allowed: %s", name)
		}
	}
	return b.base.Get(ctx, name)
}

func skillAllowMap(names []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}

func (uc *AgentUsecase) officialSkillBackend(ctx context.Context, allowlist []string) (einoskill.Backend, error) {
	fe, ok := uc.skillExecutor.(*FileSkillExecutor)
	if !ok {
		return nil, nil
	}
	dirs := fe.Dirs()
	backends := make([]einoskill.Backend, 0, len(dirs))
	fs := localSkillFilesystemBackend{}
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		backend, err := einoskill.NewBackendFromFilesystem(ctx, &einoskill.BackendFromFilesystemConfig{
			Backend: fs,
			BaseDir: dir,
		})
		if err != nil {
			return nil, err
		}
		backends = append(backends, backend)
	}
	if len(backends) == 0 {
		return nil, nil
	}
	var backend einoskill.Backend = multiSkillBackend{backends: backends}
	allow := skillAllowMap(allowlist)
	if len(allow) == 0 {
		allow = skillAllowMap(fe.ListAvailableSkillNames())
	}
	if len(allow) > 0 {
		backend = filteredSkillBackend{base: backend, allow: allow}
	}
	return backend, nil
}

func (uc *AgentUsecase) officialSkillMiddleware(ctx context.Context, allowlist []string) (adk.ChatModelAgentMiddleware, error) {
	backend, err := uc.officialSkillBackend(ctx, allowlist)
	if err != nil || backend == nil {
		return nil, err
	}
	return einoskill.NewMiddleware(ctx, &einoskill.Config{
		Backend:    backend,
		UseChinese: true,
	})
}

func (uc *AgentUsecase) buildOfficialSkillTools(ctx context.Context, allowlist []string) ([]*HarnessTool, string, error) {
	middleware, err := uc.officialSkillMiddleware(ctx, allowlist)
	if err != nil || middleware == nil {
		return nil, "", err
	}
	runCtx := &adk.ChatModelAgentContext{}
	_, runCtx, err = middleware.BeforeAgent(ctx, runCtx)
	if err != nil {
		return nil, "", err
	}
	out := make([]*HarnessTool, 0, len(runCtx.Tools))
	for _, t := range runCtx.Tools {
		ht, err := harnessToolFromEinoTool(ctx, t)
		if err != nil {
			return nil, "", err
		}
		out = append(out, ht)
	}
	return out, strings.TrimSpace(runCtx.Instruction), nil
}

func harnessToolFromEinoTool(ctx context.Context, t tool.BaseTool) (*HarnessTool, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil || strings.TrimSpace(info.Name) == "" {
		return nil, errors.New("tool info is empty")
	}
	invokable, ok := t.(tool.InvokableTool)
	if !ok {
		return nil, fmt.Errorf("tool %s is not invokable", info.Name)
	}
	return &HarnessTool{
		Info: info,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			return invokable.InvokableRun(ctx, rawArgs)
		},
	}, nil
}

func (uc *AgentUsecase) officialSkillInstruction(ctx context.Context, allowlist []string) string {
	_, instruction, err := uc.buildOfficialSkillTools(ctx, allowlist)
	if err != nil {
		return ""
	}
	return instruction
}

var _ filesystem.Backend = localSkillFilesystemBackend{}
var _ einoskill.Backend = multiSkillBackend{}
var _ einoskill.Backend = filteredSkillBackend{}
