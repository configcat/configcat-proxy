//go:build !debug

package diag

import (
	"net/http"
)

func setupDebugEndpoints(_ *http.ServeMux) {}
