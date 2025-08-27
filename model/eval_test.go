package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadFromEvalData(t *testing.T) {
	eval := &EvalData{Value: "test", VariationId: "varId"}
	payload := PayloadFromEvalData(eval)

	assert.Equal(t, "test", payload.Value)
	assert.Equal(t, "varId", payload.VariationId)
}
