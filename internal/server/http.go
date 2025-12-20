package server

import (
	nethttp "net/http"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, rules *service.RuleGoService, runLogs *service.RunLogService, components *service.ComponentService, md *service.MdWorkflowService, chat *service.ChatService, logger log.Logger) *http.Server {
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
	v1.RegisterMdWorkflowHTTPServer(srv, md)
	v1.RegisterChatHTTPServer(srv, chat)

	// 注册Chat服务的HTTP SSE路由
	// 注意：流式响应需要直接使用底层ResponseWriter，绕过Kratos的响应处理
	srv.Route("/").POST("/api/v1/chat/stream", func(ctx http.Context) error {
		// 直接调用流式处理函数，它已经处理了所有响应写入
		// 返回nil表示响应已经由ChatStreamHTTP处理完成
		chat.ChatStreamHTTP(ctx.Response(), ctx.Request())
		// 返回nil，告诉Kratos响应已经处理，不要尝试再次写入
		return nil
	})

	return srv
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
