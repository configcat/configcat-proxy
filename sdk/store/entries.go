package store

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"sync/atomic"
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

type EntryWithEtag struct {
	CachedJson []byte
	Etag       string
}

type EntryStore struct {
	entry    atomic.Pointer[EntryWithEtag]
	modified chan struct{}
}

func NewEntryStore() *EntryStore {
	e := EntryStore{
		modified: make(chan struct{}, 1),
	}
	root := RootNode{}
	root.Fixup()
	initial, _ := json.Marshal(root)
	e.entry.Store(parseEntryWithEtag(initial))
	return &e
}

func (e *EntryStore) LoadEntry() *EntryWithEtag {
	return e.entry.Load()
}

func (e *EntryStore) StoreEntry(data []byte) {
	e.entry.Store(parseEntryWithEtag(data))
}

func (e *EntryStore) Notify() {
	e.modified <- struct{}{}
}

func (e *EntryStore) GetLatestJson() *EntryWithEtag {
	return e.LoadEntry()
}

func (e *EntryStore) Modified() <-chan struct{} {
	return e.modified
}

func parseEntryWithEtag(data []byte) *EntryWithEtag {
	hash := sha1.Sum(data)
	etag := fmt.Sprintf("W/\"%x\"", hash)
	return &EntryWithEtag{CachedJson: data, Etag: etag}
}
