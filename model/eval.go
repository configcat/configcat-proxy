package model

import "github.com/configcat/configcat-proxy/sdk"

type ResponsePayload struct {
	Value       interface{} `json:"value"`
	VariationId string      `json:"variationId"`
}

type EvalRequest struct {
	Key  string        `json:"key"`
	User sdk.UserAttrs `json:"user"`
}

func PayloadFromEvalData(evalData *sdk.EvalData) ResponsePayload {
	return ResponsePayload{Value: evalData.Value, VariationId: evalData.VariationId}
}
