package astilog

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"path/filepath"
)

type sourceHook struct{}

func (h *sourceHook) Fire(e *logrus.Entry) error {
	// Skip logrus and asticode callers
	i := 0
	_, file, line, ok := runtime.Caller(i)
	for ok && (strings.Contains(file, "/go-astilog/") || strings.Contains(file, "/logrus/")) {
		i++
		_, file, line, ok = runtime.Caller(i)
	}

	// Process file
	if !ok {
		file = "<???>"
		line = 1
	} else {
		file = filepath.Base(file)
	}
	e.Data["source"] = fmt.Sprintf("%s:%d", file, line)
	return nil
}

func (h *sourceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
