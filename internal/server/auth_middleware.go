package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/google/uuid"
)

// contextKey 用于从 context 提取值的 key 类型
type contextKey string

const (
	userIDKey       contextKey = "x-user-id"
	projectPathKey  contextKey = "x-project-path"
	sessionIDKey    contextKey = "x-session-id"
)

const (
	userIDHeaderKey      = "x-user-id"
	projectPathHeaderKey = "x-project-path"
	sessionIDHeaderKey   = "x-session-id"
)

// setIdentityContextValues 同时写入 typed key 与 string key，便于跨包读取。
func setIdentityContextValues(ctx context.Context, userID, projectPath, sessionID string) context.Context { 
	if userID != "" {
		ctx = context.WithValue(ctx, userIDKey, userID)
		ctx = context.WithValue(ctx, userIDHeaderKey, userID) //lint:ignore SA1029 "cross-package compatibility"
	}
	if projectPath != "" {
		ctx = context.WithValue(ctx, projectPathKey, projectPath)
		ctx = context.WithValue(ctx, projectPathHeaderKey, projectPath) //lint:ignore SA1029 "cross-package compatibility"
	}
	if sessionID != "" {
		ctx = context.WithValue(ctx, sessionIDKey, sessionID)
		ctx = context.WithValue(ctx, sessionIDHeaderKey, sessionID) //lint:ignore SA1029 "cross-package compatibility"
	}
	return ctx
}

// resolveIdentityWithFallback 在无上游 header 透传时提供会话级兜底。
// 规则：
// 1) userID 优先使用 x-user-id；
// 2) userID 缺失时回退到 x-session-id；
// 3) 两者都缺失时生成 session_<uuid>，并作为 userID；
// 4) projectPath 缺失时按 session/<sessionId> 生成，保证 memory 可写入。
func resolveIdentityWithFallback(userID, projectPath, sessionID string) (string, string, string) {
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
		projectPath = "session/" + sanitizeIdentitySegment(sessionID)
	}
	return userID, projectPath, sessionID
}

func sanitizeIdentitySegment(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."
	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if strings.ContainsRune(allowed, r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return "unknown"
	}
	return out
}

// UserIDFromContext 从 context 中提取 user_id
func UserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(userIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ProjectPathFromContext 从 context 中提取 project_path
func ProjectPathFromContext(ctx context.Context) string {
	if v := ctx.Value(projectPathKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// AuthMiddleware 认证中间件，从 header 提取用户身份并注入 context。
// 在无网关的单体服务中，客户端需在请求时传递身份信息。
func AuthMiddleware() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, in interface{}) (interface{}, error) {
			if r, ok := in.(interface{ Header() http.Header }); ok {
				headers := r.Header()
				userID, projectPath, sessionID := resolveIdentityWithFallback(
					headers.Get(userIDHeaderKey),
					headers.Get(projectPathHeaderKey),
					headers.Get(sessionIDHeaderKey),
				)
				ctx = setIdentityContextValues(ctx, userID, projectPath, sessionID)
			}
			return next(ctx, in)
		}
	}
}

// AuthMiddlewareWithToken 从 Authorization header 解析用户身份（无网关场景）。
// 支持两种模式：
//   - "Bearer <token>": 从 token 解析 user_id（需自行实现 token 解析逻辑）
//   - 直接传递 user_id 到 x-user-id header
func AuthMiddlewareWithToken(tokenParser func(token string) (userID, projectPath string)) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, in interface{}) (interface{}, error) {
			if r, ok := in.(interface{ Header() http.Header }); ok {
				headers := r.Header()
				if userID := headers.Get("x-user-id"); userID != "" {
					userID, projectPath, sessionID := resolveIdentityWithFallback(
						userID,
						headers.Get(projectPathHeaderKey),
						headers.Get(sessionIDHeaderKey),
					)
					ctx = setIdentityContextValues(ctx, userID, projectPath, sessionID)
				} else if auth := headers.Get("Authorization"); auth != "" {
					if strings.HasPrefix(auth, "Bearer ") {
						token := strings.TrimPrefix(auth, "Bearer ")
						if userID, projectPath := tokenParser(token); userID != "" {
							userID, projectPath, sessionID := resolveIdentityWithFallback(
								userID,
								projectPath,
								headers.Get(sessionIDHeaderKey),
							)
							ctx = setIdentityContextValues(ctx, userID, projectPath, sessionID)
						}
					}
				} else {
					userID, projectPath, sessionID := resolveIdentityWithFallback("", "", headers.Get(sessionIDHeaderKey))
					ctx = setIdentityContextValues(ctx, userID, projectPath, sessionID)
				}
			}
			return next(ctx, in)
		}
	}
}