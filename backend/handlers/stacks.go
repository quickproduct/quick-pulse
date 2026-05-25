package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"

	"quickpulse/backend/models"
)

// ListStacksHandler handles GET /api/v1/stacks
func ListStacksHandler(w http.ResponseWriter, r *http.Request) {
	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list containers: %v", err))
		return
	}

	stacksMap := make(map[string]*models.StackDetailResponse)

	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		if project == "" || project == "quickpulse" {
			continue
		}

		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		} else {
			name = c.ID[:12]
		}

		// Check if qp- prefix
		if strings.HasPrefix(name, "qp-") {
			continue
		}

		if _, exists := stacksMap[project]; !exists {
			workingDir := c.Labels["com.docker.compose.project.working_dir"]
			stacksMap[project] = &models.StackDetailResponse{
				StackResponse: models.StackResponse{
					Name:       project,
					ProjectDir: workingDir,
					Status:     "unknown",
				},
				Services: []models.ComposeService{},
			}
		}

		serviceName := c.Labels["com.docker.compose.service"]
		if serviceName == "" {
			serviceName = name
		}

		stacksMap[project].Services = append(stacksMap[project].Services, models.ComposeService{
			Name:        serviceName,
			ContainerID: c.ID[:12],
			Status:      c.State,
		})
	}

	result := []models.StackDetailResponse{}
	for _, stack := range stacksMap {
		total := len(stack.Services)
		running := 0
		for _, svc := range stack.Services {
			if svc.Status == "running" {
				running++
			}
		}

		if running == total && total > 0 {
			stack.Status = "running"
		} else if running > 0 {
			stack.Status = "partial"
		} else {
			stack.Status = "stopped"
		}
		stack.ServicesCount = total
		result = append(result, *stack)
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetStackHandler handles GET /api/v1/stacks/{name}
func GetStackHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", name))),
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to inspect stack containers: %v", err))
		return
	}

	if len(containers) == 0 {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Stack %s not found", name))
		return
	}

	var projectDir string
	services := []models.ComposeService{}
	runningCount := 0

	for _, c := range containers {
		if projectDir == "" {
			projectDir = c.Labels["com.docker.compose.project.working_dir"]
		}

		cName := ""
		if len(c.Names) > 0 {
			cName = strings.TrimPrefix(c.Names[0], "/")
		} else {
			cName = c.ID[:12]
		}

		serviceName := c.Labels["com.docker.compose.service"]
		if serviceName == "" {
			serviceName = cName
		}

		services = append(services, models.ComposeService{
			Name:        serviceName,
			ContainerID: c.ID[:12],
			Status:      c.State,
		})

		if c.State == "running" {
			runningCount++
		}
	}

	status := "stopped"
	if runningCount == len(services) && len(services) > 0 {
		status = "running"
	} else if runningCount > 0 {
		status = "partial"
	}

	detail := models.StackDetailResponse{
		StackResponse: models.StackResponse{
			Name:          name,
			ProjectDir:    projectDir,
			Status:        status,
			ServicesCount: len(services),
		},
		Services: services,
	}

	WriteJSON(w, http.StatusOK, detail)
}

// StartStackHandler handles POST /api/v1/stacks/{name}/start
func StartStackHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", name))),
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	failed := []string{}
	for _, c := range containers {
		if c.State != "running" {
			err = cli.ContainerStart(ctx, c.ID, container.StartOptions{})
			if err != nil {
				failed = append(failed, c.ID[:12])
			}
		}
	}

	WriteJSON(w, http.StatusOK, models.StackActionResponse{
		Success:   len(failed) == 0,
		Message:   fmt.Sprintf("Stack %s started (failed: %s)", name, strings.Join(failed, ",")),
		StackName: name,
	})
}

// StopStackHandler handles POST /api/v1/stacks/{name}/stop
func StopStackHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", name))),
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	failed := []string{}
	for _, c := range containers {
		if c.State == "running" {
			err = cli.ContainerStop(ctx, c.ID, container.StopOptions{})
			if err != nil {
				failed = append(failed, c.ID[:12])
			}
		}
	}

	WriteJSON(w, http.StatusOK, models.StackActionResponse{
		Success:   len(failed) == 0,
		Message:   fmt.Sprintf("Stack %s stopped (failed: %s)", name, strings.Join(failed, ",")),
		StackName: name,
	})
}

// RestartStackHandler handles POST /api/v1/stacks/{name}/restart
func RestartStackHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Docker daemon unavailable")
		return
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", name))),
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	failed := []string{}
	for _, c := range containers {
		err = cli.ContainerRestart(ctx, c.ID, container.StopOptions{})
		if err != nil {
			failed = append(failed, c.ID[:12])
		}
	}

	WriteJSON(w, http.StatusOK, models.StackActionResponse{
		Success:   len(failed) == 0,
		Message:   fmt.Sprintf("Stack %s restarted (failed: %s)", name, strings.Join(failed, ",")),
		StackName: name,
	})
}
