package utils

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"github.com/cespare/xxhash/v2"
	"github.com/julienschmidt/httprouter"
	"github.com/puzpuzpuz/xsync/v3"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

func Base64URLDecode(encoded string) ([]byte, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
	}
	return decoded, nil
}

func FastHashHex(b []byte) string {
	h := xxhash.New()
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateEtag(b []byte) string {
	return "W/" + "\"" + FastHashHex(b) + "\""
}

func Obfuscate(str string, clearLen int) string {
	l := len(str)
	if l < clearLen {
		return strings.Repeat("*", utf8.RuneCountInString(str))
	}
	toObfuscate := str[0 : l-clearLen]
	return strings.Repeat("*", utf8.RuneCountInString(toObfuscate)) + str[l-clearLen:l]
}

func KeysOfMap[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

func KeysOfSyncMap[K comparable, V any](m *xsync.MapOf[K, V]) []K {
	keys := make([]K, 0, m.Size())
	m.Range(func(key K, value V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

func Except[T ~[]K, K comparable](a T, b T) T {
	var r T
	for _, va := range a {
		found := false
		for _, vb := range b {
			if va == vb {
				found = true
				break
			}
		}
		if !found {
			r = append(r, va)
		}
	}
	return r
}

func DedupStringSlice(strings []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, item := range strings {
		if _, value := keys[item]; !value {
			keys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func WithTimeout(timeout time.Duration, f func()) {
	t := time.After(timeout)
	done := make(chan struct{})
	go func() {
		select {
		case <-t:
			panic("timeout expired")
		case <-done:
		}
	}()
	f()
	done <- struct{}{}
}

func WaitUntil(timeout time.Duration, f func() bool) {
	t := time.After(timeout)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			select {
			case <-t:
				panic("timeout expired")
			default:
				if f() {
					wg.Done()
					return
				}
			}
		}
	}()
	wg.Wait()
}

func AddSdkIdContextParam(r *http.Request) {
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}

func AddSdkIdContextParamWithSdkId(r *http.Request, sdkId string) {
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: sdkId}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}
