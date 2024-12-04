package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger *zap.Logger
	once   sync.Once
)

// Initialize initializes the logger instance.
// It uses sync.Once to ensure the logger is only initialized once.
func Initialize() {
	once.Do(func() {
		// Lumberjack configuration for log rotation
		lumberjackLogger := &lumberjack.Logger{
			Filename:   "./logs/app.log",
			MaxSize:    5,    // Max megabytes before log rotation
			MaxBackups: 3,    // Max number of old log files to keep
			MaxAge:     28,   // Max number of days to retain old log files
			Compress:   true, // Compress rotated files
		}

		// Encoder configuration
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Encoders
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

		// Write syncers
		consoleWriter := zapcore.AddSync(os.Stdout)
		fileWriter := zapcore.AddSync(lumberjackLogger)

		// Core
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleWriter, zap.DebugLevel),
			zapcore.NewCore(fileEncoder, fileWriter, zap.InfoLevel),
		)

		Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	})
}

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

func Info(message string, fields ...zap.Field) {
	Initialize()
	Logger.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	Initialize()
	Logger.Debug(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	Initialize()
	Logger.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	Initialize()
	Logger.Error(message, fields...)
}

func Infof(format string, args ...interface{}) {
	Initialize()
	Logger.Sugar().Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	Initialize()
	Logger.Sugar().Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Initialize()
	Logger.Sugar().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Initialize()
	Logger.Sugar().Errorf(format, args...)
}
