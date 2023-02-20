package log

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("logger debug", func(t *testing.T) {
		var out, err bytes.Buffer
		l := NewLogger(&err, &out, Debug)
		l.Debugf("debug")
		l.Infof("info")
		l.Warnf("warn")
		l.Errorf("error")
		l.Reportf("rep")
		o := out.String()
		assert.Contains(t, o, "[debug] debug")
		assert.Contains(t, o, "[info] info")
		assert.Contains(t, o, "[warning] warn")
		assert.Contains(t, o, "rep")
		assert.Contains(t, err.String(), "[error] error")
	})
	t.Run("logger info", func(t *testing.T) {
		var out, err bytes.Buffer
		l := NewLogger(&err, &out, Info)
		l.Debugf("debug")
		l.Infof("info")
		l.Warnf("warn")
		l.Errorf("error")
		l.Reportf("rep")
		o := out.String()
		assert.NotContains(t, o, "[debug] debug")
		assert.Contains(t, o, "[info] info")
		assert.Contains(t, o, "[warning] warn")
		assert.Contains(t, o, "rep")
		assert.Contains(t, err.String(), "[error] error")
	})
	t.Run("logger warn", func(t *testing.T) {
		var out, err bytes.Buffer
		l := NewLogger(&err, &out, Warn)
		l.Debugf("debug")
		l.Infof("info")
		l.Warnf("warn")
		l.Errorf("error")
		l.Reportf("rep")
		o := out.String()
		assert.NotContains(t, o, "[debug] debug")
		assert.NotContains(t, o, "[info] info")
		assert.Contains(t, o, "[warning] warn")
		assert.Contains(t, o, "rep")
		assert.Contains(t, err.String(), "[error] error")
	})
	t.Run("logger error", func(t *testing.T) {
		var out, err bytes.Buffer
		l := NewLogger(&err, &out, Error)
		l.Debugf("debug")
		l.Infof("info")
		l.Warnf("warn")
		l.Errorf("error")
		l.Reportf("rep")
		o := out.String()
		assert.NotContains(t, o, "[debug] debug")
		assert.NotContains(t, o, "[info] info")
		assert.NotContains(t, o, "[warning] warn")
		assert.Contains(t, o, "rep")
		assert.Contains(t, err.String(), "[error] error")
	})
	t.Run("logger with level", func(t *testing.T) {
		var out, err bytes.Buffer
		l := NewLogger(&err, &out, Error)
		l = l.WithLevel(Debug)
		l.Debugf("debug")
		l.Infof("info")
		l.Warnf("warn")
		l.Errorf("error")
		l.Reportf("rep")
		o := out.String()
		assert.Contains(t, o, "[debug] debug")
		assert.Contains(t, o, "[info] info")
		assert.Contains(t, o, "[warning] warn")
		assert.Contains(t, o, "rep")
		assert.Contains(t, err.String(), "[error] error")
	})
	t.Run("logger with prefix", func(t *testing.T) {
		var out, err bytes.Buffer
		l := NewLogger(&err, &out, Debug)
		l = l.WithPrefix("pref1").WithPrefix("pref2")
		l.Errorf("error")
		assert.Contains(t, err.String(), "[error] <pref1/pref2> error")
	})
	t.Run("null logger", func(t *testing.T) {
		l := NewNullLogger()
		l.Debugf("debug")
		l.Infof("info")
		l.Warnf("warn")
		l.Errorf("error")
		l.Reportf("rep")
	})
}
