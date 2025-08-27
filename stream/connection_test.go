package stream

import (
	"testing"

	"github.com/configcat/configcat-proxy/model"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	conn := newConnection(42)
	pl := &model.ResponsePayload{}
	go func() {
		conn.receive <- pl
	}()
	rec := <-conn.Receive()
	assert.Equal(t, pl, rec)
	assert.Equal(t, uint64(42), conn.discriminator)
}
