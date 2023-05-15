package utils

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"github.com/cespare/xxhash/v2"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func WithTimeout(timeout time.Duration, f func()) {
	t := time.After(timeout)
	done := make(chan struct{})
	go func() {
		select {
		case <-t:
			panic("test timeout expired")
		case <-done:
		}
	}()
	f()
	done <- struct{}{}
}

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

func Min(args ...int) int {
	min := args[0]
	for _, x := range args {
		if x < min {
			min = x
		}
	}
	return min
}

func Sha1Hex(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func FastHash(b []byte) uint64 {
	h := xxhash.New()
	_, _ = h.Write(b)
	return h.Sum64()
}

func FastHashHex(b []byte) string {
	h := xxhash.New()
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func Obfuscate(str string, clearLen int) string {
	l := len(str)
	if l < clearLen {
		return strings.Repeat("*", utf8.RuneCountInString(str))
	}
	toObfuscate := str[0 : l-clearLen]
	return strings.Repeat("*", utf8.RuneCountInString(toObfuscate)) + str[l-clearLen:l]
}

func AddSdkIdContextParam(r *http.Request) {
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}
