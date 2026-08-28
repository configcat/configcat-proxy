package stream

import (
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
)

const AllFlagsDiscriminator = "[ALL]"
const NotifyOnlyDiscriminator = "[NOTIFY]"

type channel interface {
	Notify(sdkClient sdk.Client, key string) int
	AddConnection(conn *Connection)
	RemoveConnection(conn *Connection)
	LastPayload() interface{}
	IsEmpty() bool
}

type connectionHolder struct {
	connections []*Connection
	user        model.UserAttrs
}

type singleFlagChannel struct {
	lastPayload *model.ResponsePayload

	connectionHolder
}

type allFlagsChannel struct {
	lastPayload map[string]*model.ResponsePayload

	connectionHolder
}

type notifyOnlyChannel struct {
	connectionHolder
}

func createChannel(established *connEstablished, sdkClient sdk.Client) channel {
	if established.key == NotifyOnlyDiscriminator {
		return &notifyOnlyChannel{}
	} else if established.key == AllFlagsDiscriminator {
		values := sdkClient.EvalAll(established.user)
		payloads := make(map[string]*model.ResponsePayload)
		for key, val := range values {
			payload := model.PayloadFromEvalData(&val)
			payloads[key] = &payload
		}
		return &allFlagsChannel{connectionHolder: connectionHolder{user: established.user}, lastPayload: payloads}
	}
	val := sdkClient.Eval(established.key, established.user)
	payload := model.PayloadFromEvalData(&val)
	return &singleFlagChannel{connectionHolder: connectionHolder{user: established.user}, lastPayload: &payload}
}

func (sf *singleFlagChannel) LastPayload() interface{} {
	return sf.lastPayload
}

func (af *allFlagsChannel) LastPayload() interface{} {
	return af.lastPayload
}

func (c *notifyOnlyChannel) LastPayload() interface{} {
	return nil
}

func (sf *singleFlagChannel) Notify(sdkClient sdk.Client, key string) int {
	val := sdkClient.Eval(key, sf.user)
	if val.Error != nil {
		return 0
	}
	if sf.lastPayload == nil || val.Value != sf.lastPayload.Value {
		payload := model.PayloadFromEvalData(&val)
		sf.lastPayload = &payload
		return sf.notify(&payload)
	}
	return 0
}

func (af *allFlagsChannel) Notify(sdkClient sdk.Client, _ string) int {
	values := sdkClient.EvalAll(af.user)
	if values == nil || len(values) == 0 {
		return 0
	}
	final := make(map[string]*model.ResponsePayload)
	for key, val := range values {
		payload := model.PayloadFromEvalData(&val)
		final[key] = &payload
	}
	af.lastPayload = final
	if len(final) != 0 {
		return af.notify(final)
	}
	return 0
}

func (c *notifyOnlyChannel) Notify(_ sdk.Client, _ string) int {
	return c.notify(nil)
}

func (c *connectionHolder) notify(msg interface{}) int {
	sent := 0
	for _, conn := range c.connections {
		sent++
		conn.receive <- msg
	}
	return sent
}

func (c *connectionHolder) AddConnection(conn *Connection) {
	c.connections = append(c.connections, conn)
}

func (c *connectionHolder) RemoveConnection(conn *Connection) {
	index := -1
	for i := range c.connections {
		if c.connections[i] == conn {
			index = i
			break
		}
	}
	if index != -1 {
		c.connections[index] = nil
		c.connections = append(c.connections[:index], c.connections[index+1:]...)
	}
}

func (c *connectionHolder) IsEmpty() bool {
	return len(c.connections) == 0
}
