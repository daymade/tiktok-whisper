package common

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new zap logger with appropriate configuration
func NewLogger(development bool) (*zap.Logger, error) {
	var config zap.Config
	
	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}
	
	return config.Build()
}

// MustNewLogger creates a new logger and panics if it fails
func MustNewLogger(development bool) *zap.Logger {
	logger, err := NewLogger(development)
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	return logger
}