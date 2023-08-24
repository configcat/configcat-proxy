package statistics

type EvalEvent struct {
	SdkId     string
	FlagKey   string
	Value     interface{}
	UserAttrs map[string]string
}

type Reporter interface {
	ReportEvaluation(event *EvalEvent)
	Close()
}
