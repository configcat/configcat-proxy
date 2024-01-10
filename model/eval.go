package model

import (
	configcat "github.com/configcat/go-sdk/v9"
)

type EvalData struct {
	Value       interface{}
	VariationId string
	User        configcat.User
}

type ResponsePayload struct {
	Value       interface{} `json:"value"`
	VariationId string      `json:"variationId"`
}

type EvalRequest struct {
	Key  string    `json:"key"`
	User UserAttrs `json:"user"`
}

func PayloadFromEvalData(evalData *EvalData) ResponsePayload {
	return ResponsePayload{Value: evalData.Value, VariationId: evalData.VariationId}
}
