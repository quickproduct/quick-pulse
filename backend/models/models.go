package models

import (
	"time"
)

// User represents the user db/internal structure
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"-"`
	Role           string    `json:"role"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserResponse matches the response for /me and register
type UserResponse struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// LoginRequest defines parameters for logging in
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse defines credentials returned on successful login
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// RefreshRequest defines token refresh request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ChangePasswordRequest defines change password payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// RegisterRequest defines parameters for registering a new account
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Host represents server details
type Host struct {
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ip_address"`
	OSInfo      string `json:"os_info"`
	CPUCount    int    `json:"cpu_count"`
	TotalMemory uint64 `json:"total_memory"`
	TotalDisk   uint64 `json:"total_disk"`
}

// MetricSnapshot contains a reading of VPS metrics
type MetricSnapshot struct {
	CPUPercent     float64   `json:"cpu_percent"`
	MemoryPercent  float64   `json:"memory_percent"`
	MemoryUsed     uint64    `json:"memory_used"`
	MemoryTotal    uint64    `json:"memory_total"`
	DiskPercent    float64   `json:"disk_percent"`
	DiskReadBytes  uint64    `json:"disk_read_bytes"`
	DiskWriteBytes uint64    `json:"disk_write_bytes"`
	NetBytesSent   uint64    `json:"net_bytes_sent"`
	NetBytesRecv   uint64    `json:"net_bytes_recv"`
	Load1m         float64   `json:"load_1m"`
	Load5m         float64   `json:"load_5m"`
	Load15m        float64   `json:"load_15m"`
	ProcessCount   int       `json:"process_count"`
	UptimeSeconds  int       `json:"uptime_seconds"`
	Timestamp      time.Time `json:"timestamp,omitempty"`
}

// MetricHistoryPoint represents a single aggregated data point
type MetricHistoryPoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

// MetricHistoryResponse represents the collection of points over a timeframe
type MetricHistoryResponse struct {
	Metric string               `json:"metric"`
	Range  string               `json:"range"`
	Data   []MetricHistoryPoint `json:"data"`
}

// ContainerResponse matches docker summary details
type ContainerResponse struct {
	DockerID   string      `json:"docker_id"`
	Name       string      `json:"name"`
	Image      string      `json:"image"`
	Status     string      `json:"status"`
	Ports      interface{} `json:"ports"` // Can be list/dict/null
	State      string      `json:"state,omitempty"`
	StatusText string      `json:"status_text,omitempty"`
	CreatedAt  *time.Time  `json:"created_at,omitempty"`
	UpdatedAt  *time.Time  `json:"updated_at,omitempty"`
}

// ContainerDetailResponse includes inspection details
type ContainerDetailResponse struct {
	ContainerResponse
	Env             []string               `json:"env"`
	NetworkSettings map[string]interface{} `json:"network_settings"`
	Mounts          []interface{}          `json:"mounts"`
	ResourceUsage   map[string]interface{} `json:"resource_usage"`
}

// ContainerActionResponse matches action outputs
type ContainerActionResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ContainerID string `json:"container_id"`
}

// EventResponse represents docker events recorded
type EventResponse struct {
	ID                string                 `json:"id"`
	ContainerDockerID string                 `json:"container_docker_id"`
	ContainerName     string                 `json:"container_name"`
	EventType         string                 `json:"event_type"`
	Timestamp         *time.Time             `json:"timestamp"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// ComposeService defines stack's child container
type ComposeService struct {
	Name        string `json:"name"`
	ContainerID string `json:"container_id"`
	Status      string `json:"status"`
}

// StackResponse matches Compose summary details
type StackResponse struct {
	Name          string `json:"name"`
	ProjectDir    string `json:"project_dir"`
	Status        string `json:"status"`
	ServicesCount int    `json:"services_count"`
}

// StackDetailResponse includes nested services details
type StackDetailResponse struct {
	StackResponse
	Services []ComposeService `json:"services"`
}

// StackActionResponse matches compose control actions
type StackActionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	StackName string `json:"stack_name"`
}

// AlertRuleCreate defines rule creation payload
type AlertRuleCreate struct {
	MetricType      string  `json:"metric_type"`
	Threshold       float64 `json:"threshold"`
	Operator        string  `json:"operator"`
	DurationSeconds int     `json:"duration_seconds"`
	Enabled         bool    `json:"enabled"`
}

// AlertRuleUpdate defines rule modification payload
type AlertRuleUpdate struct {
	Threshold       *float64 `json:"threshold"`
	Operator        *string  `json:"operator"`
	DurationSeconds *int     `json:"duration_seconds"`
	Enabled         *bool    `json:"enabled"`
}

// AlertRuleResponse matches rules returned
type AlertRuleResponse struct {
	ID              string     `json:"id"`
	MetricType      string     `json:"metric_type"`
	Threshold       float64    `json:"threshold"`
	Operator        string     `json:"operator"`
	DurationSeconds int        `json:"duration_seconds"`
	Enabled         bool       `json:"enabled"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}

// AlertResponse matches alert entries
type AlertResponse struct {
	ID           string     `json:"id"`
	RuleID       *string    `json:"rule_id"`
	Severity     string     `json:"severity"`
	Message      string     `json:"message"`
	Acknowledged bool       `json:"acknowledged"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
}

// DashboardResponse details the home screen overview data
type DashboardResponse struct {
	Host         *Host           `json:"host"`
	Metrics      *MetricSnapshot `json:"metrics"`
	Containers   map[string]int  `json:"containers"` // {"total": X, "running": Y, "stopped": Z}
	RecentEvents []EventResponse `json:"recent_events"`
	StackSummary map[string]int  `json:"stack_summary"` // {"total": X, "running": Y, "partial": Z, "stopped": W}
}
