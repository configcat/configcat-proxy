package status

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"net/http"
	"sync"
	"time"
)

type SDKSource string
type HealthStatus string
type SDKMode string

const (
	Cache    = "cache"
	InfluxDb = "influxdb"

	FileSrc   SDKSource = "file"
	RemoteSrc SDKSource = "remote"
	CacheSrc  SDKSource = "cache"

	Healthy  HealthStatus = "healthy"
	Degraded HealthStatus = "degraded"
	NA       HealthStatus = "n/a"

	Offline SDKMode = "offline"
	Online  SDKMode = "online"
)

const maxRecordCount = 5
const maxLastErrorsMeaningDegraded = 2

type Reporter interface {
	ReportOk(component string, message string)
	ReportError(component string, err error)

	HttpHandler() http.HandlerFunc
}

type Status struct {
	Status HealthStatus          `json:"status"`
	SDKs   map[string]*SdkStatus `json:"sdks"`
	Cache  CacheStatus           `json:"cache"`
}

type SdkStatus struct {
	SdkKey string          `json:"key"`
	Mode   SDKMode         `json:"mode"`
	Source SdkSourceStatus `json:"source"`
}

type SdkSourceStatus struct {
	Type    SDKSource    `json:"type"`
	Status  HealthStatus `json:"status"`
	Records []string     `json:"records"`
}

type CacheStatus struct {
	Status  HealthStatus `json:"status"`
	Records []string     `json:"records"`
}

type record struct {
	time    time.Time
	isError bool
	message string
}

type reporter struct {
	records map[string][]record
	mu      sync.RWMutex
	status  Status
	conf    *config.Config
}

func NewNullReporter() Reporter {
	return &reporter{records: make(map[string][]record)}
}

func NewReporter(conf *config.Config) Reporter {
	r := &reporter{
		conf:    conf,
		records: make(map[string][]record),
		status: Status{
			Status: Healthy,
			Cache: CacheStatus{
				Status: Healthy,
			},
		},
	}
	sdks := make(map[string]*SdkStatus, len(conf.SDKs))
	for key, sdk := range conf.SDKs {
		status := &SdkStatus{
			Mode:   Online,
			SdkKey: utils.Obfuscate(sdk.Key, 5),
			Source: SdkSourceStatus{
				Type:   RemoteSrc,
				Status: Healthy,
			},
		}
		if sdk.Offline.Enabled {
			status.Mode = Offline
			if sdk.Offline.Local.FilePath != "" {
				status.Source.Type = FileSrc
				r.status.Cache.Status = NA
			} else {
				status.Source.Type = CacheSrc
			}
		}
		if !conf.Cache.Redis.Enabled {
			r.status.Cache.Status = NA
			if status.Source.Type == CacheSrc {
				status.Source.Status = Degraded
				r.ReportError(key, fmt.Errorf("cache offline source enabled without a configured cache"))
			}
		}
		sdks[key] = status
	}
	r.status.SDKs = sdks
	return r
}

func (r *reporter) ReportOk(component string, message string) {
	r.appendRecord(component, "[ok] "+message, false)
}

func (r *reporter) ReportError(component string, err error) {
	r.appendRecord(component, "[error] "+err.Error(), true)
}

func (r *reporter) HttpHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status, err := json.Marshal(r.getStatus())
		if err != nil {
			http.Error(w, "Error producing status", http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(status)
	}
}

func (r *reporter) getStatus() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current := r.status
	overallStatus := Healthy
	for key := range r.conf.SDKs {
		if sdk, ok := r.records[key]; ok {
			stat := current.SDKs[key].Source.Status
			rec, stat := r.checkStatus(sdk)
			current.SDKs[key].Source.Records = rec
			current.SDKs[key].Source.Status = stat
			if stat == Degraded {
				overallStatus = Degraded
			}
		}
	}
	if cache, ok := r.records[Cache]; ok {
		rec, stat := r.checkStatus(cache)
		current.Cache.Records = rec
		current.Cache.Status = stat
	}
	current.Status = overallStatus
	return current
}

func (r *reporter) checkStatus(records []record) ([]string, HealthStatus) {
	length := len(records)
	targetRecords := make([]string, length)
	var errorCount = 0
	for i, msg := range records {
		targetRecords[i] = msg.time.UTC().Format(time.RFC1123) + ": " + msg.message
		if i >= length-maxLastErrorsMeaningDegraded {
			if msg.isError {
				errorCount++
			} else {
				errorCount--
			}
		}
	}
	if errorCount > 0 && errorCount >= utils.Min(maxLastErrorsMeaningDegraded, length) {
		return targetRecords, Degraded
	}
	return targetRecords, Healthy
}

func (r *reporter) appendRecord(component string, message string, isError bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recs, ok := r.records[component]
	if !ok {
		recs = make([]record, 0, maxRecordCount)
	}
	recs = append(recs, record{time: time.Now(), isError: isError, message: message})
	if len(recs) > maxRecordCount {
		recs = recs[1:]
	}
	r.records[component] = recs
}
