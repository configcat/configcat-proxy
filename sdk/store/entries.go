package store

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/internal/utils"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"sync/atomic"
	"time"
)

type ConfigJsonV5 struct {
	Entries     map[string]*Setting `json:"f"`
	Preferences *Preferences        `json:"p,omitempty"`
}

type Setting struct {
	VariationID     string                `json:"i"`
	Value           interface{}           `json:"v"`
	Type            configcat.SettingType `json:"t"`
	RolloutRules    []*RolloutRule        `json:"r"`
	PercentageRules []*PercentageRule     `json:"p"`
}

type RolloutRule struct {
	VariationID         string               `json:"i"`
	Value               interface{}          `json:"v"`
	ComparisonAttribute string               `json:"a"`
	ComparisonValue     string               `json:"c"`
	Comparator          configcat.Comparator `json:"t"`
}

type PercentageRule struct {
	VariationID string      `json:"i"`
	Value       interface{} `json:"v"`
	Percentage  int64       `json:"p"`
}

type Preferences struct {
	URL      string                     `json:"u"`
	Redirect *configcat.RedirectionKind `json:"r"`
}

type EntryStore interface {
	LoadEntry() *EntryWithEtag
	ComposeBytes() []byte
	StoreEntry(data []byte, fetchTime time.Time, eTag string)
}

type EntryWithEtag struct {
	ConfigJson []byte
	ETag       string
	FetchTime  time.Time
}

type entryStore struct {
	entry    atomic.Pointer[EntryWithEtag]
	modified chan struct{}
}

func NewEntryStore() EntryStore {
	e := entryStore{
		modified: make(chan struct{}, 1),
	}
	config := configcat.ConfigJson{}
	initial, _ := json.Marshal(config)
	e.entry.Store(entryWithEtag(initial, time.Time{}, utils.GenerateEtag(initial)))
	return &e
}

func (e *entryStore) LoadEntry() *EntryWithEtag {
	return e.entry.Load()
}

func (e *entryStore) ComposeBytes() []byte {
	entry := e.entry.Load()
	return configcatcache.CacheSegmentsToBytes(entry.FetchTime, entry.ETag, entry.ConfigJson)
}

func (e *entryStore) StoreEntry(configJson []byte, fetchTime time.Time, eTag string) {
	e.entry.Store(entryWithEtag(configJson, fetchTime, eTag))
}

func entryWithEtag(configJson []byte, fetchTime time.Time, eTag string) *EntryWithEtag {
	return &EntryWithEtag{
		ConfigJson: configJson,
		ETag:       eTag,
		FetchTime:  fetchTime}
}
