package testutils

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestAddSdkIdContextParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	AddSdkIdContextParam(req)

	assert.Equal(t, "test", req.PathValue("sdkId"))
}

func TestAddSdkIdContextParamWithSdkId(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	AddSdkIdContextParamWithSdkId(req, "t1")

	assert.Equal(t, "t1", req.PathValue("sdkId"))
}
