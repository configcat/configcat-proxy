package stream

import (
	"github.com/configcat/configcat-proxy/model"
)

type Connection struct {
	receive    chan *model.ResponsePayload
	extraAttrs string
}

func newConnection(extraAttrs string) *Connection {
	return &Connection{
		receive:    make(chan *model.ResponsePayload, 64),
		extraAttrs: extraAttrs,
	}
}

func (conn *Connection) Receive() <-chan *model.ResponsePayload {
	return conn.receive
}
