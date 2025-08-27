package utils

import (
	"testing"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/stretchr/testify/assert"
)

func TestObfuscate(t *testing.T) {
	assert.Equal(t, "**st", Obfuscate("test", 2))
	assert.Equal(t, "****-text", Obfuscate("test-text", 5))
	assert.Equal(t, "****", Obfuscate("test", 6))
}

func TestDedupStringSlice(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, DedupStringSlice([]string{"a", "b", "b", "a"}))
	assert.Equal(t, []string{"a", "b"}, DedupStringSlice([]string{"a", "b"}))
}

func TestKeysOfMap(t *testing.T) {
	assert.Contains(t, KeysOfMap(map[string]int{"a": 1, "b": 2}), "a")
	assert.Contains(t, KeysOfMap(map[string]int{"a": 1, "b": 2}), "b")
}

func TestKeysOfSyncMap(t *testing.T) {
	map1 := xsync.NewMapOf[string, int]()
	map1.Store("a", 1)
	map1.Store("b", 2)
	assert.Contains(t, KeysOfSyncMap(map1), "a")
	assert.Contains(t, KeysOfSyncMap(map1), "b")
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

func TestExcept(t *testing.T) {
	assert.Equal(t, []string{"c", "d"}, Except([]string{"a", "b", "c", "d"}, []string{"a", "b"}))
}

func TestUint64ToBytes(t *testing.T) {
	assert.Equal(t, []byte{0x9a, 0xb7, 0x7, 0x2f, 0x62, 0xaf, 0xa7, 0xd}, Uint64ToBytes(983947879834433434))
}
