//go:build !race

package stream

import (
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStream_Connections(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test").(*stream)
	defer str.Close()

	user1 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u1"}}
	user2 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u2"}}

	user1Discriminator := user1.Discriminator()
	user2Discriminator := user2.Discriminator()

	conn1 := str.CreateConnection("test", user1)
	conn2 := str.CreateConnection("test", user1)
	conn3 := str.CreateConnection("test", user2)
	conn4 := str.CreateConnection("test", nil)
	conn5 := str.CreateConnection("test", nil)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Same(t, conn1, str.channels["test"+user1Discriminator].connections[0])
	assert.Same(t, conn2, str.channels["test"+user1Discriminator].connections[1])
	assert.Same(t, conn3, str.channels["test"+user2Discriminator].connections[0])
	assert.Same(t, conn4, str.channels["test"].connections[0])
	assert.Same(t, conn5, str.channels["test"].connections[1])

	assert.Equal(t, 3, len(str.channels))
	assert.Equal(t, 2, len(str.channels["test"+user1Discriminator].connections))
	assert.Equal(t, 1, len(str.channels["test"+user2Discriminator].connections))
	assert.Equal(t, 2, len(str.channels["test"].connections))

	assert.Same(t, user1, str.channels["test"+user1Discriminator].user)
	assert.Same(t, user2, str.channels["test"+user2Discriminator].user)
	assert.Nil(t, str.channels["test"].user)

	str.CloseConnection(conn2, "test")
	str.CloseConnection(conn3, "test")
	str.CloseConnection(conn4, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections

	assert.Equal(t, 2, len(str.channels))
	assert.Equal(t, 1, len(str.channels["test"+user1Discriminator].connections))
	assert.Nil(t, str.channels["test"+user2Discriminator])
	assert.Equal(t, 1, len(str.channels["test"].connections))

	str.CloseConnection(conn1, "test")
	str.CloseConnection(conn5, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections
	assert.Empty(t, str.channels)
}

func TestStream_Close(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test").(*stream)

	user1 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u1"}}
	user2 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u2"}}

	user1Discriminator := user1.Discriminator()
	user2Discriminator := user2.Discriminator()

	_ = str.CreateConnection("test", user1)
	_ = str.CreateConnection("test", user2)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Equal(t, 2, len(str.channels))
	assert.Equal(t, 1, len(str.channels["test"+user1Discriminator].connections))
	assert.Equal(t, 1, len(str.channels["test"+user2Discriminator].connections))

	str.Close()
	_ = str.CreateConnection("test", user1)
	_ = str.CreateConnection("test", user2)
}
