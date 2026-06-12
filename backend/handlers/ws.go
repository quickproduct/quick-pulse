package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gorilla/websocket"

	"quickpulse/backend/auth"
	"quickpulse/backend/db"
	"quickpulse/backend/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for CORS compatibility
	},
}

func validateWSAuth(r *http.Request) (string, bool) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return "", false
	}
	claims, err := auth.VerifyToken(token, "access")
	if err != nil {
		return "", false
	}
	var isActive int
	err = db.DB.QueryRow("SELECT is_active FROM users WHERE id = ?", claims.Sub).Scan(&isActive)
	if err != nil || isActive == 0 {
		return "", false
	}
	return claims.Sub, true
}

// HandleWSChannel upgrades connection and maps it to a standard pub/sub channel
func HandleWSChannel(channel string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := validateWSAuth(r)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WS upgrade failed for channel %s: %v", channel, err)
			return
		}

		log.Printf("WS client connected to channel %s: user %s", channel, userID)
		ws.Manager.Connect(channel, conn)

		// Loop to keep connection alive and detect client disconnect
		defer func() {
			ws.Manager.Disconnect(channel, conn)
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}
}

type wsLogWriter struct {
	mu          sync.Mutex
	conn        *websocket.Conn
	containerID string
	paused      *bool
	buf         []byte
}

func (w *wsLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx == -1 {
			break
		}
		line := w.buf[:idx]
		w.buf = w.buf[idx+1:]

		if !*w.paused {
			trimmed := string(bytes.TrimRight(line, "\r\n"))
			err = w.conn.WriteJSON(map[string]interface{}{
				"line":         trimmed,
				"container_id": w.containerID,
			})
			if err != nil {
				return len(p), err
			}
		}
	}
	return len(p), nil
}

// HandleWSLogs handles streaming docker logs via WebSocket /ws/logs/{container_id}
func HandleWSLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("container_id")
	if containerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Missing container ID"))
		return
	}

	_, ok := validateWSAuth(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed for container logs: %v", err)
		return
	}

	channel := "logs:" + containerID
	ws.Manager.Connect(channel, conn)

	defer func() {
		ws.Manager.Disconnect(channel, conn)
	}()

	cli, err := getDockerClient()
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "Docker daemon unavailable"})
		return
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
	})
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "Failed to stream logs: " + err.Error()})
		return
	}
	defer reader.Close()

	paused := false
	writer := &wsLogWriter{
		conn:        conn,
		containerID: containerID,
		paused:      &paused,
	}

	// Read logs and write to WS
	go func() {
		_, err := stdcopy.StdCopy(writer, writer, reader)
		if err != nil && !errors.Is(err, io.EOF) {
			// Fallback for TTY containers or when multiplexing fails
			_, _ = io.Copy(writer, reader)
		}
	}()

	// Read commands from websocket
	for {
		var msg map[string]string
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		action := msg["action"]
		writer.mu.Lock()
		if action == "pause" {
			paused = true
		} else if action == "resume" {
			paused = false
		}
		writer.mu.Unlock()
	}
}

var terminalBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// HandleWSContainerTerminal handles interactive shell sessions over WebSocket.
func HandleWSContainerTerminal(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Missing container ID"))
		return
	}

	userID, ok := validateWSAuth(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	var role string
	err := db.DB.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil || role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden: Admin privileges required"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WS upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	cli, err := getDockerClient()
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "Docker daemon unavailable"})
		return
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Create Exec configuration for shell
	execCfg := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"/bin/sh"},
	}

	execCreate, err := cli.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "Failed to create exec instance: " + err.Error()})
		return
	}

	// 2. Attach to Exec instance (Hijacks the connection)
	resp, err := cli.ContainerExecAttach(ctx, execCreate.ID, types.ExecStartCheck{
		Tty: true,
	})
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "Failed to attach to exec: " + err.Error()})
		return
	}
	defer resp.Close()

	// 3. Pipe stdout/stderr from container to WebSocket
	go func() {
		defer cancel()
		buf := terminalBufPool.Get().([]byte)
		defer terminalBufPool.Put(buf)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := resp.Reader.Read(buf)
				if n > 0 {
					errWrite := conn.WriteMessage(websocket.TextMessage, buf[:n])
					if errWrite != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}
	}()

	// 4. Pipe stdin from WebSocket to container, and listen for resize events
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(payload, &msg); err != nil {
			// Fallback to raw text if not json
			_, _ = resp.Conn.Write(payload)
			continue
		}

		action, _ := msg["action"].(string)
		if action == "input" {
			data, _ := msg["data"].(string)
			_, _ = resp.Conn.Write([]byte(data))
		} else if action == "resize" {
			colsVal, hasCols := msg["cols"]
			rowsVal, hasRows := msg["rows"]
			if hasCols && hasRows {
				cols := int(colsVal.(float64))
				rows := int(rowsVal.(float64))
				_ = cli.ContainerExecResize(ctx, execCreate.ID, container.ResizeOptions{
					Height: uint(rows),
					Width:  uint(cols),
				})
			}
		}
	}
}
