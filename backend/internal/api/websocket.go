package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/os-baka/backend/internal/config"
	"golang.org/x/crypto/ssh"
)

// WebSocket message types exchanged between frontend and backend.
//
// Client → Server:
//
//	{type: "auth",   host, port, username, method, credential, keyPassphrase?}
//	{type: "input",  data}
//	{type: "resize", cols, rows}
//
// Server → Client:
//
//	{type: "connected"}
//	{type: "output",       data (base64)}
//	{type: "error",        message}
//	{type: "disconnected"}

// --- Message structures ---

type wsClientMessage struct {
	Type          string `json:"type"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	Username      string `json:"username,omitempty"`
	Method        string `json:"method,omitempty"` // "password" or "key"
	Credential    string `json:"credential,omitempty"`
	KeyPassphrase string `json:"keyPassphrase,omitempty"`
	Data          string `json:"data,omitempty"`
	Cols          int    `json:"cols,omitempty"`
	Rows          int    `json:"rows,omitempty"`
}

type wsServerMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// --- WebSocket handler ---

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS is handled by Gin middleware
	},
}

type SSHHandler struct {
	Config *config.Config
}

func NewSSHHandler(cfg *config.Config) *SSHHandler {
	return &SSHHandler{Config: cfg}
}

// HandleSSH upgrades the HTTP connection to WebSocket and proxies SSH I/O.
// Authentication is performed via the first WebSocket message (type: "auth")
// instead of the HTTP Authorization header, because the browser WebSocket API
// does not support custom headers.
//
// @Summary      SSH WebSocket proxy
// @Description  WebSocket endpoint for interactive SSH sessions
// @Tags         websocket
// @Param        token query string false "JWT token (optional, for browser clients that cannot set headers)"
// @Router       /ws/ssh [get]
func (h *SSHHandler) HandleSSH(c *gin.Context) {
	// Optional: allow JWT via query param for WebSocket (browsers can't set headers)
	if token := c.Query("token"); token != "" {
		c.Request.Header.Set("Authorization", "Bearer "+token)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Set read deadline for auth message
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		slog.Error("Failed to set read deadline", "error", err)
		return
	}

	// Wait for auth message
	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		writeWSError(conn, "Failed to read auth message")
		return
	}

	var authMsg wsClientMessage
	if err := json.Unmarshal(rawMsg, &authMsg); err != nil || authMsg.Type != "auth" {
		writeWSError(conn, "First message must be type 'auth'")
		return
	}

	// Validate auth fields
	if authMsg.Host == "" || authMsg.Username == "" || authMsg.Credential == "" {
		writeWSError(conn, "Missing required auth fields: host, username, credential")
		return
	}
	if authMsg.Port <= 0 || authMsg.Port > 65535 {
		authMsg.Port = 22
	}

	// Build SSH config
	sshConfig, err := buildSSHConfig(authMsg)
	if err != nil {
		writeWSError(conn, fmt.Sprintf("Invalid SSH configuration: %v", err))
		return
	}

	// Connect to SSH server
	addr := net.JoinHostPort(authMsg.Host, fmt.Sprintf("%d", authMsg.Port))
	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		writeWSError(conn, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer sshConn.Close()

	// Open session
	session, err := sshConn.NewSession()
	if err != nil {
		writeWSError(conn, fmt.Sprintf("SSH session failed: %v", err))
		return
	}
	defer session.Close()

	// Request PTY
	cols, rows := 80, 24
	if err := session.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		writeWSError(conn, fmt.Sprintf("PTY request failed: %v", err))
		return
	}

	// Get stdin/stdout pipes
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		writeWSError(conn, fmt.Sprintf("Failed to get stdin: %v", err))
		return
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		writeWSError(conn, fmt.Sprintf("Failed to get stdout: %v", err))
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		writeWSError(conn, fmt.Sprintf("Failed to start shell: %v", err))
		return
	}

	// Notify client of successful connection
	writeWSMessage(conn, wsServerMessage{Type: "connected"})

	// Clear read deadline for interactive session
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		slog.Error("Failed to clear read deadline", "error", err)
	}

	// Use WaitGroup + done channel for clean shutdown
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Goroutine: SSH stdout → WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				select {
				case <-done:
					return
				default:
				}
				return
			}
			if n > 0 {
				encoded := base64.StdEncoding.EncodeToString(buf[:n])
				writeWSMessage(conn, wsServerMessage{Type: "output", Data: encoded})
			}
		}
	}()

	// Goroutine: WebSocket → SSH stdin (main read loop)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		for {
			_, rawMsg, err := conn.ReadMessage()
			if err != nil {
				return // WebSocket closed
			}

			var msg wsClientMessage
			if err := json.Unmarshal(rawMsg, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "input":
				if _, err := stdinPipe.Write([]byte(msg.Data)); err != nil {
					return
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					// WindowChange is best-effort
					_ = session.WindowChange(msg.Rows, msg.Cols)
				}
			}
		}
	}()

	// Wait for session to finish
	_ = session.Wait()

	// Signal goroutines to stop
	select {
	case <-done:
	default:
		close(done)
	}

	writeWSMessage(conn, wsServerMessage{Type: "disconnected"})
	wg.Wait()

	slog.Info("SSH session ended", "host", authMsg.Host, "user", authMsg.Username)
}

// buildSSHConfig creates an ssh.ClientConfig from the auth message.
func buildSSHConfig(msg wsClientMessage) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            msg.Username,
		Timeout:         15 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // PXE provisioning environment on trusted LAN
	}

	switch strings.ToLower(msg.Method) {
	case "password":
		config.Auth = []ssh.AuthMethod{
			ssh.Password(msg.Credential),
		}
	case "key":
		var signer ssh.Signer
		var err error
		if msg.KeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(msg.Credential), []byte(msg.KeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(msg.Credential))
		}
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", msg.Method)
	}

	return config, nil
}

// --- helpers ---

var wsMu sync.Mutex

func writeWSMessage(conn *websocket.Conn, msg wsServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	wsMu.Lock()
	defer wsMu.Unlock()
	_ = conn.WriteMessage(websocket.TextMessage, data)
}

func writeWSError(conn *websocket.Conn, message string) {
	writeWSMessage(conn, wsServerMessage{Type: "error", Message: message})
}

// ValidateSSHTarget checks if the target host is reachable on the given port.
// Useful for a pre-flight check before establishing WebSocket.
func ValidateSSHTarget(host string, port int) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("host %s is not reachable: %w", addr, err)
	}
	conn.Close()
	return nil
}
