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
	Key1 string
	Key2 string
}
