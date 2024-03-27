package model

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"hash/maphash"
	"testing"
)

func TestUserAttributes_Identical(t *testing.T) {
	user1 := UserAttrs{"email": "user1@test.com", "id": "user1", "custom1": 42}
	user2 := UserAttrs{"id": "user1", "custom1": 42, "email": "user1@test.com"}
	s := maphash.MakeSeed()
	assert.Equal(t, user1.Discriminator(s), user2.Discriminator(s))
}

func TestUserAttributes_GetAttributes(t *testing.T) {
	user := UserAttrs{"email": "user1@test.com", "id": "user1", "custom1": 42}

	assert.Equal(t, "user1@test.com", user.GetAttribute("email"))
	assert.Equal(t, "user1", user.GetAttribute("id"))
	assert.Equal(t, 42, user.GetAttribute("custom1"))
}

func TestUserAttributes_Merge(t *testing.T) {
	a := UserAttrs{"a": "b", "c": "d"}
	b := UserAttrs{"e": "f", "g": "h"}
	c := UserAttrs{"a": "i", "g": "j"}

	assert.Equal(t, UserAttrs{"a": "b", "c": "d", "e": "f", "g": "h"}, MergeUserAttrs(a, b))
	assert.Equal(t, UserAttrs{"a": "i", "c": "d", "g": "j"}, MergeUserAttrs(a, c))
	assert.Equal(t, UserAttrs{"e": "f", "g": "j", "a": "i"}, MergeUserAttrs(b, c))
	assert.Equal(t, a, MergeUserAttrs(a, nil))
	assert.Equal(t, a, MergeUserAttrs(nil, a))
	assert.Nil(t, MergeUserAttrs(nil, nil))
}

type testStruct struct {
	U UserAttrs `json:"user" yaml:"user"`
}

func TestUserAttributes_UnmarshalJSON(t *testing.T) {
	j := `{"user":{"a":1,"b":["x","z"],"c":"test"}}`
	var test testStruct
	err := json.Unmarshal([]byte(j), &test)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), test.U["a"])
	assert.Equal(t, []string{"x", "z"}, test.U["b"])
	assert.Equal(t, "test", test.U["c"])
}

func TestUserAttributes_UnmarshalJSON_Invalid(t *testing.T) {
	j := `{"user":{"a":true}}`
	var test testStruct
	err := json.Unmarshal([]byte(j), &test)
	assert.ErrorContains(t, err, "'a' has an invalid type, only 'string', 'number', and 'string[]' types are allowed")
}

func TestUserAttributes_UnmarshalYAML(t *testing.T) {
	j := `
user:
  a: 1
  b: ["x","z"]
  c: "test"
`
	var test testStruct
	err := yaml.Unmarshal([]byte(j), &test)
	assert.NoError(t, err)
	assert.Equal(t, 1, test.U["a"])
	assert.Equal(t, []string{"x", "z"}, test.U["b"])
	assert.Equal(t, "test", test.U["c"])
}

func TestUserAttributes_UnmarshalYAML_Invalid(t *testing.T) {
	j := `
user:
  a: true
`
	var test testStruct
	err := yaml.Unmarshal([]byte(j), &test)
	assert.ErrorContains(t, err, "'a' has an invalid type, only 'string', 'number', and 'string[]' types are allowed")
}
