//go:build !race

package stream

import (
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/stretchr/testify/assert"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStream_Connections(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test").(*stream)
	defer str.Close()

	user1 := sdk.UserAttrs{"id": "u1"}
	user2 := sdk.UserAttrs{"id": "u2"}

	user1Discriminator := user1.Discriminator(str.seed)
	user2Discriminator := user2.Discriminator(str.seed)

	conn1 := str.CreateSingleFlagConnection("test", user1)
	conn2 := str.CreateSingleFlagConnection("test", user1)
	conn3 := str.CreateSingleFlagConnection("test", user2)
	conn4 := str.CreateSingleFlagConnection("test", nil)
	conn5 := str.CreateSingleFlagConnection("test", nil)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Same(t, conn1, str.singleFlagChannels["test"][user1Discriminator].connections[0])
	assert.Same(t, conn2, str.singleFlagChannels["test"][user1Discriminator].connections[1])
	assert.Same(t, conn3, str.singleFlagChannels["test"][user2Discriminator].connections[0])
	assert.Same(t, conn4, str.singleFlagChannels["test"][0].connections[0])
	assert.Same(t, conn5, str.singleFlagChannels["test"][0].connections[1])

	assert.Equal(t, 3, len(str.singleFlagChannels["test"]))
	assert.Equal(t, 2, len(str.singleFlagChannels["test"][user1Discriminator].connections))
	assert.Equal(t, 1, len(str.singleFlagChannels["test"][user2Discriminator].connections))
	assert.Equal(t, 2, len(str.singleFlagChannels["test"][0].connections))

	assert.Equal(t, user1, str.singleFlagChannels["test"][user1Discriminator].user)
	assert.Equal(t, user2, str.singleFlagChannels["test"][user2Discriminator].user)
	assert.Nil(t, str.singleFlagChannels["test"][0].user)

	str.CloseSingleFlagConnection(conn2, "test")
	str.CloseSingleFlagConnection(conn3, "test")
	str.CloseSingleFlagConnection(conn4, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections

	assert.Equal(t, 2, len(str.singleFlagChannels["test"]))
	assert.Equal(t, 1, len(str.singleFlagChannels["test"][user1Discriminator].connections))
	assert.Nil(t, str.singleFlagChannels["test"][user2Discriminator])
	assert.Equal(t, 1, len(str.singleFlagChannels["test"][0].connections))

	str.CloseSingleFlagConnection(conn1, "test")
	str.CloseSingleFlagConnection(conn5, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections
	assert.Empty(t, str.singleFlagChannels)
}

func TestStream_Close(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test").(*stream)

	user1 := sdk.UserAttrs{"id": "u1"}
	user2 := sdk.UserAttrs{"id": "u2"}

	user1Discriminator := user1.Discriminator(str.seed)
	user2Discriminator := user2.Discriminator(str.seed)

	_ = str.CreateSingleFlagConnection("test", user1)
	_ = str.CreateSingleFlagConnection("test", user2)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Equal(t, 2, len(str.singleFlagChannels["test"]))
	assert.Equal(t, 1, len(str.singleFlagChannels["test"][user1Discriminator].connections))
	assert.Equal(t, 1, len(str.singleFlagChannels["test"][user2Discriminator].connections))

	str.Close()
	_ = str.CreateSingleFlagConnection("test", user1)
	_ = str.CreateSingleFlagConnection("test", user2)
}

func TestStream_Collision(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test").(*stream)

	iter := 500000
	wg := &sync.WaitGroup{}
	wg.Add(iter + iter)
	for i := 0; i < iter; i++ {
		go func(it int) {
			is := strconv.Itoa(it)
			user := sdk.UserAttrs{"id": "u" + is}
			_ = str.CreateSingleFlagConnection("test", user)
			wg.Done()
		}(i)
	}
	for i := iter; i < iter+iter; i++ {
		go func(it int) {
			is := strconv.Itoa(it)
			user := sdk.UserAttrs{"id": "u" + is}
			_ = str.CreateAllFlagsConnection(user)
			wg.Done()
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, iter, len(str.singleFlagChannels["test"]))
	assert.Equal(t, iter, len(str.allFlagChannels))
}
