package statistics

type Reporter interface {
	ReportEvaluation(sdkId string, flagKey string, value interface{}, attrs map[string]string)
	Close()
}
