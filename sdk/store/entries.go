package store

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"strconv"
	"strings"
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
	LoadEntry(version config.SDKVersion) *EntryWithEtag
	ComposeBytes(version config.SDKVersion) []byte
	StoreEntry(data []byte, fetchTime time.Time, eTag string)
}

type EntryWithEtag struct {
	ConfigJson []byte
	ETag       string
	FetchTime  time.Time
}

type entryStore struct {
	entryV6    atomic.Pointer[EntryWithEtag]
	entryV5    atomic.Pointer[EntryWithEtag]
	cacheKeyV6 string
	cacheKeyV5 string
	sdkVersion config.SDKVersion
	modified   chan struct{}
}

func NewEntryStore(version config.SDKVersion) EntryStore {
	e := entryStore{
		modified:   make(chan struct{}, 1),
		sdkVersion: version,
	}
	configV5 := ConfigJsonV5{}
	configV6 := configcat.ConfigJson{}
	initialV5, _ := json.Marshal(configV5)
	initialV6, _ := json.Marshal(configV6)
	e.entryV5.Store(entryWithEtag(initialV5, time.Time{}, utils.GenerateEtag(initialV5)))
	e.entryV6.Store(entryWithEtag(initialV6, time.Time{}, utils.GenerateEtag(initialV6)))
	return &e
}

func (e *entryStore) LoadEntry(version config.SDKVersion) *EntryWithEtag {
	if version == config.V5 {
		return e.entryV5.Load()
	} else {
		return e.entryV6.Load()
	}
}

func (e *entryStore) ComposeBytes(version config.SDKVersion) []byte {
	var entry *EntryWithEtag
	if version == config.V5 {
		entry = e.entryV5.Load()
	} else {
		entry = e.entryV6.Load()
	}
	return configcatcache.CacheSegmentsToBytes(entry.FetchTime, entry.ETag, entry.ConfigJson)
}

func (e *entryStore) StoreEntry(configJson []byte, fetchTime time.Time, eTag string) {
	e.entryV6.Store(entryWithEtag(configJson, fetchTime, eTag))
	if e.sdkVersion == config.V5 {
		v5, err := buildV5Json(configJson)
		if err == nil {
			e.entryV5.Store(entryWithEtag(v5, fetchTime, utils.GenerateEtag(v5)))
		}
	}
}

func entryWithEtag(configJson []byte, fetchTime time.Time, eTag string) *EntryWithEtag {
	return &EntryWithEtag{
		ConfigJson: configJson,
		ETag:       eTag,
		FetchTime:  fetchTime}
}

func buildV5Json(configV6Json []byte) ([]byte, error) {
	var configV6 configcat.ConfigJson
	err := json.Unmarshal(configV6Json, &configV6)
	if err != nil {
		return nil, err
	}
	configV5 := ConfigJsonV5{}
	if configV6.Preferences != nil {
		configV5.Preferences = &Preferences{
			URL:      configV6.Preferences.URL,
			Redirect: configV6.Preferences.Redirect,
		}
	}
	configV5.Entries = make(map[string]*Setting, len(configV6.Settings))
	for key, setting := range configV6.Settings {
		settingV5 := &Setting{
			Value:       setting.Value.Value,
			Type:        setting.Type,
			VariationID: setting.VariationID,
		}
		if len(setting.TargetingRules) > 0 {
			for _, rule := range setting.TargetingRules {
				ruleV5 := &RolloutRule{
					Value:       rule.ServedValue.Value.Value,
					VariationID: rule.ServedValue.VariationID,
				}
				if len(rule.Conditions) > 0 {
					cond := rule.Conditions[0]
					if cond.UserCondition != nil {
						ruleV5.Comparator = cond.UserCondition.Comparator
						ruleV5.ComparisonAttribute = cond.UserCondition.ComparisonAttribute
						ruleV5.ComparisonValue = getCompValue(cond.UserCondition)
					}
					if cond.SegmentCondition != nil {
						segment := configV6.Segments[cond.SegmentCondition.Index]
						ruleV5.Comparator = segment.Conditions[0].Comparator
						ruleV5.ComparisonAttribute = segment.Conditions[0].ComparisonAttribute
						ruleV5.ComparisonValue = getCompValue(segment.Conditions[0])
					}
				}
				settingV5.RolloutRules = append(settingV5.RolloutRules, ruleV5)
			}
		}
		if len(setting.PercentageOptions) > 0 {
			for _, option := range setting.PercentageOptions {
				settingV5.PercentageRules = append(settingV5.PercentageRules, &PercentageRule{
					Percentage:  option.Percentage,
					Value:       option.Value.Value,
					VariationID: option.VariationID,
				})
			}
		}
		configV5.Entries[key] = settingV5
	}
	return json.Marshal(configV5)
}

func getCompValue(cond *configcat.UserCondition) string {
	if cond.Comparator.IsList() {
		return strings.Join(cond.StringArrayValue, ",")
	}
	if cond.Comparator.IsNumeric() && cond.DoubleValue != nil {
		return strconv.FormatFloat(*cond.DoubleValue, 'f', -1, 64)
	}
	if cond.StringValue != nil {
		return *cond.StringValue
	}
	return ""
}
