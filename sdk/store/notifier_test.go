package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNotifier_Modified(t *testing.T) {
	n := NewNotifier()
	go func() {
		n.Notify()
	}()
	select {
	case <-n.Closed():
		assert.Fail(t, "modified expected")
	case <-n.Modified():
	}
}

func TestNotifier_Closed(t *testing.T) {
	n := NewNotifier()
	go func() {
		n.Close()
	}()
	select {
	case <-n.Closed():
	case <-n.Modified():
		assert.Fail(t, "close expected")
	}
}
