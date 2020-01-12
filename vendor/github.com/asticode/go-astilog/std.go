package astilog

import (
	"context"
	"log"
	"io"
)

// Global logger
var gb = NopLogger()

// FlagInit initializes the package based on flags
func FlagInit() {
	SetLogger(New(FlagConfig()))
}

// SetLogger sets the global logger
func SetLogger(l Logger) {
	gb = l
	if w, ok := l.(io.Writer); ok {
		log.SetFlags(0)
		log.SetOutput(w)
	}
}

// SetDefaultLogger sets the default logger
func SetDefaultLogger() {
	SetLogger(New(Configuration{Verbose: true}))
}

// GetLogger returns the global logger
func GetLogger() Logger {
	return gb
}

// Global logger shortcuts
func Debug(v ...interface{})                                       { gb.Debug(v...) }
func DebugC(ctx context.Context, v ...interface{})                 { gb.DebugC(ctx, v...) }
func DebugCf(ctx context.Context, format string, v ...interface{}) { gb.DebugCf(ctx, format, v...) }
func Debugf(format string, v ...interface{})                       { gb.Debugf(format, v...) }
func Info(v ...interface{})                                        { gb.Info(v...) }
func InfoC(ctx context.Context, v ...interface{})                  { gb.InfoC(ctx, v...) }
func InfoCf(ctx context.Context, format string, v ...interface{})  { gb.InfoCf(ctx, format, v...) }
func Infof(format string, v ...interface{})                        { gb.Infof(format, v...) }
func Warn(v ...interface{})                                        { gb.Warn(v...) }
func WarnC(ctx context.Context, v ...interface{})                  { gb.WarnC(ctx, v...) }
func WarnCf(ctx context.Context, format string, v ...interface{})  { gb.WarnCf(ctx, format, v...) }
func Warnf(format string, v ...interface{})                        { gb.Warnf(format, v...) }
func Error(v ...interface{})                                       { gb.Error(v...) }
func ErrorC(ctx context.Context, v ...interface{})                 { gb.ErrorC(ctx, v...) }
func ErrorCf(ctx context.Context, format string, v ...interface{}) { gb.ErrorCf(ctx, format, v...) }
func Errorf(format string, v ...interface{})                       { gb.Errorf(format, v...) }
func Fatal(v ...interface{})                                       { gb.Fatal(v...) }
func FatalC(ctx context.Context, v ...interface{})                 { gb.FatalC(ctx, v...) }
func FatalCf(ctx context.Context, format string, v ...interface{}) { gb.FatalCf(ctx, format, v...) }
func Fatalf(format string, v ...interface{})                       { gb.Fatalf(format, v...) }
func WithField(k, v string)                                        { gb.WithField(k, v) }
func WithFields(fs Fields)                                         { gb.WithFields(fs) }
