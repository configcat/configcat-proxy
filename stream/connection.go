package stream

import (
	"github.com/configcat/configcat-proxy/model"
)

type Connection interface {
	Receive() <-chan *model.ResponsePayload
	Close()
}

type connection struct {
	receive       chan *model.ResponsePayload
	closeNotifier chan *connection
	discriminator string
}

func newConnection(closeNotifier chan *connection, discriminator string) *connection {
	return &connection{
		receive:       make(chan *model.ResponsePayload, 128),
		closeNotifier: closeNotifier,
		discriminator: discriminator,
	}
}

func (conn *connection) Receive() <-chan *model.ResponsePayload {
	return conn.receive
}

func (conn *connection) Close() {
	conn.closeNotifier <- conn
	close(conn.receive)
}
