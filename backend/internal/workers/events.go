package workers

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"quickpulse/backend/internal/db"
	"quickpulse/backend/internal/ws"
)

var dockerEventMap = map[string]string{
	"start":         "container_start",
	"stop":          "container_stop",
	"restart":       "container_restart",
	"die":           "container_die",
	"health_status": "container_health",
	"create":        "container_create",
	"destroy":       "container_destroy",
}

// StartEventsWorker kicks off the Docker event listener in a goroutine
func StartEventsWorker() {
	zap.L().Sugar().Infof("Starting Docker events worker")
	go func() {
		consecutiveErrors := 0
		for {
			err := listenToDockerEvents()
			if err != nil {
				consecutiveErrors++
				zap.L().Sugar().Infof("Events worker error (consecutive %d): %v", consecutiveErrors, err)
				backoff := 5 * consecutiveErrors
				if backoff > 300 {
					backoff = 300
				}
				time.Sleep(time.Duration(backoff) * time.Second)
				continue
			}
			consecutiveErrors = 0
			time.Sleep(1 * time.Second)
		}
	}()
}

func listenToDockerEvents() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()
	msgChan, errChan := cli.Events(ctx, types.EventsOptions{})

	for {
		select {
		case msg := <-msgChan:
			if msg.Type != "container" {
				continue
			}

			mappedType, exists := dockerEventMap[string(msg.Action)]
			if !exists {
				continue
			}

			attrs := msg.Actor.Attributes
			if attrs == nil {
				attrs = make(map[string]string)
			}
			project := attrs["com.docker.compose.project"]
			containerID := msg.Actor.ID
			if len(containerID) > 12 {
				containerID = containerID[:12]
			}
			containerName := attrs["name"]
			if containerName == "" {
				containerName = containerID
			}

			// Filter out QuickPulse's own infrastructure events
			if (len(containerName) >= 3 && containerName[:3] == "qp-") || project == "quickpulse" {
				continue
			}

			attrsJSON, _ := json.Marshal(attrs)
			eventID := uuid.New().String()
			nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")

			_, err = db.DB.Exec(
				"INSERT INTO container_events (id, container_docker_id, container_name, event_type, timestamp, metadata) VALUES (?, ?, ?, ?, ?, ?)",
				eventID, containerID, containerName, mappedType, nowStr, string(attrsJSON),
			)
			if err != nil {
				zap.L().Sugar().Infof("Warning: failed to store container event: %v", err)
			}

			// Broadcast
			timestampRFC := time.Now().UTC().Format(time.RFC3339)
			ws.Manager.Broadcast("events", map[string]interface{}{
				"id":                  eventID,
				"container_docker_id": containerID,
				"container_name":      containerName,
				"event_type":          mappedType,
				"timestamp":           timestampRFC,
				"metadata":            attrs,
			})

			ws.Manager.Broadcast("container-status", map[string]interface{}{
				"container_id": containerID,
				"name":         containerName,
				"status":       msg.Action,
				"timestamp":    timestampRFC,
			})

		case err := <-errChan:
			if err != nil {
				return err
			}
			return nil
		}
	}
}
