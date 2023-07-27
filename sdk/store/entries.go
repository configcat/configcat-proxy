package store

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/go-sdk/v8/configcatcache"
	"sync/atomic"
	"time"
)

type RootNode struct {
	Entries     map[string]*Entry `json:"f"`
	Preferences *Preferences      `json:"p,omitempty"`
}

type Entry struct {
	VariationID     string            `json:"i"`
	Value           interface{}       `json:"v"`
	Type            int               `json:"t"`
	RolloutRules    []*RolloutRule    `json:"r"`
	PercentageRules []*PercentageRule `json:"p"`
}

type RolloutRule struct {
	VariationID         string      `json:"i"`
	Value               interface{} `json:"v"`
	ComparisonAttribute string      `json:"a"`
	ComparisonValue     string      `json:"c"`
	Comparator          int         `json:"t"`
}

type PercentageRule struct {
	VariationID string      `json:"i"`
	Value       interface{} `json:"v"`
	Percentage  int64       `json:"p"`
}

type Preferences struct {
	URL      string `json:"u"`
	Redirect int    `json:"r"`
}

func (r *RootNode) Fixup() {
	if r.Entries == nil {
		r.Entries = make(map[string]*Entry)
	} else {
		for _, e := range r.Entries {
			if e.PercentageRules == nil {
				e.PercentageRules = make([]*PercentageRule, 0)
			}
			if e.RolloutRules == nil {
				e.RolloutRules = make([]*RolloutRule, 0)
			}
		}
	}
}

type EntryStore interface {
	LoadEntry() *EntryWithEtag
	ComposeBytes() []byte
	StoreEntry(data []byte, fetchTime time.Time, eTag string)
}

type EntryWithEtag struct {
	ConfigJson    []byte
	CachedETag    string
	FetchTime     time.Time
	GeneratedETag string
}

type entryStore struct {
	entry    atomic.Pointer[EntryWithEtag]
	modified chan struct{}
}

func NewEntryStore() EntryStore {
	e := entryStore{
		modified: make(chan struct{}, 1),
	}
	root := RootNode{}
	root.Fixup()
	initial, _ := json.Marshal(root)
	e.entry.Store(parseEntryWithEtag(initial, time.Time{}, "initial-etag"))
	return &e
}

func (e *entryStore) LoadEntry() *EntryWithEtag {
	return e.entry.Load()
}

func (e *entryStore) ComposeBytes() []byte {
	entry := e.entry.Load()
	return configcatcache.CacheSegmentsToBytes(entry.FetchTime, entry.CachedETag, entry.ConfigJson)
}

func (e *entryStore) StoreEntry(configJson []byte, fetchTime time.Time, eTag string) {
	e.entry.Store(parseEntryWithEtag(configJson, fetchTime, eTag))
}

func parseEntryWithEtag(configJson []byte, fetchTime time.Time, eTag string) *EntryWithEtag {
	return &EntryWithEtag{
		ConfigJson:    configJson,
		GeneratedETag: utils.GenerateEtag(configJson),
		CachedETag:    eTag,
		FetchTime:     fetchTime}
}
