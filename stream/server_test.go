package stream

import (
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/stretchr/testify/assert"
)

func TestServer_GetStreamOrNil(t *testing.T) {
	reg, _, key := sdk.NewTestRegistrarT(t)
	srv := NewServer(reg, telemetry.NewEmptyReporter(), log.NewNullLogger(), "test").(*server)

	str := srv.GetStreamOrNil("test")
	assert.NotNil(t, str)

	strBySdkKey := srv.GetStreamBySdkKeyOrNil(key)
	assert.Equal(t, str, strBySdkKey)

	assert.Nil(t, srv.GetStreamOrNil("nonexisting"))
	assert.Nil(t, srv.GetStreamBySdkKeyOrNil("nonexisting"))

	assert.Equal(t, 1, srv.streams.Size())

	srv.Close()
	assert.Equal(t, 0, srv.streams.Size())
}

func TestServer_AutoRegistrar(t *testing.T) {
	reg, h, _ := sdk.NewTestAutoRegistrarWithAutoConfig(t, config.ProfileConfig{PollInterval: 60}, log.NewNullLogger())
	srv := NewServer(reg, telemetry.NewEmptyReporter(), log.NewNullLogger(), "test").(*server)

	str := srv.GetStreamOrNil("test")
	assert.NotNil(t, str)
	assert.Equal(t, 1, srv.streams.Size())

	h.AddSdk("test2")
	reg.Refresh()

	testutils.WaitUntil(5*time.Second, func() bool {
		return nil != srv.GetStreamOrNil("test2")
	})

	str = srv.GetStreamOrNil("test2")

	h.RemoveSdk("test2")
	reg.Refresh()

	// test that stream closed on removed sdk
	<-str.Closed()

	// test that modified global options resets the sdk client
	str1 := srv.GetStreamOrNil("test").(*stream)
	sdkClient := str1.sdkClient.Load()

	h.ModifyGlobalOpts(model.OptionsModel{PollInterval: 120})
	reg.Refresh()

	testutils.WaitUntil(5*time.Second, func() bool {
		return nil != srv.GetStreamOrNil("test")
	})

	testutils.WaitUntil(5*time.Second, func() bool {
		return sdkClient != str1.sdkClient.Load()
	})
	sdkClient2 := str1.sdkClient.Load()
	assert.NotSame(t, sdkClient, sdkClient2)

	// test close
	srv.Close()
	assert.Equal(t, 0, srv.streams.Size())
}
