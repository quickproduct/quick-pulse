// Package logging configures a structured zap logger shared across the backend.
package logging

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a *zap.Logger. When env is "development" it uses a human-friendly
// console encoder; otherwise it emits JSON to stdout.
func New(env string) *zap.Logger {
	var cfg zap.Config
	if strings.EqualFold(env, "development") || strings.EqualFold(env, "dev") {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "ts"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	logger, err := cfg.Build()
	if err != nil {
		logger = zap.NewExample()
	}
	return logger
}
