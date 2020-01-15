package astilog

import (
	"bytes"
	"regexp"
)

// Vars
var (
	bytesEmpty  = []byte("")
	bytesEOL    = []byte("\n")
	regexpColor = regexp.MustCompile("\\[[\\d]+m")
)

// Writer represents an object capable of writing to the logger
type Writer struct {
	buffer *bytes.Buffer
	fs     []WriterFunc
}

// WriterFunc represents a writer func
type WriterFunc func(text string)

// LoggerFuncToWriterFunc converts a logger func to a writer func
func LoggerFuncToWriterFunc(fn func(args ...interface{})) WriterFunc {
	return func(text string) {
		fn(text)
	}
}

// NewWriter creates a new writer
func NewWriter(fs ...WriterFunc) *Writer {
	return &Writer{
		buffer: &bytes.Buffer{},
		fs:     fs,
	}
}

// Close closes the writer
func (w *Writer) Close() error {
	if w.buffer.Len() > 0 {
		w.write(w.buffer.Bytes())
	}
	return nil
}

// Write implements the io.Writer interface
func (w *Writer) Write(i []byte) (n int, err error) {
	// Update n to avoid broken pipe error
	defer func() {
		n = len(i)
	}()

	// No EOL in the log, write in buffer
	if bytes.Index(i, bytesEOL) == -1 {
		w.buffer.Write(i)
		return
	}

	// Loop in items split by EOL
	var items = bytes.Split(i, bytesEOL)
	for i := 0; i < len(items)-1; i++ {
		// If first item, add the buffer
		if i == 0 {
			items[i] = append(w.buffer.Bytes(), items[i]...)
			w.buffer.Reset()
		}

		// Log
		w.write(items[i])
	}

	// Add remaining to buffer
	w.buffer.Write(items[len(items)-1])
	return
}

func (w *Writer) write(i []byte) {
	// Sanitize text
	text := bytes.Trim(bytes.TrimSpace(regexpColor.ReplaceAll(i, bytesEmpty)), "\x00")

	// Empty text
	if len(text) == 0 {
		return
	}

	// Write
	for _, fn := range w.fs {
		fn(string(text))
	}
}
