// Package db owns the database connection and schema. GORM (via the pure-Go
// glebarez/modernc driver, CGO-free) owns the *gorm.DB handle and drives the
// schema through AutoMigrate. The underlying *sql.DB is exposed as DB so the
// analytical/metrics/log queries elsewhere keep running on the same connection.
package db

import (
	"fmt"
	"database/sql"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Gorm is the GORM handle; DB is the shared *sql.DB drawn from the same pool.
var (
	Gorm *gorm.DB
	DB   *sql.DB
)

// Config carries the settings InitDB needs (resolved by the config package).
type Config struct {
	Path          string
	AdminEmail    string
	AdminPassword string
}

// InitDB opens SQLite via GORM/glebarez, applies pragmas, runs AutoMigrate plus
// the raw objects GORM cannot express (the time-series host_metrics table and
// its indexes), and seeds the default admin user.
func InitDB(cfg Config, logger *zap.Logger) error {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)&_pragma=auto_vacuum(2)&_pragma=cache_size(-4000)&_pragma=mmap_size(0)",
		cfg.Path,
	)

	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	Gorm = gdb

	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("acquire sql.DB: %w", err)
	}
	// SQLite is single-writer; serialise writes through one connection.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	DB = sqlDB

	if err := gdb.AutoMigrate(entityModels()...); err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}

	// Time-series + extra index objects AutoMigrate cannot express.
	rawObjects := []string{
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
		)`,
		`CREATE INDEX IF NOT EXISTS idx_host_metrics_time ON host_metrics(time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_host_metrics_host_id_time ON host_metrics(host_id, time DESC)`,
	}
	for _, q := range rawObjects {
		if err := gdb.Exec(q).Error; err != nil {
			return fmt.Errorf("create raw object: %w", err)
		}
	}

	if err := seedAdmin(cfg, logger); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	return nil
}

// seedAdmin creates the default admin user if it does not yet exist.
func seedAdmin(cfg Config, logger *zap.Logger) error {
	var count int64
	if err := Gorm.Model(&UserEntity{}).Where("email = ?", cfg.AdminEmail).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admin := UserEntity{
		ID:             uuid.New().String(),
		Email:          cfg.AdminEmail,
		HashedPassword: string(hashed),
		Role:           "admin",
		IsActive:       true,
	}
	if err := Gorm.Create(&admin).Error; err != nil {
		return err
	}
	logger.Info("created default admin user", zap.String("email", cfg.AdminEmail))
	return nil
}
