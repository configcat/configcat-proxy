package pubsub

import (
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPubSub_Sub_Unsub(t *testing.T) {
	pubSub := NewPublisher[struct{}]().(*pubSub[struct{}])

	sub := make(chan struct{})
	pubSub.Subscribe(sub)
	utils.WaitUntil(2*time.Second, func() bool {
		_, ok := pubSub.subscriptions[sub]
		return ok
	})

	msg := struct{}{}
	pubSub.Publish(msg)
	recv := <-sub
	assert.Equal(t, msg, recv)

	pubSub.Unsubscribe(sub)
	utils.WaitUntil(2*time.Second, func() bool {
		return len(pubSub.subscriptions) == 0
	})

	pubSub.Close()
	sub2 := make(chan struct{})
	pubSub.Subscribe(sub2)
	_, ok := pubSub.subscriptions[sub2]
	assert.False(t, ok)
}
