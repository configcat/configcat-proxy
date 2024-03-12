package main

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestAppMain(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_HTTP_PORT", "5081")
	t.Setenv("CONFIGCAT_GRPC_PORT", "5082")
	t.Setenv("CONFIGCAT_DIAG_PORT", "5083")

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

func TestAppMain_Disabled_Everything(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_HTTP_ENABLED", "false")
	t.Setenv("CONFIGCAT_GRPC_ENABLED", "false")
	t.Setenv("CONFIGCAT_DIAG_ENABLED", "false")

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

	assert.Equal(t, 0, exitCode)
}

func TestAppMain_Invalid_Conf(t *testing.T) {
	resetFlags()
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
	resetFlags()
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

func TestAppMain_Invalid_TLS_Cert(t *testing.T) {
	resetFlags()
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

func TestAppMain_ErrorChan_Diag_Conflicting_Ports(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_DIAG_PORT", "8050")
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

func TestAppMain_ErrorChan_Grpc_Conflicting_Ports(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_GRPC_PORT", "8051")
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

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
}
