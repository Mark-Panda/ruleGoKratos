package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const (
	maxWorkspaceFileReadBytes = 4 << 20 // 4MiB
	maxShellCombinedOutput    = 256 << 10
)

// harnessWorkspaceRootKey 本轮 Harness 内文件/shell 工具的有效 workspace 根（绝对路径）。
type harnessWorkspaceRootKey struct{}

func withHarnessWorkspaceRoot(ctx context.Context, absRoot string) context.Context {
	return context.WithValue(ctx, harnessWorkspaceRootKey{}, filepath.Clean(absRoot))
}

func (uc *AgentUsecase) effectiveWorkspaceRoot(ctx context.Context) (string, error) {
	if v := ctx.Value(harnessWorkspaceRootKey{}); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return filepath.Clean(s), nil
		}
	}
	return uc.resolveAgentWorkspaceRoot()
}

// sanitizePlaygroundWorkspaceSessionDir 校验相对配置的子路径：禁止绝对路径、..、空段。
func sanitizePlaygroundWorkspaceSessionDir(sub string) string {
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return ""
	}
	if filepath.IsAbs(sub) {
		return ""
	}
	sub = filepath.Clean(sub)
	sub = strings.TrimPrefix(sub, string(filepath.Separator))
	if sub == "" || sub == "." {
		return ""
	}
	for _, seg := range strings.Split(sub, string(filepath.Separator)) {
		if seg == "" || seg == "." || seg == ".." {
			return ""
		}
	}
	return sub
}

func (uc *AgentUsecase) resolveAgentWorkspaceRoot() (string, error) {
	var raw string
	if uc.config != nil && uc.config.Agent != nil {
		if s := strings.TrimSpace(uc.config.Agent.GetWorkspaceRoot()); s != "" {
			raw = s
		} else if uc.config.Agent.Skill != nil {
			raw = strings.TrimSpace(uc.config.Agent.Skill.GetDir())
		}
	}
	if raw == "" {
		raw = "."
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("解析 workspace 根路径: %w", err)
	}
	return abs, nil
}

func absPathUnderWorkspace(rootAbs, userPath string) (string, error) {
	userPath = strings.TrimSpace(userPath)
	if userPath == "" {
		return "", errors.New("path 不能为空")
	}
	p := userPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(rootAbs, p)
	}
	p = filepath.Clean(p)
	rootAbs = filepath.Clean(rootAbs)
	rel, err := filepath.Rel(rootAbs, p)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("路径必须在 workspace 根目录内")
	}
	return p, nil
}

func absPathUnderAnyRoot(roots []string, userPath string) (string, error) {
	p := filepath.Clean(strings.TrimSpace(userPath))
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" {
			continue
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return p, nil
	}
	return "", errors.New("路径必须在允许写入的根目录内")
}

func (uc *AgentUsecase) writableAbsoluteRoots(ctx context.Context) []string {
	roots := make([]string, 0, 8)
	appendIfNeeded := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if !filepath.IsAbs(raw) {
			return
		}
		clean := filepath.Clean(raw)
		for _, exist := range roots {
			if exist == clean {
				return
			}
		}
		roots = append(roots, clean)
	}

	if root, err := uc.effectiveWorkspaceRoot(ctx); err == nil {
		appendIfNeeded(root)
	}
	if uc.config != nil && uc.config.Agent != nil {
		appendIfNeeded(uc.config.Agent.GetWorkspaceRoot())
	}
	agentSkillRoot := strings.TrimSpace(os.Getenv("AGENT_SKILL_DIR"))
	if agentSkillRoot == "" {
		agentSkillRoot = "/agent/skills"
	}
	appendIfNeeded(agentSkillRoot)
	return roots
}

// resolveReadablePath 解析文件/目录路径：
//   - 相对路径 → 必须在 workspace 根目录内（禁止 ..）
//   - 绝对路径 → 文件/目录本身存在即可（skill 目录、WORK_DIR、任意已存在路径）
func (uc *AgentUsecase) resolveReadablePath(ctx context.Context, userPath string) (string, error) {
	userPath = strings.TrimSpace(userPath)
	if filepath.IsAbs(userPath) {
		clean := filepath.Clean(userPath)
		return clean, nil
	}
	root, err := uc.effectiveWorkspaceRoot(ctx)
	if err != nil {
		return "", err
	}
	return absPathUnderWorkspace(root, userPath)
}

// BuildReadWorkspaceFileTool 读取 workspace 内或任意已存在绝对路径的 UTF-8 文本文件。
func (uc *AgentUsecase) BuildReadWorkspaceFileTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "read_workspace_file",
		Desc: "读取文件内容（文本）。path 可为：相对 workspace 的路径（禁止 ..）；或任意已存在的绝对路径（如 /app/skills/...、/root/bizCompareWarehouse/... 等）。注意：路径须含完整文件名及扩展名（如 SKILL.md，不可省略 .md）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "相对 workspace 的路径（如 notes/a.txt），或任意绝对路径（如 /root/bizCompareWarehouse/repo/file.js）",
				Required: true,
			},
		}),
	}
	type args struct {
		Path string `json:"path"`
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var a args
			if err := json.Unmarshal([]byte(rawArgs), &a); err != nil {
				return "", err
			}
			full, err := uc.resolveReadablePath(ctx, a.Path)
			if err != nil {
				return "", err
			}
			// 若文件不存在且路径无扩展名，自动尝试追加常见文档扩展名（SKILL → SKILL.md 等）
			if _, statErr := os.Stat(full); os.IsNotExist(statErr) && filepath.Ext(full) == "" {
				for _, ext := range []string{".md", ".txt"} {
					candidate := full + ext
					if _, candErr := os.Stat(candidate); candErr == nil {
						full = candidate
						break
					}
				}
			}
			st, err := os.Stat(full)
			if err != nil {
				return "", err
			}
			if st.IsDir() {
				return "", errors.New("path 是目录，不是文件")
			}
			if st.Size() > maxWorkspaceFileReadBytes {
				return "", fmt.Errorf("文件过大（>%d 字节），拒绝读取", maxWorkspaceFileReadBytes)
			}
			b, err := os.ReadFile(full)
			if err != nil {
				return "", err
			}
			if !isMostlyText(b) {
				return "", errors.New("文件疑似二进制，仅支持以文本方式读取")
			}
			return string(b), nil
		},
	}, nil
}

