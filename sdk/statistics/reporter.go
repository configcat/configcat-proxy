package statistics

type Reporter interface {
	ReportEvaluation(envId string, flagKey string, value interface{}, attrs map[string]string)
	Close()
}
