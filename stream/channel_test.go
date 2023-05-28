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

	conn1 := str.CreateConnection("test", user1)
	conn2 := str.CreateConnection("test", user1)
	conn3 := str.CreateConnection("test", user2)
	conn4 := str.CreateConnection("test", nil)
	conn5 := str.CreateConnection("test", nil)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Same(t, conn1, str.channels["test"][user1Discriminator].(*singleFlagChannel).connections[0])
	assert.Same(t, conn2, str.channels["test"][user1Discriminator].(*singleFlagChannel).connections[1])
	assert.Same(t, conn3, str.channels["test"][user2Discriminator].(*singleFlagChannel).connections[0])
	assert.Same(t, conn4, str.channels["test"][0].(*singleFlagChannel).connections[0])
	assert.Same(t, conn5, str.channels["test"][0].(*singleFlagChannel).connections[1])

	assert.Equal(t, 3, len(str.channels["test"]))
	assert.Equal(t, 2, len(str.channels["test"][user1Discriminator].(*singleFlagChannel).connections))
	assert.Equal(t, 1, len(str.channels["test"][user2Discriminator].(*singleFlagChannel).connections))
	assert.Equal(t, 2, len(str.channels["test"][0].(*singleFlagChannel).connections))

	assert.Equal(t, user1, str.channels["test"][user1Discriminator].(*singleFlagChannel).user)
	assert.Equal(t, user2, str.channels["test"][user2Discriminator].(*singleFlagChannel).user)
	assert.Nil(t, str.channels["test"][0].(*singleFlagChannel).user)

	str.CloseConnection(conn2, "test")
	str.CloseConnection(conn3, "test")
	str.CloseConnection(conn4, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections

	assert.Equal(t, 2, len(str.channels["test"]))
	assert.Equal(t, 1, len(str.channels["test"][user1Discriminator].(*singleFlagChannel).connections))
	assert.Nil(t, str.channels["test"][user2Discriminator])
	assert.Equal(t, 1, len(str.channels["test"][0].(*singleFlagChannel).connections))

	str.CloseConnection(conn1, "test")
	str.CloseConnection(conn5, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections
	assert.Empty(t, str.channels)
}

func TestStream_Close(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test").(*stream)

	user1 := sdk.UserAttrs{"id": "u1"}
	user2 := sdk.UserAttrs{"id": "u2"}

	user1Discriminator := user1.Discriminator(str.seed)
	user2Discriminator := user2.Discriminator(str.seed)

	_ = str.CreateConnection("test", user1)
	_ = str.CreateConnection("test", user2)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Equal(t, 2, len(str.channels["test"]))
	assert.Equal(t, 1, len(str.channels["test"][user1Discriminator].(*singleFlagChannel).connections))
	assert.Equal(t, 1, len(str.channels["test"][user2Discriminator].(*singleFlagChannel).connections))

	str.Close()
	_ = str.CreateConnection("test", user1)
	_ = str.CreateConnection("test", user2)
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
			_ = str.CreateConnection("test", user)
			wg.Done()
		}(i)
	}
	for i := iter; i < iter+iter; i++ {
		go func(it int) {
			is := strconv.Itoa(it)
			user := sdk.UserAttrs{"id": "u" + is}
			_ = str.CreateConnection(AllFlagsDiscriminator, user)
			wg.Done()
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, iter, len(str.channels["test"]))
	assert.Equal(t, iter, len(str.channels[AllFlagsDiscriminator]))
}
