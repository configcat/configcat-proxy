package statistics

type EvalEvent struct {
	SdkId     string
	FlagKey   string
	Value     interface{}
	UserAttrs map[string]interface{}
}

type Reporter interface {
	ReportEvaluation(event *EvalEvent)
	Close()
}
