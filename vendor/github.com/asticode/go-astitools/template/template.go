package astitemplate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ParseDirectory parses a directory recursively
func ParseDirectory(i, ext string) (t *template.Template, err error) {
	// Parse templates
	i = filepath.Clean(i)
	t = template.New("Root")
	return t, filepath.Walk(i, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = e
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Check extension
		if ext != "" && filepath.Ext(path) != ext {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			return
		}

		// Parse template
		var c = t.New(filepath.ToSlash(strings.TrimPrefix(path, i)))
		if _, err = c.Parse(string(b)); err != nil {
			return fmt.Errorf("%s while parsing template %s", err, path)
		}
		return
	})
}
