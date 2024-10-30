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
	assert.NotEmpty(t, pubSub.subscriptions)
	assert.NotNil(t, pubSub.subscriptions[sub])

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
	assert.Empty(t, pubSub.subscriptions)
}
