package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMin(t *testing.T) {
	assert.Equal(t, 2, Min(4, 2))
	assert.Equal(t, 1, Min(7, 3, 1, 2))
}
