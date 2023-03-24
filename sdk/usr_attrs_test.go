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
