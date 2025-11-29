package biz

import (
	"context"
	"log"
	"time"

	"github.com/go-kratos/blades"
)

type AgentLogger struct {
	next blades.Handler
}

// NewLogging creates a new Logging middleware.
func NewLogging(next blades.Handler) blades.Handler {
	return &AgentLogger{next}
}

func (m *AgentLogger) onError(start time.Time, agent blades.AgentContext, invocation *blades.Invocation, err error) {
	log.Printf("失败日志: 模型名称(%s) prompt(%s) 失败之后耗时 %s: 错误信息为%v", agent.Name(), invocation.Message.String(), time.Since(start), err)
}

func (m *AgentLogger) onSuccess(start time.Time, agent blades.AgentContext, invocation *blades.Invocation, output *blades.Message) {
	log.Printf("成功日志: 模型名称(%s) prompt(%s) 成功之后耗时 %s: 输出内容为%s", agent.Name(), invocation.Message.String(), time.Since(start), output.String())
}

func (m *AgentLogger) Handle(ctx context.Context, invocation *blades.Invocation) blades.Generator[*blades.Message, error] {
	return func(yield func(*blades.Message, error) bool) {
		start := time.Now()
		agent, ok := blades.FromAgentContext(ctx)
		if !ok {
			yield(nil, blades.ErrNoAgentContext)
			return
		}
		streaming := m.next.Handle(ctx, invocation)
		for msg, err := range streaming {
			if err != nil {
				m.onError(start, agent, invocation, err)
			} else {
				m.onSuccess(start, agent, invocation, msg)
			}
			if !yield(msg, err) {
				break
			}
		}
		// 打印完整的成功日志
		// log.Printf("完整成功日志: 模型名称(%s) prompt(%s) 成功之后耗时 %s: 输出内容为%s", agent.Name(), invocation.Message.String(), time.Since(start), output.String())
	}
}
