package astilog

import (
	"context"
	"os"
)

// NopLogger returns a nop logger
func NopLogger() Logger {
	return &nop{}
}

// nop is a nop logger
type nop struct{}

func (n nop) Debug(v ...interface{})                                       {}
func (n nop) DebugC(ctx context.Context, v ...interface{})                 {}
func (n nop) DebugCf(ctx context.Context, format string, v ...interface{}) {}
func (n nop) Debugf(format string, v ...interface{})                       {}
func (n nop) Info(v ...interface{})                                        {}
func (n nop) InfoC(ctx context.Context, v ...interface{})                  {}
func (n nop) InfoCf(ctx context.Context, format string, v ...interface{})  {}
func (n nop) Infof(format string, v ...interface{})                        {}
func (n nop) Warn(v ...interface{})                                        {}
func (n nop) WarnC(ctx context.Context, v ...interface{})                  {}
func (n nop) WarnCf(ctx context.Context, format string, v ...interface{})  {}
func (n nop) Warnf(format string, v ...interface{})                        {}
func (n nop) Error(v ...interface{})                                       {}
func (n nop) ErrorC(ctx context.Context, v ...interface{})                 {}
func (n nop) ErrorCf(ctx context.Context, format string, v ...interface{}) {}
func (n nop) Errorf(format string, v ...interface{})                       {}
func (n nop) Fatal(v ...interface{})                                       { os.Exit(1) }
func (n nop) FatalC(ctx context.Context, v ...interface{})                 { os.Exit(1) }
func (n nop) FatalCf(ctx context.Context, format string, v ...interface{}) { os.Exit(1) }
func (n nop) Fatalf(format string, v ...interface{})                       { os.Exit(1) }
func (n nop) WithField(k string, v interface{})                            {}
func (n nop) WithFields(fs Fields)                                         {}
