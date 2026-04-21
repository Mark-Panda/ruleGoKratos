//go:build !windows

package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"ruleGoKratos/internal/service"

	"github.com/creack/pty"
	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/websocket"
)

// RegisterTerminalWebSocket 注册交互式终端 WebSocket：PTY + bash/sh，cwd 白名单与 RunTerminal 一致。
func RegisterTerminalWebSocket(srv *khttp.Server, admin *service.AdminService, logger log.Logger) {
	h := &terminalWSHandler{admin: admin, log: log.NewHelper(logger)}
	srv.HandleFunc("/api/v1/admin/terminal/ws", h.serve)
}

type terminalWSHandler struct {
	admin *service.AdminService
	log   *log.Helper
}

var terminalUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *terminalWSHandler) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cwd := r.URL.Query().Get("cwd")
	validated, err := h.admin.ValidateTerminalCwd(cwd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := terminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorf("terminal ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	shell := "/bin/bash"
	if _, stErr := os.Stat(shell); os.IsNotExist(stErr) {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-i")
	cmd.Dir = validated
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n[server] 启动 shell 失败: "+err.Error()+"\r\n"))
		return
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					h.log.Warnf("pty read: %v", readErr)
				}
				_ = conn.Close()
				return
			}
		}
	}()

	for {
		mt, data, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}
		if mt == websocket.TextMessage {
			var msg struct {
				Type string `json:"type"`
				Rows uint16 `json:"rows"`
				Cols uint16 `json:"cols"`
			}
			if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" && msg.Rows > 0 && msg.Cols > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols})
			}
			continue
		}
		if mt == websocket.BinaryMessage {
			_, _ = ptmx.Write(data)
		}
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	wg.Wait()
}
