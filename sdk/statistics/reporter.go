package statistics

type Reporter interface {
	ReportEvaluation(flagKey string, value interface{}, attrs map[string]string)
	Close()
}
