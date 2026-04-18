package service

import (
	"encoding/json"
	"io"
	nethttp "net/http"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/encoding/protojson"
)

// ChatService 管理端 Code 助手对话：走 Harness 全量 SKILL/MCP/workspace 工具。
type ChatService struct {
	v1.UnimplementedChatServer
	agent *biz.AgentUsecase
}

func NewChatService(agent *biz.AgentUsecase) *ChatService {
	return &ChatService{agent: agent}
}

func protoHistoryToHarness(in []*v1.ChatMessage) []biz.HistoryMessage {
	if len(in) == 0 {
		return nil
	}
	out := make([]biz.HistoryMessage, 0, len(in))
	for _, m := range in {
		if m == nil {
			continue
		}
		out = append(out, biz.HistoryMessage{
			Role:    m.GetRole(),
			Content: m.GetContent(),
		})
	}
	return out
}

func (s *ChatService) harnessRequestFromProto(req *v1.ChatStreamReq) biz.HarnessRequest {
	atts := make([]biz.HarnessAttachment, 0, len(req.GetAttachments()))
	for _, a := range req.GetAttachments() {
		if a == nil {
			continue
		}
		atts = append(atts, biz.HarnessAttachment{
			Filename:      a.GetFilename(),
			MimeType:      a.GetMimeType(),
			Text:          a.GetText(),
			ContentBase64: a.GetContentBase64(),
		})
	}
	return biz.HarnessRequest{
		Model:           req.GetModel(),
		History:         protoHistoryToHarness(req.GetHistory()),
		Input:           req.GetMessage(),
		Attachments:     atts,
		LlmConfigID:     req.GetLlmConfigId(),
		LlmModelEntryID: req.GetLlmModelEntryId(),
		ToolOptions:     nil,
	}
}

// ChatStream gRPC 流式对话。
func (s *ChatService) ChatStream(req *v1.ChatStreamReq, stream v1.Chat_ChatStreamServer) error {
	ctx := stream.Context()
	enrichReqFromMessageImageURLs(ctx, req)
	gen := s.agent.StreamHarness(ctx, s.harnessRequestFromProto(req))
	gen(func(sm *biz.StreamMessage, err error) bool {
		if err != nil {
			_ = stream.Send(&v1.ChatStreamReply{
				Done:  true,
				Error: err.Error(),
			})
			return false
		}
		if sm == nil {
			return true
		}
		_ = stream.Send(&v1.ChatStreamReply{
			Content: sm.Content,
			Done:    sm.Done,
		})
		return true
	})
	return nil
}

// RegisterChatHTTPRoute 注册 POST /api/v1/chat/stream（SSE，与 proto 注解一致）。
func RegisterChatHTTPRoute(s *khttp.Server, chat *ChatService) {
	r := s.Route("/")
	r.POST("/api/v1/chat/stream", chat.chatStreamHTTP)
}

func (s *ChatService) chatStreamHTTP(ctx khttp.Context) error {
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return err
	}
	var req v1.ChatStreamReq
	if err := protojson.Unmarshal(body, &req); err != nil {
		return err
	}
	enrichReqFromMessageImageURLs(ctx.Request().Context(), &req)
	w := ctx.Response()
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(nethttp.Flusher)

	gen := s.agent.StreamHarness(ctx, s.harnessRequestFromProto(&req))
	gen(func(sm *biz.StreamMessage, err error) bool {
		reply := &v1.ChatStreamReply{}
		if err != nil {
			reply.Done = true
			reply.Error = err.Error()
		} else if sm != nil {
			reply.Content = sm.Content
			reply.Done = sm.Done
		}
		line, jerr := json.Marshal(reply)
		if jerr != nil {
			reply = &v1.ChatStreamReply{Done: true, Error: jerr.Error()}
			line, _ = json.Marshal(reply)
		}
		_, _ = io.WriteString(w, "data: ")
		_, _ = w.Write(line)
		_, _ = io.WriteString(w, "\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		if err != nil {
			return false
		}
		return true
	})
	return nil
}
