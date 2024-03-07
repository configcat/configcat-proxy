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

	var exitCode int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		exitCode = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(5 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 0, exitCode)
}

func TestAppMain_Invalid_Conf(t *testing.T) {
	var exitCode int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		exitCode = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(1 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 1, exitCode)
}

func TestAppMain_Invalid_Config_YAML(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "-c=/tmp/non-existing.yml"}

	var exitCode int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		exitCode = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(1 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 1, exitCode)
}

func TestAppMain_ErrorChannel(t *testing.T) {
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_TLS_ENABLED", "true")
	t.Setenv("CONFIGCAT_TLS_CERTIFICATES", `[{"key":"./key","cert":"./cert"}]`)
	var exitCode int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		exitCode = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(1 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 1, exitCode)
}
