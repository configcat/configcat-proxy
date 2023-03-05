package stream

import (
	"github.com/configcat/configcat-proxy/model"
)

type Connection interface {
	Receive() <-chan *model.ResponsePayload
	Publish(payload *model.ResponsePayload)
	GetExtraAttrs() string
}

type connection struct {
	receive    chan *model.ResponsePayload
	extraAttrs string
}

func newConnection(extraAttrs string) Connection {
	return &connection{
		receive:    make(chan *model.ResponsePayload, 64),
		extraAttrs: extraAttrs,
	}
}

func (conn *connection) Publish(payload *model.ResponsePayload) {
	conn.receive <- payload
}

func (conn *connection) Receive() <-chan *model.ResponsePayload {
	return conn.receive
}

func (conn *connection) GetExtraAttrs() string {
	return conn.extraAttrs
}
