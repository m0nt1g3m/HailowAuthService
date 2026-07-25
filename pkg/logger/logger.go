package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

func InitLogger(env string) {

	logDir := "logs"
	logFile := filepath.Join(logDir, "app.log")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Не удалось создать директорию для логов: %v\n", err.Error())
		os.Exit(1)
	}

	encoderConfig := zap.NewDevelopmentEncoderConfig()

	if env == "production" {
		encoderConfig = zap.NewProductionEncoderConfig()
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleConfig := encoderConfig
	consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	fileConfig := encoderConfig
	fileConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	var consoleEncoder, fileEncoder zapcore.Encoder
	if env == "production" {
		consoleEncoder = zapcore.NewJSONEncoder(consoleConfig)
		fileEncoder = zapcore.NewJSONEncoder(fileConfig)
	} else {
		consoleEncoder = zapcore.NewConsoleEncoder(consoleConfig)
		fileEncoder = zapcore.NewConsoleEncoder(fileConfig)
	}
	logFileWriter, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		println("Не удалось открыть файл логов: " + err.Error())
		os.Exit(1)
	}

	level := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	if env == "production" {
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), level)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.Lock(logFileWriter), level)

	coreLogger := zapcore.NewTee(consoleCore, fileCore)

	logger := zap.New(coreLogger, zap.WithCaller(false))

	Log = logger.Sugar()
}