func isMostlyText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	n := len(b)
	if n > 8192 {
		n = 8192
	}
	ctl := 0
	for i := 0; i < n; i++ {
		c := b[i]
		if c == 0 || (c < 0x09 && c != 0x09 && c != 0x0a && c != 0x0d) {
			ctl++
		}
	}
	return ctl*20 < n
}

// BuildWriteWorkspaceFileTool 在 workspace 内创建或覆盖文件（含自动创建父目录）。
func (uc *AgentUsecase) BuildWriteWorkspaceFileTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "write_workspace_file",
		Desc: "创建或覆盖文本文件。path 可为：相对 workspace 根目录的路径；或允许写入根目录下的绝对路径（如 /agent/skills/...）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "相对 workspace 的文件路径，或允许写入根目录下的绝对路径",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "写入的完整文件内容（UTF-8）",
				Required: true,
			},
		}),
	}
	type writeFileArgs struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var a writeFileArgs
			if err := json.Unmarshal([]byte(rawArgs), &a); err != nil {
				return "", err
			}
			if len(a.Content) > maxWorkspaceFileReadBytes {
				return "", fmt.Errorf("content 过长（>%d 字节）", maxWorkspaceFileReadBytes)
			}
			root, err := uc.effectiveWorkspaceRoot(ctx)
			if err != nil {
				return "", err
			}
			var full string
			if filepath.IsAbs(strings.TrimSpace(a.Path)) {
				full, err = absPathUnderAnyRoot(uc.writableAbsoluteRoots(ctx), a.Path)
				if err != nil {
					return "", err
				}
			} else {
				full, err = absPathUnderWorkspace(root, a.Path)
				if err != nil {
					return "", err
				}
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(full, []byte(a.Content), 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("已写入 %d 字节 -> %s", len(a.Content), full), nil
		},
	}, nil
}

// BuildRunWorkspaceShellTool 在 workspace 下用 bash -lc 执行命令（工作目录可指定相对路径或 skill 目录绝对路径）。
func (uc *AgentUsecase) BuildRunWorkspaceShellTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "run_workspace_shell",
		Desc: "在沙箱内执行 shell：使用 bash -lc。working_directory 可选：留空表示 workspace 根；相对路径须在 workspace 内；绝对路径须是已存在的目录（如 /app/skills/...、/root/bizCompareWarehouse 等）。勿执行破坏性操作。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "交给 bash -lc 的一条或多条命令",
				Required: true,
			},
			"working_directory": {
				Type:     schema.String,
				Desc:     "工作目录：留空=workspace 根；相对路径=workspace 内；绝对路径=任意已存在目录（skill 目录 /app/skills/... 或 WORK_DIR 等）",
				Required: false,
			},
		}),
	}
	type shellArgs struct {
		Command          string `json:"command"`
		WorkingDirectory string `json:"working_directory"`
	}
	return &HarnessTool{
		Info: toolInfo,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var a shellArgs
			if err := json.Unmarshal([]byte(rawArgs), &a); err != nil {
				return "", err
			}
			cmdStr := strings.TrimSpace(a.Command)
			if cmdStr == "" {
				return "", errors.New("command 不能为空")
			}
			root, err := uc.effectiveWorkspaceRoot(ctx)
			if err != nil {
				return "", err
			}
			cwd := root
			if wd := strings.TrimSpace(a.WorkingDirectory); wd != "" {
				if filepath.IsAbs(wd) {
					// 绝对路径 working_directory：只要目录存在即允许。
					// run_workspace_shell 的命令本身已可访问任意路径（bash 权限），
					// 限制 CWD 参数不能实际提升安全性，反而阻碍 skill 在 WORK_DIR 等目录下执行。
					cwd = filepath.Clean(wd)
				} else {
					cwd, err = absPathUnderWorkspace(root, wd)
					if err != nil {
						return "", err
					}
				}
				if fi, statErr := os.Stat(cwd); statErr != nil || !fi.IsDir() {
					return "", fmt.Errorf("working_directory 不是有效目录: %s", wd)
				}
			}
			cmd := exec.CommandContext(ctx, "bash", "-lc", cmdStr)
			cmd.Dir = cwd
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			runErr := cmd.Run()
			exitCode := -1
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			}
			out := trimOutput(stdout.Bytes(), maxShellCombinedOutput/2)
			errOut := trimOutput(stderr.Bytes(), maxShellCombinedOutput/2)
			msg := fmt.Sprintf("exit_code=%d\n--- stdout ---\n%s\n--- stderr ---\n%s", exitCode, out, errOut)
			if runErr != nil {
				var exitErr *exec.ExitError
				if errors.As(runErr, &exitErr) {
					return msg, nil
				}
				return "", fmt.Errorf("执行 shell 失败: %w\n%s", runErr, msg)
			}
			return msg, nil
		},
	}, nil
}

func trimOutput(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + fmt.Sprintf("\n…(截断，仅保留前 %d 字节)", max)
}
