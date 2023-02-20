package utils

import (
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
