package astilog

import (
	"io"
	"log"
	"os"

	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Logrus represents a logrus logger
type Logrus struct {
	c  Configuration
	fs *fields
	l  *logrus.Logger
}

func newLogrus(c Configuration) (l *Logrus) {
	// Init
	l = &Logrus{
		c:  c,
		fs: newFields(),
		l:  logrus.New(),
	}

	// Out
	var out string
	l.l.Out, out = logrusOut(c)

	// Formatter
	l.l.Formatter = logrusFormatter(c, out)

	// Level
	l.l.Level = logrusLevel(c)

	// Hooks
	l.l.AddHook(l.fs)
	l.l.AddHook(&sourceHook{})

	// Default fields
	if c.AppName != "" {
		l.WithFields(Fields{"app_name": c.AppName})
	}
	return
}

func logrusOut(c Configuration) (w io.Writer, out string) {
	switch c.Out {
	case OutStdOut:
		return stdOut(), c.Out
	case OutSyslog:
		return syslogOut(c), c.Out
	default:
		if isInteractive() {
			w = stdOut()
			out = OutStdOut
		} else {
			w = syslogOut(c)
			out = OutSyslog
		}
		if len(c.Filename) > 0 {
			f, err := os.OpenFile(c.Filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				log.Println(errors.Wrapf(err, "astilog: creating %s failed", c.Filename))
			} else {
				w = f
				out = OutFile
			}
		}
		return
	}
}

func logrusFormatter(c Configuration, out string) logrus.Formatter {
	switch c.Format {
	case FormatJSON:
		return logrusJSONFormatter(c)
	case FormatText:
		return logrusTextFormatter(c, out)
	default:
		switch out {
		case OutFile, OutStdOut:
			return logrusTextFormatter(c, out)
		default:
			return logrusJSONFormatter(c)
		}
	}
}

func logrusJSONFormatter(c Configuration) logrus.Formatter {
	f := &logrus.JSONFormatter{
		FieldMap:        make(logrus.FieldMap),
		TimestampFormat: c.TimestampFormat,
	}
	if len(c.MessageKey) > 0 {
		f.FieldMap[logrus.FieldKeyMsg] = c.MessageKey
	}
	return f
}

func logrusTextFormatter(c Configuration, out string) logrus.Formatter {
	return &logrus.TextFormatter{
		DisableColors:    c.DisableColors || out == OutFile,
		DisableTimestamp: c.DisableTimestamp,
		ForceColors:      !c.DisableColors && out != OutFile,
		FullTimestamp:    c.FullTimestamp,
		TimestampFormat:  c.TimestampFormat,
	}
}

func logrusLevel(c Configuration) logrus.Level {
	if c.Verbose {
		return logrus.DebugLevel
	}
	return logrus.InfoLevel
}

func logrusFieldsFromContext(ctx context.Context) (fs logrus.Fields) {
	if cfs := fieldsFromContext(ctx); cfs != nil {
		return logrus.Fields(cfs)
	}
	return nil
}

func (l *Logrus) Debug(v ...interface{}) { l.l.Debug(v...) }

func (l *Logrus) DebugC(ctx context.Context, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Debug(v...)
}

func (l *Logrus) DebugCf(ctx context.Context, format string, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Debugf(format, v...)
}

func (l *Logrus) Debugf(format string, v ...interface{}) { l.l.Debugf(format, v...) }

func (l *Logrus) Info(v ...interface{}) { l.l.Info(v...) }

func (l *Logrus) InfoC(ctx context.Context, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Info(v...)
}

func (l *Logrus) InfoCf(ctx context.Context, format string, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Infof(format, v...)
}

func (l *Logrus) Infof(format string, v ...interface{}) { l.l.Infof(format, v...) }

func (l *Logrus) Warn(v ...interface{}) { l.l.Warn(v...) }

func (l *Logrus) WarnC(ctx context.Context, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Warn(v...)
}

func (l *Logrus) WarnCf(ctx context.Context, format string, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Warnf(format, v...)
}

func (l *Logrus) Warnf(format string, v ...interface{}) { l.l.Warnf(format, v...) }

func (l *Logrus) Error(v ...interface{}) { l.l.Error(v...) }

func (l *Logrus) ErrorC(ctx context.Context, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Error(v...)
}

func (l *Logrus) ErrorCf(ctx context.Context, format string, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Errorf(format, v...)
}

func (l *Logrus) Errorf(format string, v ...interface{}) { l.l.Errorf(format, v...) }

func (l *Logrus) Fatal(v ...interface{}) { l.l.Fatal(v...) }

func (l *Logrus) FatalC(ctx context.Context, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Fatal(v...)
}

func (l *Logrus) FatalCf(ctx context.Context, format string, v ...interface{}) {
	l.l.WithFields(logrusFieldsFromContext(ctx)).Fatalf(format, v...)
}

func (l *Logrus) Fatalf(format string, v ...interface{}) { l.l.Fatalf(format, v...) }

func (l *Logrus) WithField(k string, v interface{}) { l.fs.set(k, v) }

func (l *Logrus) WithFields(fs Fields) { l.fs.setMultiple(fs) }

func (l *Logrus) Write(b []byte) (n int, err error) {
	l.l.Debug(string(b))
	n = len(b)
	return
}
