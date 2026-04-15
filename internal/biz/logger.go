package biz

import (
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type HarnessLogger struct {
	log *log.Helper
}

func NewHarnessLogger(helper *log.Helper) *HarnessLogger {
	return &HarnessLogger{log: helper}
}

func (l *HarnessLogger) LogRunStart(requestID, model string, historySize int, input string) {
	l.log.Infof("harness run start request_id=%s model=%s history_size=%d input_len=%d", requestID, model, historySize, len(input))
}

func (l *HarnessLogger) LogRunFinish(requestID string, cost time.Duration) {
	l.log.Infof("harness run finish request_id=%s cost_ms=%d", requestID, cost.Milliseconds())
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

func (l *HarnessLogger) LogError(requestID, stage string, err error) {
	l.log.Errorf("harness error request_id=%s stage=%s err=%v", requestID, stage, err)
}

func (l *HarnessLogger) LogSandboxDecision(requestID, toolName string, allowed bool, reason string) {
	l.log.Infof("harness sandbox request_id=%s tool=%s allowed=%t reason=%s", requestID, toolName, allowed, reason)
}

func formatErrorCode(code, detail string) error {
	return fmt.Errorf("%s: %s", code, detail)
}
