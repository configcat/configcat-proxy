package sdk

import (
	"sort"
)

type UserAttrs struct {
	Attrs map[string]string
}

func (attrs UserAttrs) Discriminator() string {
	var result string
	keys := make([]string, len(attrs.Attrs))
	i := 0
	for k := range attrs.Attrs {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		result += k + attrs.Attrs[k]
	}
	return result
}
