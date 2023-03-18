package stream

import (
	"github.com/configcat/configcat-proxy/model"
)

type Connection struct {
	receive       chan *model.ResponsePayload
	discriminator string
}

func newConnection(discriminator string) *Connection {
	return &Connection{
		receive:       make(chan *model.ResponsePayload, 64),
		discriminator: discriminator,
	}
}

func (conn *Connection) Receive() <-chan *model.ResponsePayload {
	return conn.receive
}
