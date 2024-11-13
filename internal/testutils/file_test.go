package testutils

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestTmpFile(t *testing.T) {
	var f string
	UseTempFile("", func(file string) {
		f = file
		_, err := os.Stat(file)
		assert.NoError(t, err)

		WriteIntoFile(file, "test")
		res := ReadFile(file)
		assert.Equal(t, "test", res)
	})
	_, err := os.Stat(f)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
