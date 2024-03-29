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

func TestDedupStringSlice(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, DedupStringSlice([]string{"a", "b", "b", "a"}))
	assert.Equal(t, []string{"a", "b"}, DedupStringSlice([]string{"a", "b"}))
}

func TestKeys(t *testing.T) {
	assert.Contains(t, Keys(map[string]int{"a": 1, "b": 2}), "a")
	assert.Contains(t, Keys(map[string]int{"a": 1, "b": 2}), "b")
}

func TestBase64URLDecode(t *testing.T) {
	res, err := Base64URLDecode("dGVzdA==")
	assert.NoError(t, err)
	assert.Equal(t, "test", string(res))
}

func TestFastHashHex(t *testing.T) {
	assert.Equal(t, "4fdcca5ddb678139", FastHashHex([]byte("test")))
}

func TestGenerateEtag(t *testing.T) {
	assert.Equal(t, "W/\"4fdcca5ddb678139\"", GenerateEtag([]byte("test")))
}
