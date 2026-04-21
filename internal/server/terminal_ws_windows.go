//go:build windows

package server

import (
	"net/http"

	"ruleGoKratos/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// RegisterTerminalWebSocket 在 Windows 上不可用（PTY 交互终端需 Linux/macOS 或 Docker/WSL）。
func RegisterTerminalWebSocket(srv *khttp.Server, _ *service.AdminService, _ log.Logger) {
	srv.HandleFunc("/api/v1/admin/terminal/ws", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.Error(w, "interactive terminal requires Linux/macOS or Docker/WSL", http.StatusNotImplemented)
	})
}
