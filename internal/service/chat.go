package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"strings"
	"sync"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
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

// contextKey 用于从 context 提取值的 key 类型
type contextKey string

const (
	userIDKey       contextKey = "x-user-id"
	projectPathKey  contextKey = "x-project-path"
)

const (
	userIDHeaderKey      = "x-user-id"
	projectPathHeaderKey = "x-project-path"
	sessionIDHeaderKey   = "x-session-id"
)

// extractUserIDFromContext 从 context 中提取 user_id
func extractUserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(userIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if v := ctx.Value(userIDHeaderKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractProjectPathFromContext 从 context 中提取 project_path
func extractProjectPathFromContext(ctx context.Context) string {
	if v := ctx.Value(projectPathKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if v := ctx.Value(projectPathHeaderKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractSessionIDFromContext(ctx context.Context) string {
	if v := ctx.Value(sessionIDHeaderKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func resolveIdentityWithSessionFallback(userID, projectPath, sessionID string) (resolvedUserID, resolvedProjectPath, resolvedSessionID string) {
	userID = strings.TrimSpace(userID)
	projectPath = strings.TrimSpace(projectPath)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" {
		if sessionID == "" {
			sessionID = "session_" + uuid.NewString()
		}
		userID = sessionID
	}
	if sessionID == "" {
		sessionID = userID
	}
	if projectPath == "" {
		projectPath = "session/" + strings.ReplaceAll(sessionID, "/", "_")
	}
	return userID, projectPath, sessionID
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

func (s *ChatService) harnessRequestFromProto(ctx context.Context, req *v1.ChatStreamReq) biz.HarnessRequest {
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

	// 从 context 中提取 user_id 和 project_path（通常由 auth middleware 设置）
	userID := extractUserIDFromContext(ctx)
	projectPath := extractProjectPathFromContext(ctx)

	return biz.HarnessRequest{
		Model:            req.GetModel(),
		History:          protoHistoryToHarness(req.GetHistory()),
		Input:            req.GetMessage(),
		Attachments:      atts,
		LlmConfigID:      req.GetLlmConfigId(),
		LlmModelEntryID:  req.GetLlmModelEntryId(),
		ManagedAgentID:   req.GetManagedAgentId(),
		ToolOptions:      nil,
		UserID:           userID,
		ProjectPath:      projectPath,
	}
}

// ChatStream gRPC 流式对话。
func (s *ChatService) ChatStream(req *v1.ChatStreamReq, stream v1.Chat_ChatStreamServer) error {
	ctx := stream.Context()
	enrichReqFromMessageImageURLs(ctx, req)
	gen := s.agent.StreamHarness(ctx, s.harnessRequestFromProto(ctx, req))
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

// RegisterChatHTTPRoute 注册聊天相关 HTTP 路由。
func RegisterChatHTTPRoute(s *khttp.Server, chat *ChatService) {
	r := s.Route("/")
	r.POST("/api/v1/chat/stream", chat.chatStreamHTTP)
	r.GET("/api/v1/chat/workspace/file", chat.serveChatWorkspaceFile)
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
	baseCtx := ctx.Request().Context()
	incomingUserID := extractUserIDFromContext(baseCtx)
	incomingProjectPath := extractProjectPathFromContext(baseCtx)
	incomingSessionID := extractSessionIDFromContext(baseCtx)
	if incomingSessionID == "" {
		incomingSessionID = ctx.Request().Header.Get(sessionIDHeaderKey)
	}
	resolvedUserID, resolvedProjectPath, resolvedSessionID := resolveIdentityWithSessionFallback(
		incomingUserID,
		incomingProjectPath,
		incomingSessionID,
	)
	if resolvedUserID != "" {
		w.Header().Set(userIDHeaderKey, resolvedUserID)
	}
	if resolvedProjectPath != "" {
		w.Header().Set(projectPathHeaderKey, resolvedProjectPath)
	}
	if resolvedSessionID != "" {
		w.Header().Set(sessionIDHeaderKey, resolvedSessionID)
	}
	requestCtx := context.WithValue(baseCtx, userIDHeaderKey, resolvedUserID)            //lint:ignore SA1029 "cross-package compatibility"
	requestCtx = context.WithValue(requestCtx, projectPathHeaderKey, resolvedProjectPath) //lint:ignore SA1029 "cross-package compatibility"
	requestCtx = context.WithValue(requestCtx, sessionIDHeaderKey, resolvedSessionID)     //lint:ignore SA1029 "cross-package compatibility"
	flusher, _ := w.(nethttp.Flusher)
	requestCtx, cancel := context.WithCancel(requestCtx)
	defer cancel()
	var writeMu sync.Mutex
	writeSSE := func(payload string) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		if _, err := io.WriteString(w, payload); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}
	if err := writeSSE(": connected\n\n"); err != nil {
		return err
	}

	heartbeatDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := writeSSE(": heartbeat\n\n"); err != nil {
					cancel()
					return
				}
			case <-requestCtx.Done():
				return
			case <-heartbeatDone:
				return
			}
		}
	}()
	defer close(heartbeatDone)

	var streamWriteErr error
	gen := s.agent.StreamHarness(requestCtx, s.harnessRequestFromProto(requestCtx, &req))
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
		if werr := writeSSE("data: " + string(line) + "\n\n"); werr != nil {
			streamWriteErr = werr
			cancel()
			return false
		}
		if err != nil {
			return false
		}
		return true
	})
	if streamWriteErr != nil {
		return streamWriteErr
	}
	if cerr := requestCtx.Err(); cerr != nil && !errors.Is(cerr, context.Canceled) {
		return cerr
	}
	return nil
}

const maxChatFileServeBytes = 4 * 1024 * 1024 // 4 MiB

// serveChatWorkspaceFile 读取并返回工作区中的文件，支持二进制文件（图片/SVG 等）。
// GET /api/v1/chat/workspace/file?path=<absPath>&download=true
func (s *ChatService) serveChatWorkspaceFile(ctx khttp.Context) error {
	filePath := strings.TrimSpace(ctx.Request().URL.Query().Get("path"))
	if filePath == "" {
		ctx.Response().WriteHeader(nethttp.StatusBadRequest)
		_, _ = ctx.Response().Write([]byte(`{"error":"path is required"}`))
		return nil
	}

	if !filepath.IsAbs(filePath) {
		ctx.Response().WriteHeader(nethttp.StatusBadRequest)
		_, _ = ctx.Response().Write([]byte(`{"error":"path must be absolute"}`))
		return nil
	}
	absPath := filepath.Clean(filePath)

	// 验证路径在工作区根目录内，防止路径穿越
	rootAbs, err := s.agent.ResolveAgentWorkspaceRoot()
	if err != nil {
		ctx.Response().WriteHeader(nethttp.StatusInternalServerError)
		_, _ = ctx.Response().Write([]byte(`{"error":"failed to resolve workspace root"}`))
		return nil
	}
	rootAbs = filepath.Clean(rootAbs)
	rel, relErr := filepath.Rel(rootAbs, absPath)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		ctx.Response().WriteHeader(nethttp.StatusForbidden)
		_, _ = ctx.Response().Write([]byte(`{"error":"path is outside workspace root"}`))
		return nil
	}

	st, statErr := os.Stat(absPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			ctx.Response().WriteHeader(nethttp.StatusNotFound)
			_, _ = ctx.Response().Write([]byte(`{"error":"file not found"}`))
			return nil
		}
		ctx.Response().WriteHeader(nethttp.StatusInternalServerError)
		b, _ := json.Marshal(map[string]string{"error": statErr.Error()})
		_, _ = ctx.Response().Write(b)
		return nil
	}
	if st.IsDir() {
		ctx.Response().WriteHeader(nethttp.StatusBadRequest)
		_, _ = ctx.Response().Write([]byte(`{"error":"path is a directory, not a file"}`))
		return nil
	}
	if st.Size() > maxChatFileServeBytes {
		ctx.Response().WriteHeader(nethttp.StatusBadRequest)
		_, _ = ctx.Response().Write([]byte(`{"error":"file too large (exceeds 4 MiB)"}`))
		return nil
	}

	data, readErr := os.ReadFile(absPath)
	if readErr != nil {
		ctx.Response().WriteHeader(nethttp.StatusInternalServerError)
		b, _ := json.Marshal(map[string]string{"error": readErr.Error()})
		_, _ = ctx.Response().Write(b)
		return nil
	}

	contentType := detectFileContentType(absPath, data)
	w := ctx.Response()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "private, max-age=60")

	isDownload := strings.EqualFold(ctx.Request().URL.Query().Get("download"), "true")
	baseName := filepath.Base(absPath)
	if isDownload {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, baseName))
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, baseName))
	}

	_, _ = w.Write(data)
	return nil
}

// detectFileContentType 根据扩展名和内容检测文件的 Content-Type。
// 扩展名优先级高于 http.DetectContentType，因为 Go 内置嗅探对 SVG 等类型判断不准确。
func detectFileContentType(absPath string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(absPath))

	switch ext {
	case ".svg":
		return "image/svg+xml"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".xml":
		return "text/xml; charset=utf-8"
	case ".yaml", ".yml":
		return "text/yaml; charset=utf-8"
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	}

	if len(data) > 0 {
		if detected := nethttp.DetectContentType(data); detected != "application/octet-stream" {
			return detected
		}
	}

	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}

	return "application/octet-stream"
}
