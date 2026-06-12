// Package config loads runtime configuration via Viper (env vars + optional
// config file) into a typed Config.
package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config is the resolved application configuration.
type Config struct {
	Env  string
	Port string

	DatabasePath string

	JWTSecret string

	AdminEmail    string
	AdminPassword string

	AllowRegistration bool
}

// Load reads configuration via Viper: defaults, then environment variables,
// then an optional config.yaml. Environment names match the historical keys
// (PORT, SQLITE_DB_PATH, JWT_SECRET_KEY, ...).
func Load() Config {
	v := viper.New()

	v.SetDefault("environment", "development")
	v.SetDefault("port", "8000")
	v.SetDefault("sqlite_db_path", "quickpulse.db")
	v.SetDefault("jwt_secret_key", "change-me-in-production")
	v.SetDefault("default_admin_email", "admin@quickpulse.local")
	v.SetDefault("default_admin_password", "admin")
	v.SetDefault("allow_registration", false)

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/quickpulse")
	_ = v.ReadInConfig() // optional; ignore not-found

	return Config{
		Env:               v.GetString("environment"),
		Port:              v.GetString("port"),
		DatabasePath:      v.GetString("sqlite_db_path"),
		JWTSecret:         v.GetString("jwt_secret_key"),
		AdminEmail:        v.GetString("default_admin_email"),
		AdminPassword:     v.GetString("default_admin_password"),
		AllowRegistration: v.GetBool("allow_registration"),
	}
}
