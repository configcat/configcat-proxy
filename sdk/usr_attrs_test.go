package sdk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserAttributes_Identical(t *testing.T) {
	user1 := UserAttrs{map[string]string{"id": "user1", "email": "user1@test.com"}}
	user2 := UserAttrs{map[string]string{"email": "user1@test.com", "id": "user1"}}
	assert.Equal(t, "emailuser1@test.comiduser1", user1.Discriminator())
	assert.Equal(t, "emailuser1@test.comiduser1", user2.Discriminator())
	assert.Equal(t, user1.Discriminator(), user2.Discriminator())
}
