// +build windows

package astilog

import (
	"io"

	colorable "github.com/mattn/go-colorable"
)

// DefaultOut is the default out
func DefaultOut(c Configuration) io.Writer {
	return colorable.NewColorableStdout()
}
