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

// BuildReadWorkspaceFileTool 读取 workspace 内 UTF-8 文本文件（二进制会按错误提示）。
func (uc *AgentUsecase) BuildReadWorkspaceFileTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "read_workspace_file",
		Desc: "读取 Agent workspace 根目录下的文件内容（文本）。path 为相对根目录的路径，禁止含 .. 越权。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "相对 workspace 的文件路径，例如 notes/a.txt",
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
			root, err := uc.effectiveWorkspaceRoot(ctx)
			if err != nil {
				return "", err
			}
			full, err := absPathUnderWorkspace(root, a.Path)
			if err != nil {
				return "", err
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
		Desc: "在 Agent workspace 内创建或覆盖文件。path 相对根目录；content 为完整文件文本。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "相对 workspace 的文件路径",
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
			full, err := absPathUnderWorkspace(root, a.Path)
			if err != nil {
				return "", err
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

// BuildRunWorkspaceShellTool 在 workspace 下用 bash -lc 执行命令（工作目录可指定相对路径）。
func (uc *AgentUsecase) BuildRunWorkspaceShellTool() (*HarnessTool, error) {
	toolInfo := &schema.ToolInfo{
		Name: "run_workspace_shell",
		Desc: "在 workspace 沙箱内执行 shell：使用 bash -lc。working_directory 可选、相对 workspace；勿执行破坏性操作。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "交给 bash -lc 的一条或多条命令",
				Required: true,
			},
			"working_directory": {
				Type:     schema.String,
				Desc:     "相对 workspace 的工作目录，默认可留空表示 workspace 根",
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
				cwd, err = absPathUnderWorkspace(root, wd)
				if err != nil {
					return "", err
				}
				if fi, err := os.Stat(cwd); err != nil || !fi.IsDir() {
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
