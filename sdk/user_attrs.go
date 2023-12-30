package sdk

import (
	"encoding/json"
	"fmt"
	"hash/maphash"
	"maps"
	"strconv"
)

type UserAttrs map[string]interface{}

func (attrs *UserAttrs) UnmarshalJSON(data []byte) error {
	var res map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	*attrs = make(map[string]interface{}, len(res))
	for k, v := range res {
		switch val := v.(type) {
		case []interface{}:
			strArr := make([]string, 0, len(val))
			for _, strVal := range val {
				if str, ok := strVal.(string); ok {
					strArr = append(strArr, str)
				}
			}
			(*attrs)[k] = strArr
		case string, float64, int:
			(*attrs)[k] = v
		default:
			return fmt.Errorf("'%s' has an invalid type, only 'string', 'number', and 'string[]' types are allowed", k)
		}
	}
	return nil
}

func (attrs UserAttrs) Discriminator(s maphash.Seed) uint64 {
	var h maphash.Hash
	h.SetSeed(s)
	var curr uint64
	for k, v := range attrs {
		h.Reset()
		_, _ = h.WriteString(k)
		hk := h.Sum64()
		h.Reset()
		switch val := v.(type) {
		case string:
			_, _ = h.WriteString(val)
		case int:
			_, _ = h.Write(strconv.AppendInt(nil, int64(val), 10))
		case float64:
			_, _ = h.Write(strconv.AppendFloat(nil, val, 'f', -1, 64))
		case []interface{}:
			for _, strVal := range val {
				if str, ok := strVal.(string); ok {
					_, _ = h.WriteString(str)
				}
			}
		}
		curr ^= h.Sum64() + 0x9e3779b97f4a7c15 + (hk << 12) + (hk >> 4)
	}
	return curr
}

func (attrs UserAttrs) GetAttribute(attr string) interface{} { // for the ConfigCat SDK
	return attrs[attr]
}

func MergeUserAttrs(first UserAttrs, second UserAttrs) UserAttrs {
	if first == nil && second == nil {
		return nil
	}
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	final := make(map[string]interface{})
	maps.Copy(final, first)
	maps.Copy(final, second)
	return final
}
