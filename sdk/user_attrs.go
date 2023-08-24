package sdk

import (
	"hash/maphash"
	"maps"
)

type UserAttrs map[string]string

func (attrs UserAttrs) GetAttribute(attr string) string { // for the SDK
	return attrs[attr]
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
		_, _ = h.WriteString(v)
		curr ^= h.Sum64() + 0x9e3779b97f4a7c15 + (hk << 12) + (hk >> 4)
	}
	return curr
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
	final := make(map[string]string)
	maps.Copy(final, first)
	maps.Copy(final, second)
	return final
}
