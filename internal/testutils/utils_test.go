package testutils

import (
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestAddSdkIdContextParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	AddSdkIdContextParam(req)

	assert.Equal(t, httprouter.Params{httprouter.Param{Key: "sdkId", Value: "test"}}, req.Context().Value(httprouter.ParamsKey))
}

func TestAddSdkIdContextParamWithSdkId(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	AddSdkIdContextParamWithSdkId(req, "t1")

	assert.Equal(t, httprouter.Params{httprouter.Param{Key: "sdkId", Value: "t1"}}, req.Context().Value(httprouter.ParamsKey))
}
