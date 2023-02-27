package status

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"net/http"
	"sync"
	"time"
)

type Component int
type SDKSource string
type HealthStatus string
type SDKMode string

const (
	SDK   Component = 1
	Cache Component = 2

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
	ReportOk(component Component, message string)
	ReportError(component Component, err error)

	HttpHandler() http.HandlerFunc
}

type Status struct {
	Status HealthStatus `json:"status"`
	SDK    SdkStatus    `json:"sdk"`
	Cache  CacheStatus  `json:"cache"`
}

type SdkStatus struct {
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
	records map[Component][]record
	mu      sync.RWMutex
	status  Status
}

func NewNullReporter() Reporter {
	return &reporter{records: make(map[Component][]record)}
}

func NewReporter(conf config.Config) Reporter {
	r := &reporter{
		records: make(map[Component][]record),
		status: Status{
			Status: Healthy,
			SDK: SdkStatus{
				Mode: Online,
				Source: SdkSourceStatus{
					Type:   RemoteSrc,
					Status: Healthy,
				},
			},
			Cache: CacheStatus{
				Status: Healthy,
			},
		},
	}
	if conf.SDK.Offline.Enabled {
		r.status.SDK.Mode = Offline
		if conf.SDK.Offline.Local.FilePath != "" {
			r.status.SDK.Source.Type = FileSrc
			r.status.Cache.Status = NA
		} else {
			r.status.SDK.Source.Type = CacheSrc
		}
	}
	if !conf.SDK.Cache.Redis.Enabled {
		r.status.Cache.Status = NA
	}

	return r
}

func (r *reporter) ReportOk(component Component, message string) {
	r.appendRecord(component, "[ok] "+message, false)
}

func (r *reporter) ReportError(component Component, err error) {
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
	if sdk, ok := r.records[SDK]; ok {
		r.checkStatus(sdk, &current.SDK.Source.Records, &current.SDK.Source.Status)
	}
	if cache, ok := r.records[Cache]; ok {
		r.checkStatus(cache, &current.Cache.Records, &current.Cache.Status)
	}
	if current.SDK.Source.Status == Degraded {
		current.Status = Degraded
	}
	return current
}

func (r *reporter) checkStatus(records []record, targetRecords *[]string, status *HealthStatus) {
	length := len(records)
	*targetRecords = make([]string, length)
	var errorCount = 0
	for i, msg := range records {
		(*targetRecords)[i] = fmt.Sprintf("%s: %s", msg.time.UTC().Format(time.RFC1123), msg.message)
		if i >= length-maxLastErrorsMeaningDegraded {
			if msg.isError {
				errorCount++
			} else {
				errorCount--
			}
		}
		if errorCount >= maxLastErrorsMeaningDegraded {
			*status = Degraded
		}
	}
}

func (r *reporter) appendRecord(component Component, message string, isError bool) {
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
