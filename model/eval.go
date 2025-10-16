package model

import (
	configcat "github.com/configcat/go-sdk/v9"
)

type EvalData struct {
	Value       interface{}
	VariationId string
	Error       error
	User        configcat.User
	IsTargeting bool
}

type ResponsePayload struct {
	Value       interface{} `json:"value"`
	VariationId string      `json:"variationId"`
}

type EvalRequest struct {
	SdkKey string    `json:"sdkKey"`
	Key    string    `json:"key"`
	User   UserAttrs `json:"user"`
}

func PayloadFromEvalData(evalData *EvalData) ResponsePayload {
	return ResponsePayload{Value: evalData.Value, VariationId: evalData.VariationId}
}
