package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB opens SQLite and runs migrations and seeds.
func InitDB() {
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = "quickpulse.db"
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}

	// Configure DB connections
	DB.SetMaxOpenConns(1) // SQLite works best with 1 open write connection if using simple locks
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(time.Hour)

	// Enable WAL (Write-Ahead Logging) for concurrent reads/writes
	if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Printf("Warning: failed to enable WAL mode: %v", err)
	}

	// Set auto_vacuum to INCREMENTAL so empty pages can be released back to the OS
	if _, err := DB.Exec("PRAGMA auto_vacuum = INCREMENTAL;"); err != nil {
		log.Printf("Warning: failed to set auto_vacuum incremental mode: %v", err)
	}

	// PRAGMAs tuned for the logs module (and the metrics workload). NORMAL
	// is safe under WAL; a 4 MB page cache is plenty for our 64 MB budget;
	// mmap is disabled because it would inflate RSS on Linux and we can't
	// afford the slack. temp_store=MEMORY keeps small sorts off disk.
	for _, pragma := range []string{
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA cache_size=-4000;",
		"PRAGMA temp_store=MEMORY;",
		"PRAGMA mmap_size=0;",
	} {
		if _, err := DB.Exec(pragma); err != nil {
			log.Printf("Warning: %s failed: %v", pragma, err)
		}
	}

	// Create tables
	createTables()

	// Seed data
	seedAdmin()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			hashed_password TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'admin',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			refresh_token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS hosts (
			id TEXT PRIMARY KEY,
			hostname TEXT UNIQUE NOT NULL,
			ip_address TEXT NOT NULL,
			os_info TEXT,
			cpu_count INTEGER NOT NULL DEFAULT 0,
			total_memory INTEGER NOT NULL DEFAULT 0,
			total_disk INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS host_metrics (
			time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			host_id TEXT NOT NULL,
			cpu_percent REAL NOT NULL DEFAULT 0.0,
			memory_percent REAL NOT NULL DEFAULT 0.0,
			memory_used INTEGER NOT NULL DEFAULT 0,
			memory_total INTEGER NOT NULL DEFAULT 0,
			disk_percent REAL NOT NULL DEFAULT 0.0,
			disk_read_bytes INTEGER NOT NULL DEFAULT 0,
			disk_write_bytes INTEGER NOT NULL DEFAULT 0,
			net_bytes_sent INTEGER NOT NULL DEFAULT 0,
			net_bytes_recv INTEGER NOT NULL DEFAULT 0,
			load_1m REAL NOT NULL DEFAULT 0.0,
			load_5m REAL NOT NULL DEFAULT 0.0,
			load_15m REAL NOT NULL DEFAULT 0.0,
			process_count INTEGER NOT NULL DEFAULT 0,
			uptime_seconds INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(host_id) REFERENCES hosts(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS containers (
			id TEXT PRIMARY KEY,
			docker_id TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			image TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'unknown',
			ports TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS container_events (
			id TEXT PRIMARY KEY,
			container_docker_id TEXT,
			container_name TEXT,
			event_type TEXT NOT NULL,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS compose_stacks (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			project_dir TEXT,
			status TEXT NOT NULL DEFAULT 'unknown',
			services_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS compose_services (
			id TEXT PRIMARY KEY,
			stack_id TEXT NOT NULL,
			name TEXT NOT NULL,
			container_id TEXT,
			status TEXT NOT NULL DEFAULT 'unknown',
			ports TEXT,
			UNIQUE(stack_id, name),
			FOREIGN KEY(stack_id) REFERENCES compose_stacks(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS alert_rules (
			id TEXT PRIMARY KEY,
			metric_type TEXT NOT NULL,
			threshold REAL NOT NULL,
			operator TEXT NOT NULL DEFAULT '>=',
			duration_seconds INTEGER NOT NULL DEFAULT 60,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id TEXT PRIMARY KEY,
			rule_id TEXT,
			severity TEXT NOT NULL DEFAULT 'warning',
			message TEXT NOT NULL,
			acknowledged INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(rule_id) REFERENCES alert_rules(id) ON DELETE SET NULL
		);`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("Migration failed on query: %q\nError: %v", q, err)
		}
	}
}

func seedAdmin() {
	adminEmail := os.Getenv("DEFAULT_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@quickpulse.local"
	}
	adminPassword := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin" // match Svelte default
	}

	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", adminEmail).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check if admin user exists: %v", err)
	}

	if !exists {
		hashed, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash admin password: %v", err)
		}

		uid := uuid.New().String()
		_, err = DB.Exec(
			"INSERT INTO users (id, email, hashed_password, role, is_active) VALUES (?, ?, ?, 'admin', 1)",
			uid, adminEmail, string(hashed),
		)
		if err != nil {
			log.Fatalf("Failed to seed admin user: %v", err)
		}
		fmt.Printf("Created admin user: %s (Password: %s)\n", adminEmail, adminPassword)
	}
}
