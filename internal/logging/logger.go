// Package logging provides centralized logging functionality for the ekssm application
// using structured logging with different severity levels and debug mode support.
package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

func init() {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	logger, _ := config.Build()
	log = logger.Sugar()
}

// GetLogger returns the global logger instance.
func GetLogger() *zap.SugaredLogger {
	return log
}

// Info logs a message at info level.
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof logs a formatted message at info level.
func Infof(template string, args ...interface{}) {
	log.Infof(template, args...)
}

// Debug logs a message at debug level.
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf logs a formatted message at debug level.
func Debugf(template string, args ...interface{}) {
	log.Debugf(template, args...)
}

// Warn logs a message at warn level.
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf logs a formatted message at warn level.
func Warnf(template string, args ...interface{}) {
	log.Warnf(template, args...)
}

// Error logs a message at error level.
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf logs a formatted message at error level.
func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args...)
}

// Fatal logs a message at fatal level and then calls os.Exit(1).
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf logs a formatted message at fatal level and then calls os.Exit(1).
func Fatalf(template string, args ...interface{}) {
	log.Fatalf(template, args...)
}

// Sync ensures all buffered logs are written.
func Sync() {
	log.Sync()
}

// SetDebug enables or disables debug logging mode.
func SetDebug(debug bool) {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	
	logger, _ := config.Build()
	log = logger.Sugar()
}
