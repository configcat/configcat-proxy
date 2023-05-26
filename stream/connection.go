package stream

type Connection struct {
	receive       chan interface{}
	discriminator uint64
}

func newConnection(discriminator uint64) *Connection {
	return &Connection{
		receive:       make(chan interface{}, 64),
		discriminator: discriminator,
	}
}

func (conn *Connection) Receive() <-chan interface{} {
	return conn.receive
}
