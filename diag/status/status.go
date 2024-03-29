package status

import (
	"encoding/json"
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
	Cache = "cache"

	FileSrc   SDKSource = "file"
	RemoteSrc SDKSource = "remote"
	CacheSrc  SDKSource = "cache"

	Healthy      HealthStatus = "healthy"
	Degraded     HealthStatus = "degraded"
	Initializing HealthStatus = "initializing"
	Down         HealthStatus = "down"
	NA           HealthStatus = "n/a"

	Offline SDKMode = "offline"
	Online  SDKMode = "online"
)

const maxRecordCount = 5
const maxLastErrorsMeaningDegraded = 2

type Reporter interface {
	RegisterSdk(sdkId string, conf *config.SDKConfig)
	ReportOk(component string, message string)
	ReportError(component string, message string)
	GetStatus() Status

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
	conf    *config.CacheConfig
}

func NewEmptyReporter() Reporter {
	return NewReporter(&config.CacheConfig{})
}

func NewReporter(conf *config.CacheConfig) Reporter {
	r := &reporter{
		conf:    conf,
		records: make(map[string][]record),
		status: Status{
			Status: Initializing,
			Cache: CacheStatus{
				Status: Initializing,
			},
			SDKs: map[string]*SdkStatus{},
		},
	}
	return r
}

func (r *reporter) RegisterSdk(sdkId string, conf *config.SDKConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	status := &SdkStatus{
		Mode:   Online,
		SdkKey: utils.Obfuscate(conf.Key, 5),
		Source: SdkSourceStatus{
			Type:   RemoteSrc,
			Status: Initializing,
		},
	}
	r.status.SDKs[sdkId] = status
	if conf.Offline.Enabled {
		status.Mode = Offline
		if conf.Offline.Local.FilePath != "" {
			status.Source.Type = FileSrc
			r.status.Cache.Status = NA
		} else {
			status.Source.Type = CacheSrc
		}
	}
	if !r.conf.IsSet() {
		r.status.Cache.Status = NA
		if status.Source.Type == CacheSrc {
			r.appendRecord(sdkId, "cache offline source enabled without a configured cache", true)
		}
	}
}

func (r *reporter) ReportOk(component string, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.appendRecord(component, message, false)
}

func (r *reporter) ReportError(component string, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.appendRecord(component, message, true)
}

func (r *reporter) HttpHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status, err := json.Marshal(r.GetStatus())
		if err != nil {
			http.Error(w, "Error producing status", http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(status)
	}
}

func (r *reporter) GetStatus() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.status
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
	if isError {
		message = "[error] " + message
	} else {
		message = "[ok] " + message
	}

	recs, ok := r.records[component]
	if !ok {
		recs = make([]record, 0, maxRecordCount)
	}
	recs = append(recs, record{time: time.Now(), isError: isError, message: message})
	if len(recs) > maxRecordCount {
		recs = recs[1:]
	}
	r.records[component] = recs
	rec, stat := r.checkStatus(recs)
	if component == Cache {
		r.status.Cache.Records = rec
		r.status.Cache.Status = stat
	} else if sdk, ok := r.status.SDKs[component]; ok {
		sdk.Source.Records = rec
		if stat == Degraded && (sdk.Source.Status == Initializing || sdk.Source.Status == Down) {
			stat = Down
		}
		sdk.Source.Status = stat

		allSdksDown := true
		hasDegradedSdk := false
		for _, sdk := range r.status.SDKs {
			if sdk.Source.Status != Down {
				allSdksDown = false
			}
			if sdk.Source.Status != Healthy {
				hasDegradedSdk = true
			}
		}
		if !hasDegradedSdk && !allSdksDown {
			r.status.Status = Healthy
		} else {
			if hasDegradedSdk {
				r.status.Status = Degraded
			}
			if allSdksDown {
				r.status.Status = Down
			}
		}
	}
}
