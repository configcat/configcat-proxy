package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMin(t *testing.T) {
	assert.Equal(t, 2, Min(4, 2))
	assert.Equal(t, 1, Min(7, 3, 1, 2))
}

func TestObfuscate(t *testing.T) {
	assert.Equal(t, "**st", Obfuscate("test", 2))
	assert.Equal(t, "****-text", Obfuscate("test-text", 5))
	assert.Equal(t, "****", Obfuscate("test", 6))
}
