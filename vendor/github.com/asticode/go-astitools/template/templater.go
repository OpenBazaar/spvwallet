package astitemplate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"
)

// Templater represents an object capable of storing templates
type Templater struct {
	layouts   []string
	m         sync.Mutex
	templates map[string]*template.Template
}

// NewTemplater creates a new templater
func NewTemplater() *Templater {
	return &Templater{templates: make(map[string]*template.Template)}
}

// AddLayoutsFromDir walks through a dir and add files as layouts
func (t *Templater) AddLayoutsFromDir(dirPath, ext string) (err error) {
	// Get layouts
	if err = filepath.Walk(dirPath, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = errors.Wrapf(e, "astitemplate: walking layouts has an input error for path %s", path)
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

		// Read layout
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = errors.Wrapf(err, "astitemplate: reading %s failed", path)
			return
		}

		// Add layout
		t.AddLayout(string(b))
		return
	}); err != nil {
		err = errors.Wrapf(err, "astitemplate: walking layouts in %s failed", dirPath)
		return
	}
	return
}

// AddTemplatesFromDir walks through a dir and add files as templates
func (t *Templater) AddTemplatesFromDir(dirPath, ext string) (err error) {
	// Loop through templates
	if err = filepath.Walk(dirPath, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = errors.Wrapf(e, "astitemplate: walking templates has an input error for path %s", path)
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
			err = errors.Wrapf(err, "astitemplate: reading template content of %s failed", path)
			return
		}

		// Add template
		// We use ToSlash to homogenize Windows path
		if err = t.AddTemplate(filepath.ToSlash(strings.TrimPrefix(path, dirPath)), string(b)); err != nil {
			err = errors.Wrap(err, "astitemplate: adding template failed")
			return
		}
		return
	}); err != nil {
		err = errors.Wrapf(err, "astitemplate: walking templates in %s failed", dirPath)
		return
	}
	return
}

// AddLayout adds a new layout
func (t *Templater) AddLayout(c string) {
	t.layouts = append(t.layouts, c)
}

// AddTemplate adds a new template
func (t *Templater) AddTemplate(path, content string) (err error) {
	// Parse
	var tpl *template.Template
	if tpl, err = t.Parse(content); err != nil {
		err = errors.Wrapf(err, "astitemplate: parsing template for path %s failed", path)
		return
	}

	// Add template
	t.m.Lock()
	t.templates[path] = tpl
	t.m.Unlock()
	return
}

// DelTemplate deletes a template
func (t *Templater) DelTemplate(path string) {
	t.m.Lock()
	defer t.m.Unlock()
	delete(t.templates, path)
}

// Template retrieves a templates
func (t *Templater) Template(path string) (tpl *template.Template, ok bool) {
	t.m.Lock()
	defer t.m.Unlock()
	tpl, ok = t.templates[path]
	return
}

// Parse parses the content of a template
func (t *Templater) Parse(content string) (o *template.Template, err error) {
	// Parse content
	o = template.New("root")
	if o, err = o.Parse(content); err != nil {
		err = errors.Wrap(err, "astitemplate: parsing template content failed")
		return
	}

	// Parse layouts
	for idx, l := range t.layouts {
		if o, err = o.Parse(l); err != nil {
			err = errors.Wrapf(err, "astitemplate: parsing layout #%d failed", idx+1)
			return
		}
	}
	return
}
