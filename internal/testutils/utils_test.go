package testutils

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
