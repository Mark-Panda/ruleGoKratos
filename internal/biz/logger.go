package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

const harnessLogMaxChars = 24000

// TruncateHarnessLog 防止单条日志过大；过长时保留前后缀便于排查。
func TruncateHarnessLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	head := max * 3 / 4
	tail := max - head
	if tail < 64 {
		tail = 64
		head = max - tail
	}
	return s[:head] + fmt.Sprintf("\n…(truncated total=%d chars)…\n", len(s)) + s[len(s)-tail:]
}

type HarnessLogger struct {
	log *log.Helper
}

func NewHarnessLogger(helper *log.Helper) *HarnessLogger {
	return &HarnessLogger{log: helper}
}

func (l *HarnessLogger) LogRunStart(requestID, model string, historySize int, input string) {
	in := TruncateHarnessLog(input, harnessLogMaxChars)
	l.log.Infof("harness run start request_id=%s model=%s history_size=%d input_len=%d input=\n%s", requestID, model, historySize, len(input), in)
}

func (l *HarnessLogger) LogRunFinish(requestID string, cost time.Duration) {
	l.log.Infof("harness run finish request_id=%s cost_ms=%d", requestID, cost.Milliseconds())
}

// LogHarnessOutput 单次 Harness 成功结束时的助手产出（与 Stream 拼接结果一致）。
func (l *HarnessLogger) LogHarnessOutput(requestID string, output string) {
	out := TruncateHarnessLog(output, harnessLogMaxChars)
	l.log.Infof("harness output request_id=%s output_len=%d output=\n%s", requestID, len(output), out)
}

func (l *HarnessLogger) LogModelRound(requestID string, round int, cost time.Duration, err error) {
	if err != nil {
		l.log.Errorf("harness model round failed request_id=%s round=%d cost_ms=%d err=%v", requestID, round, cost.Milliseconds(), err)
		return
	}
	l.log.Infof("harness model round request_id=%s round=%d cost_ms=%d", requestID, round, cost.Milliseconds())
}

func (l *HarnessLogger) LogToolCall(requestID, toolName string, cost time.Duration, err error) {
	if err != nil {
		l.log.Errorf("harness tool call failed request_id=%s tool=%s cost_ms=%d err=%v", requestID, toolName, cost.Milliseconds(), err)
		return
	}
	l.log.Infof("harness tool call request_id=%s tool=%s cost_ms=%d", requestID, toolName, cost.Milliseconds())
}

// LogToolCallIO 工具调用的入参/出参（已截断），便于与模型流式输出区分。
func (l *HarnessLogger) LogToolCallIO(requestID, toolName string, cost time.Duration, args, result string, err error) {
	a := TruncateHarnessLog(args, 8000)
	if err != nil {
		l.log.Errorf("harness tool io request_id=%s tool=%s cost_ms=%d args=\n%s err=%v", requestID, toolName, cost.Milliseconds(), a, err)
		return
	}
	r := TruncateHarnessLog(result, 8000)
	l.log.Infof("harness tool io request_id=%s tool=%s cost_ms=%d args=\n%s result=\n%s", requestID, toolName, cost.Milliseconds(), a, r)
}

func (l *HarnessLogger) LogError(requestID, stage string, err error) {
	l.log.Errorf("harness error request_id=%s stage=%s err=%v", requestID, stage, err)
}

func (l *HarnessLogger) LogSandboxDecision(requestID, toolName string, allowed bool, reason string) {
	l.log.Infof("harness sandbox request_id=%s tool=%s allowed=%t reason=%s", requestID, toolName, allowed, reason)
}

func formatErrorCode(code, detail string) error {
	return fmt.Errorf("%s: %s", code, detail)
}
