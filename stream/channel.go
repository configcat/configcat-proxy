package stream

import (
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
)

const AllFlagsDiscriminator = "[ALL]"

type channel interface {
	Notify(sdkClient sdk.Client, key string) int
	AddConnection(conn *Connection)
	RemoveConnection(conn *Connection)
	LastPayload() interface{}
	IsEmpty() bool
}

type connectionHolder struct {
	connections []*Connection
	user        sdk.UserAttrs
}

type singleFlagChannel struct {
	lastPayload *model.ResponsePayload

	connectionHolder
}

type allFlagsChannel struct {
	lastPayload map[string]*model.ResponsePayload

	connectionHolder
}

func createChannel(established *connEstablished, sdkClient sdk.Client) channel {
	if established.key == AllFlagsDiscriminator {
		values := sdkClient.EvalAll(established.user)
		payloads := make(map[string]*model.ResponsePayload)
		for key, val := range values {
			payload := model.PayloadFromEvalData(&val)
			payloads[key] = &payload
		}
		return &allFlagsChannel{connectionHolder: connectionHolder{user: established.user}, lastPayload: payloads}
	} else {
		val, _ := sdkClient.Eval(established.key, established.user)
		payload := model.PayloadFromEvalData(&val)
		return &singleFlagChannel{connectionHolder: connectionHolder{user: established.user}, lastPayload: &payload}
	}
}

func (sf *singleFlagChannel) LastPayload() interface{} {
	return sf.lastPayload
}

func (af *allFlagsChannel) LastPayload() interface{} {
	return af.lastPayload
}

func (sf *singleFlagChannel) Notify(sdkClient sdk.Client, key string) int {
	sent := 0
	val, err := sdkClient.Eval(key, sf.user)
	if err != nil {
		return 0
	}
	if sf.lastPayload == nil || val.Value != sf.lastPayload.Value {
		payload := model.PayloadFromEvalData(&val)
		sf.lastPayload = &payload
		for _, conn := range sf.connections {
			sent++
			conn.receive <- &payload
		}
	}
	return sent
}

func (af *allFlagsChannel) Notify(sdkClient sdk.Client, _ string) int {
	sent := 0
	values := sdkClient.EvalAll(af.user)
	if values == nil || len(values) == 0 {
		return 0
	}
	final := make(map[string]*model.ResponsePayload)
	for key, val := range values {
		lp, ok := af.lastPayload[key]
		if !ok || val.Value != lp.Value {
			payload := model.PayloadFromEvalData(&val)
			af.lastPayload[key] = &payload
			final[key] = &payload
		}
	}
	if len(final) != 0 {
		for _, conn := range af.connections {
			sent++
			conn.receive <- final
		}
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
