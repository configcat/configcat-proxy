package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestAppMain(t *testing.T) {
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)

	var code int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		code = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(5 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 0, code)
}
