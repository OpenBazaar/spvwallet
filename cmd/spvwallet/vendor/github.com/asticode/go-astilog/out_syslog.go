// +build !windows

package astilog

import (
	"io"
	"log/syslog"
	"os"

	"github.com/pkg/errors"
)

// DefaultOutput is the default output
func DefaultOut(c Configuration) (w io.Writer) {
	if isTerminal(os.Stdout) {
		return os.Stdout
	}
	var err error
	if w, err = syslog.New(syslog.LOG_INFO|syslog.LOG_USER, c.AppName); err != nil {
		panic(errors.Wrap(err, "new syslog failed"))
	}
	return
}
