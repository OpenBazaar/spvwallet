package astilog

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Fields represents logger fields
type Fields map[string]interface{}

type fields struct {
	fs Fields
	m  *sync.Mutex
}

func newFields() *fields {
	return &fields{
		fs: make(Fields),
		m:  &sync.Mutex{},
	}
}

func (fs *fields) Fire(e *logrus.Entry) error {
	fs.m.Lock()
	defer fs.m.Unlock()
	for k, v := range fs.fs {
		e.Data[k] = v
	}
	return nil
}

func (fs *fields) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (fs *fields) set(k string, v interface{}) {
	fs.m.Lock()
	defer fs.m.Unlock()
	fs.fs[k] = v
}

func (fs *fields) setMultiple(i Fields) {
	fs.m.Lock()
	defer fs.m.Unlock()
	for k, v := range i {
		fs.fs[k] = v
	}
}
