package stream

import (
	"github.com/configcat/configcat-proxy/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnection(t *testing.T) {
	cl := make(chan *connection)
	conn := newConnection(cl, "test")
	pl := &model.ResponsePayload{}
	go func() {
		conn.receive <- pl
	}()
	rec := <-conn.Receive()
	assert.Equal(t, pl, rec)

	go func() {
		conn.Close()
	}()
	c := <-cl
	rec = <-conn.Receive()
	assert.Same(t, conn, c)
}
