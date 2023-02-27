package utils

import (
	"encoding/base64"
	"time"
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
