package store

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/internal/utils"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"sync/atomic"
	"time"
)

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
