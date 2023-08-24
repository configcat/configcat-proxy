package sdk

import (
	"github.com/stretchr/testify/assert"
	"hash/maphash"
	"testing"
)

func TestUserAttributes_Identical(t *testing.T) {
	user1 := UserAttrs{"email": "user1@test.com", "id": "user1"}
	user2 := UserAttrs{"id": "user1", "email": "user1@test.com"}
	s := maphash.MakeSeed()
	assert.Equal(t, user1.Discriminator(s), user2.Discriminator(s))
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
