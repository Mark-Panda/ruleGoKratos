package server

import (
	"encoding/json"
	nethttp "net/http"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, rules *service.RuleGoService, runLogs *service.RunLogService, components *service.ComponentService, admin *service.AdminService, chat *service.ChatService, playground *service.PlaygroundService, taskService *service.TaskBoardService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
		http.Filter(corsFilter),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterRuleGoHTTPServer(srv, rules)
	v1.RegisterRunLogHTTPServer(srv, runLogs)
	v1.RegisterComponentHTTPServer(srv, components)
	v1.RegisterAdminHTTPServer(srv, admin)
	registerAdminExtraRoutes(srv, admin)
	service.RegisterChatHTTPRoute(srv, chat)
	service.RegisterPlaygroundHTTPRoutes(srv, playground)
	v1.RegisterTaskBoardServiceHTTPServer(srv, taskService)
	v1.RegisterServiceManagementServiceHTTPServer(srv, taskService)
	RegisterTerminalWebSocket(srv, admin, logger)

	return srv
}

// registerAdminExtraRoutes 注册不走 proto 生成的管理后台补充接口。
func registerAdminExtraRoutes(s *http.Server, admin *service.AdminService) {
	r := s.Route("/api/v1/admin")
	// GET /api/v1/admin/skills/list?scope=system|workflow
	r.GET("/skills/list", func(ctx http.Context) error {
		scope := ctx.Request().URL.Query().Get("scope")
		reply, err := admin.ListSkillsByScope(ctx, scope)
		if err != nil {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			b, _ := json.Marshal(map[string]string{"error": err.Error()})
			_, _ = ctx.Response().Write(b)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(reply)
		_, _ = ctx.Response().Write(b)
		return nil
	})

	// GET /api/v1/admin/skills/file?path=<relative> 读取技能文件内容
	r.GET("/skills/file", func(ctx http.Context) error {
		req := ctx.Request()
		path := req.URL.Query().Get("path")
		scope := req.URL.Query().Get("scope")
		if path == "" {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			_, _ = ctx.Response().Write([]byte(`{"error":"path is required"}`))
			return nil
		}
		content, err := admin.ReadSkillFileContentByScope(scope, path)
		if err != nil {
			ctx.Response().WriteHeader(nethttp.StatusNotFound)
			b, _ := json.Marshal(map[string]string{"error": err.Error()})
			_, _ = ctx.Response().Write(b)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]string{"content": content, "path": path})
		_, _ = ctx.Response().Write(b)
		return nil
	})
	// PUT /api/v1/admin/skills/file 写入技能文件内容
	r.PUT("/skills/file", func(ctx http.Context) error {
		var payload struct {
			Scope   string `json:"scope"`
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(ctx.Request().Body).Decode(&payload); err != nil {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			_, _ = ctx.Response().Write([]byte(`{"error":"invalid request body"}`))
			return nil
		}
		if payload.Path == "" {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			_, _ = ctx.Response().Write([]byte(`{"error":"path is required"}`))
			return nil
		}
		if err := admin.WriteSkillFileContentByScope(payload.Scope, payload.Path, payload.Content); err != nil {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			b, _ := json.Marshal(map[string]string{"error": err.Error()})
			_, _ = ctx.Response().Write(b)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]string{"path": payload.Path})
		_, _ = ctx.Response().Write(b)
		return nil
	})
	// POST /api/v1/admin/skills/upload/file?scope=system|workflow 上传技能 zip（自定义 scope）
	r.POST("/skills/upload/file", func(ctx http.Context) error {
		scope := ctx.Request().URL.Query().Get("scope")
		var req v1.UploadSkillRequest
		if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			_, _ = ctx.Response().Write([]byte(`{"error":"invalid request body"}`))
			return nil
		}
		reply, err := admin.UploadSkillByScope(ctx, &req, scope)
		if err != nil {
			ctx.Response().WriteHeader(nethttp.StatusBadRequest)
			b, _ := json.Marshal(map[string]string{"error": err.Error()})
			_, _ = ctx.Response().Write(b)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(reply)
		_, _ = ctx.Response().Write(b)
		return nil
	})
}

func corsFilter(h nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Access-Token, X-Requested-With")
		if r.Method == "OPTIONS" {
			w.WriteHeader(nethttp.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
