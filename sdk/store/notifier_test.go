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
	case <-n.Context().Done():
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
	case <-n.Context().Done():
	case <-n.Modified():
		assert.Fail(t, "close expected")
	}
}
