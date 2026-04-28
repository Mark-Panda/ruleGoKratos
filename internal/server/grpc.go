package server

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc/metadata"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, rules *service.RuleGoService, runLogs *service.RunLogService, components *service.ComponentService, admin *service.AdminService, chat *service.ChatService, taskService *service.TaskBoardService, scheduledTasks *service.ScheduledTaskService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			grpcAuthMiddleware(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterRuleGoServer(srv, rules)
	v1.RegisterRunLogServer(srv, runLogs)
	v1.RegisterComponentServer(srv, components)
	v1.RegisterAdminServer(srv, admin)
	v1.RegisterChatServer(srv, chat)
	v1.RegisterTaskBoardServiceServer(srv, taskService)
	v1.RegisterServiceManagementServiceServer(srv, taskService)
	v1.RegisterScheduledTaskServiceServer(srv, scheduledTasks)
	return srv
}

func grpcAuthMiddleware() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, in interface{}) (interface{}, error) {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				if userID := md.Get("x-user-id"); len(userID) > 0 && userID[0] != "" {
					ctx = context.WithValue(ctx, userIDKey, userID[0])
				}
				if projectPath := md.Get("x-project-path"); len(projectPath) > 0 && projectPath[0] != "" {
					ctx = context.WithValue(ctx, projectPathKey, projectPath[0])
				}
			}
			return next(ctx, in)
		}
	}
}
