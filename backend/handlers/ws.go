package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gorilla/websocket"

	"quickpulse/backend/auth"
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
