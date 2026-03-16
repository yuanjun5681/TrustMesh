package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	parsedLevel := zapcore.InfoLevel
	if err := parsedLevel.Set(strings.ToLower(level)); err == nil {
		cfg.Level = zap.NewAtomicLevelAt(parsedLevel)
	}

	return cfg.Build()
}
