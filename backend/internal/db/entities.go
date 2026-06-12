package db

import "time"

// This file defines the GORM entity models that own the database schema via
// AutoMigrate. Column tags are pinned to the exact names the existing SQL in
// handlers/workers expects, so both the GORM query path and the raw-SQL path
// (running on the same glebarez/modernc connection) see an identical schema.

// UserEntity maps the users table.
type UserEntity struct {
	ID             string    `gorm:"column:id;primaryKey"`
	Email          string    `gorm:"column:email;uniqueIndex;not null"`
	HashedPassword string    `gorm:"column:hashed_password;not null"`
	Role           string    `gorm:"column:role;not null;default:admin"`
	IsActive       bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:datetime;default:CURRENT_TIMESTAMP"`
}

// TableName pins the table name.
func (UserEntity) TableName() string { return "users" }

// SessionEntity maps the sessions table.
type SessionEntity struct {
	ID           string    `gorm:"column:id;primaryKey"`
	UserID       string    `gorm:"column:user_id;not null"`
	RefreshToken string    `gorm:"column:refresh_token;uniqueIndex;not null"`
	ExpiresAt    time.Time `gorm:"column:expires_at;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP"`
}

func (SessionEntity) TableName() string { return "sessions" }

// HostEntity maps the hosts table.
type HostEntity struct {
	ID          string    `gorm:"column:id;primaryKey"`
	Hostname    string    `gorm:"column:hostname;uniqueIndex;not null"`
	IPAddress   string    `gorm:"column:ip_address;not null"`
	OSInfo      string    `gorm:"column:os_info"`
	CPUCount    int       `gorm:"column:cpu_count;not null;default:0"`
	TotalMemory int64     `gorm:"column:total_memory;not null;default:0"`
	TotalDisk   int64     `gorm:"column:total_disk;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:datetime;default:CURRENT_TIMESTAMP"`
}

func (HostEntity) TableName() string { return "hosts" }

// ContainerEntity maps the containers table.
type ContainerEntity struct {
	ID        string    `gorm:"column:id;primaryKey"`
	DockerID  string    `gorm:"column:docker_id;uniqueIndex;not null"`
	Name      string    `gorm:"column:name;not null"`
	Image     string    `gorm:"column:image;not null"`
	Status    string    `gorm:"column:status;not null;default:unknown"`
	Ports     string    `gorm:"column:ports"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;default:CURRENT_TIMESTAMP"`
}

func (ContainerEntity) TableName() string { return "containers" }

// ContainerEventEntity maps the container_events table.
type ContainerEventEntity struct {
	ID                string    `gorm:"column:id;primaryKey"`
	ContainerDockerID string    `gorm:"column:container_docker_id"`
	ContainerName     string    `gorm:"column:container_name"`
	EventType         string    `gorm:"column:event_type;not null"`
	Timestamp         time.Time `gorm:"column:timestamp;type:datetime;default:CURRENT_TIMESTAMP"`
	Metadata          string    `gorm:"column:metadata"`
}

func (ContainerEventEntity) TableName() string { return "container_events" }

// ComposeStackEntity maps the compose_stacks table.
type ComposeStackEntity struct {
	ID            string    `gorm:"column:id;primaryKey"`
	Name          string    `gorm:"column:name;uniqueIndex;not null"`
	ProjectDir    string    `gorm:"column:project_dir"`
	Status        string    `gorm:"column:status;not null;default:unknown"`
	ServicesCount int       `gorm:"column:services_count;not null;default:0"`
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:datetime;default:CURRENT_TIMESTAMP"`
}

func (ComposeStackEntity) TableName() string { return "compose_stacks" }

// ComposeServiceEntity maps the compose_services table.
type ComposeServiceEntity struct {
	ID          string `gorm:"column:id;primaryKey"`
	StackID     string `gorm:"column:stack_id;not null;uniqueIndex:idx_stack_service"`
	Name        string `gorm:"column:name;not null;uniqueIndex:idx_stack_service"`
	ContainerID string `gorm:"column:container_id"`
	Status      string `gorm:"column:status;not null;default:unknown"`
	Ports       string `gorm:"column:ports"`
}

func (ComposeServiceEntity) TableName() string { return "compose_services" }

// AlertRuleEntity maps the alert_rules table.
type AlertRuleEntity struct {
	ID              string    `gorm:"column:id;primaryKey"`
	MetricType      string    `gorm:"column:metric_type;not null"`
	Threshold       float64   `gorm:"column:threshold;not null"`
	Operator        string    `gorm:"column:operator;not null;default:>="`
	DurationSeconds int       `gorm:"column:duration_seconds;not null;default:60"`
	Enabled         bool      `gorm:"column:enabled;not null;default:true"`
	CreatedAt       time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP"`
}

func (AlertRuleEntity) TableName() string { return "alert_rules" }

// AlertEntity maps the alerts table.
type AlertEntity struct {
	ID           string    `gorm:"column:id;primaryKey"`
	RuleID       *string   `gorm:"column:rule_id;index:idx_alerts_rule_id"`
	Severity     string    `gorm:"column:severity;not null;default:warning"`
	Message      string    `gorm:"column:message;not null"`
	Acknowledged bool      `gorm:"column:acknowledged;not null;default:false"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP;index:idx_alerts_created_at,sort:desc"`
}

func (AlertEntity) TableName() string { return "alerts" }

// entityModels lists every model managed by AutoMigrate.
func entityModels() []any {
	return []any{
		&UserEntity{}, &SessionEntity{}, &HostEntity{},
		&ContainerEntity{}, &ContainerEventEntity{},
		&ComposeStackEntity{}, &ComposeServiceEntity{},
		&AlertRuleEntity{}, &AlertEntity{},
	}
}
