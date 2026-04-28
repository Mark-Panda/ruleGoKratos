package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
)

// contextKey 用于从 context 提取值的 key 类型
type contextKey string

const (
	userIDKey       contextKey = "x-user-id"
	projectPathKey  contextKey = "x-project-path"
)

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
				if userID := headers.Get("x-user-id"); userID != "" {
					ctx = context.WithValue(ctx, userIDKey, userID)
				}
				if projectPath := headers.Get("x-project-path"); projectPath != "" {
					ctx = context.WithValue(ctx, projectPathKey, projectPath)
				}
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
					ctx = context.WithValue(ctx, userIDKey, userID)
					if pp := headers.Get("x-project-path"); pp != "" {
						ctx = context.WithValue(ctx, projectPathKey, pp)
					}
				} else if auth := headers.Get("Authorization"); auth != "" {
					if strings.HasPrefix(auth, "Bearer ") {
						token := strings.TrimPrefix(auth, "Bearer ")
						if userID, projectPath := tokenParser(token); userID != "" {
							ctx = context.WithValue(ctx, userIDKey, userID)
							if projectPath != "" {
								ctx = context.WithValue(ctx, projectPathKey, projectPath)
							}
						}
					}
				}
			}
			return next(ctx, in)
		}
	}
}