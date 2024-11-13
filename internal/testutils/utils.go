//go:build testing

package testutils

import (
	"context"
	"github.com/julienschmidt/httprouter"
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
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}

func AddSdkIdContextParamWithSdkId(r *http.Request, sdkId string) {
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: sdkId}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}
