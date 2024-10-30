package stream

import (
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServer_GetStreamOrNil(t *testing.T) {
	reg, _, _ := testutils.NewTestRegistrarT(t)
	srv := NewServer(reg, nil, log.NewNullLogger(), "test").(*server)

	str := srv.GetStreamOrNil("test")
	assert.NotNil(t, str)
	assert.Equal(t, 1, srv.streams.Size())

	str = srv.GetStreamOrNil("nonexisting")
	assert.Nil(t, str)

	srv.Close()
	assert.Equal(t, 0, srv.streams.Size())
}
