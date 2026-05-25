package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	// Also scan `./stacks` directory for non-running compose projects
	if entries, err := os.ReadDir("stacks"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				stackName := entry.Name()
				if _, exists := stacksMap[stackName]; !exists {
					// Check if a compose file exists in this directory
					projectDir := filepath.Join("stacks", stackName)
					hasCompose := false
					composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
					for _, f := range composeFiles {
						if _, err := os.Stat(filepath.Join(projectDir, f)); err == nil {
							hasCompose = true
							break
						}
					}
					if hasCompose {
						stacksMap[stackName] = &models.StackDetailResponse{
							StackResponse: models.StackResponse{
								Name:          stackName,
								ProjectDir:    projectDir,
								Status:        "stopped",
								ServicesCount: 0,
								Running:       0,
								Total:         0,
							},
							Services: []models.ComposeService{},
						}
					}
				}
			}
		}
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
		stack.Running = running
		stack.Total = total
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
			Running:       runningCount,
			Total:         len(services),
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

func getStackDir(name string) string {
	cli, err := getDockerClient()
	if err == nil {
		defer cli.Close()
		ctx := context.Background()
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", name))),
		})
		if err == nil && len(containers) > 0 {
			dir := containers[0].Labels["com.docker.compose.project.working_dir"]
			if dir != "" {
				return dir
			}
		}
	}
	return filepath.Join("stacks", name)
}

func getComposeFilePath(dir string) string {
	files := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return filepath.Join(dir, "docker-compose.yml") // Default fallback
}

// GetStackConfigHandler handles GET /api/v1/stacks/{name}/config
func GetStackConfigHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	projectDir := getStackDir(name)
	composePath := getComposeFilePath(projectDir)

	content, err := os.ReadFile(composePath)
	if err != nil {
		if os.IsNotExist(err) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Compose file not found for stack %s", name))
			return
		}
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read compose file: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"name":   name,
		"config": string(content),
	})
}

type SaveStackConfigRequest struct {
	Config string `json:"config"`
}

// SaveStackConfigHandler handles POST /api/v1/stacks/{name}/config
func SaveStackConfigHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	var req SaveStackConfigRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	projectDir := getStackDir(name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create stack directory: %v", err))
		return
	}

	composePath := getComposeFilePath(projectDir)
	if err := os.WriteFile(composePath, []byte(req.Config), 0644); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to write compose file: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Compose configuration for stack %s saved successfully", name),
	})
}

type CreateStackRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// CreateStackHandler handles POST /api/v1/stacks
func CreateStackHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateStackRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "Stack name is required")
		return
	}

	// Validate stack name to prevent directory traversal
	if strings.Contains(req.Name, "..") || strings.ContainsAny(req.Name, "/\\") {
		WriteError(w, http.StatusBadRequest, "Invalid stack name")
		return
	}

	projectDir := filepath.Join("stacks", req.Name)
	if _, err := os.Stat(projectDir); err == nil {
		WriteError(w, http.StatusConflict, fmt.Sprintf("Stack %s already exists", req.Name))
		return
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create stack directory: %v", err))
		return
	}

	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if req.Config == "" {
		req.Config = fmt.Sprintf("version: '3.8'\nservices:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"80:80\"\n")
	}

	if err := os.WriteFile(composePath, []byte(req.Config), 0644); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to write compose file: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Stack %s created successfully", req.Name),
	})
}

type flushWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

// DeployStackHandler handles POST /api/v1/stacks/{name}/deploy
func DeployStackHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		WriteError(w, http.StatusBadRequest, "Missing stack name")
		return
	}

	projectDir := getStackDir(name)

	// Check if directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Stack directory %s not found", projectDir))
		return
	}

	// Set headers for streaming logs
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
		flusher.Flush()
	}

	fw := &flushWriter{w: w, f: flusher}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	_, _ = fmt.Fprintf(fw, "Starting deployment for stack %s...\n", name)
	_, _ = fmt.Fprintf(fw, "Working directory: %s\n\n", projectDir)

	cmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--remove-orphans")
	cmd.Dir = projectDir
	cmd.Stdout = fw
	cmd.Stderr = fw

	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintf(fw, "\n[ERROR] Deployment failed: %v\n", err)
	} else {
		_, _ = fmt.Fprintln(fw, "\n[SUCCESS] Stack deployed successfully!")
	}
}
