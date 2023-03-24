package stream

import (
	"github.com/configcat/configcat-proxy/model"
)

type Connection struct {
	receive       chan *model.ResponsePayload
	discriminator uint64
}

func newConnection(discriminator uint64) *Connection {
	return &Connection{
		receive:       make(chan *model.ResponsePayload, 64),
		discriminator: discriminator,
	}
}

func (conn *Connection) Receive() <-chan *model.ResponsePayload {
	return conn.receive
}
