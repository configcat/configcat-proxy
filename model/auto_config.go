package model

type ProxyConfigModel struct {
	SDKs    map[string]*SdkConfigModel
	Options OptionsModel
}

type OptionsModel struct {
	PollInterval   int
	DataGovernance string
}

type SdkConfigModel struct {
	SDKKey string
}
