package astilog

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/rs/xlog"
	"github.com/sirupsen/logrus"
)

// NopLogger returns a nop logger
func NopLogger() Logger {
	return xlog.NopLogger
}

// Logger represents a logger
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

// LoggerSetter represents a logger setter
type LoggerSetter interface {
	SetLogger(l Logger)
}

// New creates a new Logger
func New(c Configuration) Logger {
	// Init
	var l = logrus.New()
	l.WithField("app_name", c.AppName)
	l.Formatter = &logrus.TextFormatter{ForceColors: true}
	l.Level = logrus.InfoLevel
	l.Out = DefaultOut(c)

	// Formatter
	if !isTerminal(os.Stdout) {
		l.Formatter = &logrus.JSONFormatter{}
	}

	// Verbose
	if c.Verbose {
		l.Level = logrus.DebugLevel
	}
	return l
}

func isTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
