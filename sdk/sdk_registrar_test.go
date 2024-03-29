package sdk

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRegistrar_GetSdkOrNil(t *testing.T) {
	reg := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": {Key: "key"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())
	defer reg.Close()

	assert.NotNil(t, reg.GetSdkOrNil("test"))
}

func TestRegistrar_All(t *testing.T) {
	reg := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test1": {Key: "key1"}, "test2": {Key: "key2"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())
	defer reg.Close()

	assert.Equal(t, 2, len(reg.GetAll()))
}

func TestClient_Close(t *testing.T) {
	reg := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": {Key: "key"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())

	c := reg.GetSdkOrNil("test").(*client)
	reg.Close()
	utils.WithTimeout(1*time.Second, func() {
		<-c.ctx.Done()
	})
}
