package stream

import (
	"github.com/configcat/configcat-proxy/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnection(t *testing.T) {
	conn := newConnection("test")
	pl := &model.ResponsePayload{}
	go func() {
		conn.receive <- pl
	}()
	rec := <-conn.Receive()
	assert.Equal(t, pl, rec)
	assert.Equal(t, "test", conn.extraAttrs)
}
