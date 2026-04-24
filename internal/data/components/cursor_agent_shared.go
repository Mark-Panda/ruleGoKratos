package data

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// 与 flowgram CursorCliSection（agent status）一致：排除 Not logged in；接受 Login successful、Logged in as、或其它已登录提示。
var (
	cursorAgentLoggedInAsRE = regexp.MustCompile(`(?i)logged in as\b`)
	cursorAgentLoggedInRE   = regexp.MustCompile(`(?i)\blogged in\b`)
)

func cursorAgentStatusLooksAuthed(combined string) bool {
	t := strings.TrimSpace(combined)
	if t == "" {
		return false
	}
	lower := strings.ToLower(t)
	if strings.Contains(lower, "not logged in") {
		return false
	}
	if strings.Contains(lower, "login successful") {
		return true
	}
	return cursorAgentLoggedInAsRE.MatchString(t) || cursorAgentLoggedInRE.MatchString(t)
}

func cursorAgentTruncateForErr(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

// cursorAgentWrapExecErr 在进程退出错误上附带 stdout/stderr 摘录，便于排查未登录、CLI 报错等。
func cursorAgentWrapExecErr(component string, err error, stdout, stderr string) error {
	if err == nil {
		return nil
	}
	out := strings.TrimSpace(stdout)
	errOut := strings.TrimSpace(stderr)
	if out == "" && errOut == "" {
		return fmt.Errorf("%s: %w", component, err)
	}
	return fmt.Errorf("%s: %w | stdout=%q | stderr=%q", component, err,
		cursorAgentTruncateForErr(out, 4000), cursorAgentTruncateForErr(errOut, 4000))
}

// runAgentStatusCheck 使用与主命令相同的 bin、argv 前缀（--workspace/--worktree）及 cwd，用于预检登录。
func runAgentStatusCheck(parentCtx context.Context, bin string, argv []string, workDir string) (stdout, stderr string, err error) {
	statusCtx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(statusCtx, bin, argv...)
	cmd.Dir = workDir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return outBuf.String(), errBuf.String(), err
	}
	return outBuf.String(), errBuf.String(), nil
}
