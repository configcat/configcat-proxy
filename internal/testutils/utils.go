//go:build testing

package testutils

import (
	"net/http"
	"sync"
	"time"
)

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
	r.SetPathValue("sdkId", "test")
}

func AddSdkIdContextParamWithSdkId(r *http.Request, sdkId string) {
	r.SetPathValue("sdkId", sdkId)
}
