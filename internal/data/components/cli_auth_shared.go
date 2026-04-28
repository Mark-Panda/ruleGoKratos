package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

type feishuCliAuthStatus struct {
	TokenStatus string `json:"tokenStatus"`
	OK          *bool  `json:"ok"`
	UserName    string `json:"userName"`
	Identity    string `json:"identity"`
	GrantedAt   string `json:"grantedAt"`
	AppID       string `json:"appId"`
	Brand       string `json:"brand"`
}

func runCommandWithTimeout(bin string, args []string, workDir string, timeoutMs int) (string, string, error) {
	if timeoutMs <= 0 {
		timeoutMs = 15000
	}
	runCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Dir = workDir
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return stdoutBuf.String(), stderrBuf.String(), err
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
}

func parseFeishuCliAuthStatus(raw string) (*feishuCliAuthStatus, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, errors.New("empty status output")
	}
	var out feishuCliAuthStatus
	if err := json.Unmarshal([]byte(text), &out); err == nil {
		return &out, nil
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, errors.New("status output does not contain json object")
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func feishuCliStatusLooksAuthed(stdout, stderr string) (bool, *feishuCliAuthStatus, error) {
	candidates := []string{
		stdout,
		stderr,
		strings.TrimSpace(strings.TrimSpace(stdout) + "\n" + strings.TrimSpace(stderr)),
	}
	var lastErr error
	for _, candidate := range candidates {
		parsed, err := parseFeishuCliAuthStatus(candidate)
		if err != nil {
			lastErr = err
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parsed.TokenStatus), "valid") {
			return true, parsed, nil
		}
		if parsed.OK != nil {
			return *parsed.OK, parsed, nil
		}
		return false, parsed, nil
	}
	if lastErr == nil {
		lastErr = errors.New("unable to parse status output")
	}
	return false, nil, lastErr
}
