package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"quickpulse/backend/models"
)

// Helper to get docker client
func getDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// Helper to normalize status
func normalizeStatus(state string) string {
	state = strings.ToLower(state)
	switch state {
	case "running":
		return "running"
	case "paused":
		return "paused"
	case "exited", "dead":
		return "stopped"
	case "restarting":
		return "restarting"
	default:
		return "unknown"
	}
}

// ListContainersHandler handles GET /api/v1/containers
func ListContainersHandler(w http.ResponseWriter, r *http.Request) {
	showAll := r.URL.Query().Get("all") == "true"

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: showAll})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list containers: %v", err))
		return
	}

	result := []models.ContainerResponse{}
	for _, c := range containers {
		// Extract name
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		} else {
			name = c.ID[:12]
		}

		project := c.Labels["com.docker.compose.project"]

		// Filter out QuickPulse's own infrastructure
		if strings.HasPrefix(name, "qp-") || project == "quickpulse" {
			continue
		}

		// Convert ports
		var ports interface{} = c.Ports

		result = append(result, models.ContainerResponse{
			DockerID:   c.ID[:12],
			Name:       name,
			Image:      c.Image,
			Status:     normalizeStatus(c.State),
			Ports:      ports,
			State:      c.State,
			StatusText: c.Status,
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// InspectContainerHandler handles GET /api/v1/containers/{container_id}
func InspectContainerHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("container_id")
	if containerID == "" {
		WriteError(w, http.StatusBadRequest, "Missing container ID")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	jsonInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Container %s not found", containerID))
			return
		}
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to inspect container: %v", err))
		return
	}

	// Format ports
	var ports interface{} = jsonInfo.NetworkSettings.Ports

	name := strings.TrimPrefix(jsonInfo.Name, "/")

	containerResp := models.ContainerResponse{
		DockerID:   jsonInfo.ID[:12],
		Name:       name,
		Image:      jsonInfo.Config.Image,
		Status:     normalizeStatus(jsonInfo.State.Status),
		Ports:      ports,
		State:      jsonInfo.State.Status,
		StatusText: jsonInfo.State.Status,
	}

	// Empty / mock resource usage stats to keep frontend happy if it queries them
	resourceUsage := map[string]interface{}{
		"cpu_percent": 0.0,
		"mem_percent": 0.0,
		"mem_usage":   0,
		"mem_limit":   0,
	}

	detail := models.ContainerDetailResponse{
		ContainerResponse: containerResp,
		Env:               jsonInfo.Config.Env,
		NetworkSettings: map[string]interface{}{
			"Networks": jsonInfo.NetworkSettings.Networks,
		},
		Mounts:        make([]interface{}, 0),
		ResourceUsage: resourceUsage,
	}
	for _, m := range jsonInfo.Mounts {
		detail.Mounts = append(detail.Mounts, m)
	}

	WriteJSON(w, http.StatusOK, detail)
}

// StartContainerHandler handles POST /api/v1/containers/{container_id}/start
func StartContainerHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("container_id")
	if containerID == "" {
		WriteError(w, http.StatusBadRequest, "Missing container ID")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Container %s not found", containerID))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, models.ContainerActionResponse{
		Success:     true,
		Message:     fmt.Sprintf("Container %s started", containerID),
		ContainerID: containerID,
	})
}

// StopContainerHandler handles POST /api/v1/containers/{container_id}/stop
func StopContainerHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("container_id")
	if containerID == "" {
		WriteError(w, http.StatusBadRequest, "Missing container ID")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	// Stop container with default timeout
	err = cli.ContainerStop(ctx, containerID, container.StopOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Container %s not found", containerID))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, models.ContainerActionResponse{
		Success:     true,
		Message:     fmt.Sprintf("Container %s stopped", containerID),
		ContainerID: containerID,
	})
}

// RestartContainerHandler handles POST /api/v1/containers/{container_id}/restart
func RestartContainerHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("container_id")
	if containerID == "" {
		WriteError(w, http.StatusBadRequest, "Missing container ID")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	err = cli.ContainerRestart(ctx, containerID, container.StopOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Container %s not found", containerID))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, models.ContainerActionResponse{
		Success:     true,
		Message:     fmt.Sprintf("Container %s restarted", containerID),
		ContainerID: containerID,
	})
}

// GetContainerLogsHandler handles GET /api/v1/containers/{container_id}/logs
func GetContainerLogsHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("container_id")
	if containerID == "" {
		WriteError(w, http.StatusBadRequest, "Missing container ID")
		return
	}

	tailStr := r.URL.Query().Get("tail")
	tail := 100
	if tailStr != "" {
		if t, err := strconv.Atoi(tailStr); err == nil {
			tail = t
		}
	}
	if tail < 1 {
		tail = 1
	}
	if tail > 500 {
		tail = 500
	}

	since := r.URL.Query().Get("since")

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	}
	if since != "" {
		options.Since = since
	}

	reader, err := cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		if client.IsErrNotFound(err) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Container %s not found", containerID))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()

	// Demultiplex docker logs
	var stdoutBuf, stderrBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader)
	if err != nil && !errors.Is(err, io.EOF) {
		// fallback to direct reading if standard multiplexing fails (e.g. if TTY is enabled on container)
		// We read directly if StdCopy failed.
		stdoutBuf.Reset()
		_, _ = io.Copy(&stdoutBuf, reader)
	}

	// Combine stdout & stderr (or just stdout) and split into lines
	combined := stdoutBuf.String() + stderrBuf.String()
	lines := strings.Split(combined, "\n")

	// Remove trailing empty line if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Limit lines to tail count
	if len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"container_id": containerID,
		"logs":         lines,
	})
}
