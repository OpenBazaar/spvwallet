package astilog

import (
	"os"

	"context"

	"golang.org/x/crypto/ssh/terminal"
)

// Logger represents a logger
type Logger interface {
	Debug(v ...interface{})
	DebugC(ctx context.Context, v ...interface{})
	DebugCf(ctx context.Context, format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	InfoC(ctx context.Context, v ...interface{})
	InfoCf(ctx context.Context, format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	WarnC(ctx context.Context, v ...interface{})
	WarnCf(ctx context.Context, format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Error(v ...interface{})
	ErrorC(ctx context.Context, v ...interface{})
	ErrorCf(ctx context.Context, format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	FatalC(ctx context.Context, v ...interface{})
	FatalCf(ctx context.Context, format string, v ...interface{})
	Fatalf(format string, v ...interface{})
	WithField(k string, v interface{})
	WithFields(fs Fields)
}

// LoggerSetter represents a logger setter
type LoggerSetter interface {
	SetLogger(l Logger)
}

// New creates a new Logger
func New(c Configuration) Logger {
	return newLogrus(c)
}

func isInteractive() bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}
