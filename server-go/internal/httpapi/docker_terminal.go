package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var dockerTerminalUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (a *API) dockerContainerTerminal(w http.ResponseWriter, r *http.Request, containerID string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	shell := normalizeDockerShell(r.URL.Query().Get("shell"))
	conn, err := dockerTerminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	command := exec.CommandContext(ctx, "docker", "exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLUMNS=120", "-e", "LINES=30", containerID, shell)
	terminal, err := pty.StartWithSize(command, &pty.Winsize{Rows: 30, Cols: 120})
	if err != nil {
		writeDockerTerminalMessage(conn, nil, fmt.Sprintf("无法打开容器终端：%s\n", err.Error()))
		return
	}
	defer terminal.Close()

	var writeMu sync.Mutex
	write := func(text string) bool {
		return writeDockerTerminalMessage(conn, &writeMu, text)
	}
	write(fmt.Sprintf("已连接 %s · %s\n", containerID, shell))

	done := make(chan struct{})
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, readErr := terminal.Read(buffer)
			if n > 0 && !write(string(buffer[:n])) {
				return
			}
			if readErr != nil {
				return
			}
		}
	}()
	go func() {
		defer close(done)
		// Close the websocket when the container process exits so the read loop
		// below unblocks; otherwise it would hang on ReadMessage until the client
		// disconnects, leaking this goroutine and the connection.
		defer conn.Close()
		if err := command.Wait(); err != nil && ctx.Err() == nil {
			write(fmt.Sprintf("\n终端已断开：%s\n", err.Error()))
			return
		}
		if ctx.Err() == nil {
			write("\n终端已关闭。\n")
		}
	}()
	defer func() {
		_ = terminal.Close()
		cancel()
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}()

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		text := string(payload)
		if strings.TrimSpace(text) == "" {
			text = "\n"
		}
		if _, err := io.WriteString(terminal, text); err != nil {
			write(fmt.Sprintf("\n终端输入失败：%s\n", err.Error()))
			return
		}
	}
}

func normalizeDockerShell(shell string) string {
	switch strings.TrimSpace(shell) {
	case "/bin/bash", "bash":
		return "/bin/bash"
	case "/bin/ash", "ash":
		return "/bin/ash"
	case "/bin/sh", "sh":
		return "/bin/sh"
	default:
		return "/bin/sh"
	}
}

func writeDockerTerminalMessage(conn *websocket.Conn, mu *sync.Mutex, text string) bool {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
		return false
	}
	return true
}
