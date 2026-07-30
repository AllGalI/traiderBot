package myLogger

import (
	"io"
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() *zap.Logger {
	level := zap.InfoLevel

	if os.Getenv("LOG_LEVEL") == "debug" {
		level = zap.DebugLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Encoding:         "console", // или "json" для прода
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			MessageKey:     "msg",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

type FileLogger struct {
	logger *log.Logger
	file   *os.File
}

func NewFileLogger(filePath string) (*FileLogger, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	mw := io.MultiWriter(os.Stdout, f)
	logger := log.New(mw, "", log.LstdFlags|log.Lshortfile)

	return &FileLogger{
		logger: logger,
		file:   f,
	}, nil
}

func (fl *FileLogger) Info(data any) {
	fl.logger.Printf("%v", data)
}

func (fl *FileLogger) Close() error {
	return fl.file.Close()
}