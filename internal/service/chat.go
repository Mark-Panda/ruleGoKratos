package service

import (
	"encoding/json"
	"net/http"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/kratos/v2/log"
)

// ChatService 聊天服务
type ChatService struct {
	v1.UnimplementedChatServer
	agentUC *biz.AgentUsecase
	log     *log.Helper
}

// NewChatService 创建聊天服务
func NewChatService(agentUC *biz.AgentUsecase, logger log.Logger) *ChatService {
	return &ChatService{
		agentUC: agentUC,
		log:     log.NewHelper(logger),
	}
}

// ChatStream 实现gRPC流式对话（用于gRPC调用）
func (s *ChatService) ChatStream(req *v1.ChatStreamReq, stream v1.Chat_ChatStreamServer) error {
	ctx := stream.Context()

	// 构建消息历史
	history := make([]*blades.Message, 0)
	for _, msg := range req.History {
		if msg.Role == "user" {
			history = append(history, blades.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			history = append(history, blades.AssistantMessage(msg.Content))
		}
	}

	// 使用Agent进行流式对话
	generator := s.agentUC.ChatStream(ctx, req.Model, history, req.Message)

	// 遍历生成器并发送响应
	for msg, err := range generator {
		if err != nil {
			return stream.Send(&v1.ChatStreamReply{
				Error: err.Error(),
				Done:  true,
			})
		}
		if msg != nil {
            // 提取消息文本内容（空文本回退到字符串化表示）
            content := msg.Text()
            if content == "" {
                content = msg.String()
            }
			if content != "" {
				if sendErr := stream.Send(&v1.ChatStreamReply{
					Content: content,
					Done:    msg.Status == blades.StatusCompleted,
				}); sendErr != nil {
					s.log.Errorf("发送流式响应失败: %v", sendErr)
					return sendErr
				}
			}
			// 如果消息已完成，发送完成标志
			if msg.Status == blades.StatusCompleted {
				return stream.Send(&v1.ChatStreamReply{
					Done: true,
				})
			}
		}
	}

	// 流结束
	return stream.Send(&v1.ChatStreamReply{
		Done: true,
	})
}

// ChatStreamHTTP 实现HTTP SSE流式对话
func (s *ChatService) ChatStreamHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 立即写入状态码，防止Kratos框架在handler返回后尝试写入响应
	w.WriteHeader(http.StatusOK)

	// 确保有Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.log.Error("ResponseWriter不支持Flush")
		s.sendSSEError(w, "服务器不支持流式响应")
		return
	}

	// 解析请求
	var req v1.ChatStreamReq
	if r.Body == nil {
		s.log.Error("请求体为空")
		s.sendSSEError(w, "请求体不能为空")
		flusher.Flush()
		return
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Errorf("解析请求失败: %v", err)
		s.sendSSEError(w, err.Error())
		flusher.Flush()
		return
	}

	ctx := r.Context()

	// 构建消息历史
	history := make([]*blades.Message, 0)
	for _, msg := range req.History {
		if msg.Role == "user" {
			history = append(history, blades.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			history = append(history, blades.AssistantMessage(msg.Content))
		}
	}

	// 使用Agent进行流式对话
	generator := s.agentUC.CreateRuleChainPlannerAgent(ctx, req.Message)
	// generator := s.agentUC.ChatStream(ctx, req.Model, history, req.Message)

	// 遍历生成器并发送响应
	for msg, err := range generator {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			s.log.Info("客户端断开连接")
			return
		default:
		}

		if err != nil {
			s.log.Errorf("流式对话错误: %v", err)
			s.sendSSEError(w, err.Error())
			flusher.Flush()
			return
		}

		if msg != nil {
            // 提取消息文本内容（空文本回退到字符串化表示）
            content := msg.Text()
            if content == "" {
                content = msg.String()
            }
			if content != "" {
				reply := &v1.ChatStreamReply{
					Content: content,
					Done:    msg.Status == blades.StatusCompleted,
				}
				if sendErr := s.sendSSEReply(w, reply); sendErr != nil {
					s.log.Errorf("发送SSE响应失败: %v", sendErr)
					return
				}
				flusher.Flush()
			}
			// 如果消息已完成，发送完成标志
			if msg.Status == blades.StatusCompleted {
				reply := &v1.ChatStreamReply{
					Done: true,
				}
				s.sendSSEReply(w, reply)
				flusher.Flush()
				return
			}
		}
	}

	// 流结束
	reply := &v1.ChatStreamReply{
		Done: true,
	}
	s.sendSSEReply(w, reply)
	flusher.Flush()
}

// sendSSEReply 发送SSE格式的响应
func (s *ChatService) sendSSEReply(w http.ResponseWriter, reply *v1.ChatStreamReply) error {
	data, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("data: " + string(data) + "\n\n"))
	return err
}

// sendSSEError 发送SSE格式的错误
func (s *ChatService) sendSSEError(w http.ResponseWriter, errorMsg string) {
	reply := &v1.ChatStreamReply{
		Error: errorMsg,
		Done:  true,
	}
	s.sendSSEReply(w, reply)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
