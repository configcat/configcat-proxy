package main

import (
	"flag"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestAppMain(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDK1_BASE_URL", "https://test-cdn-global.configcat.com")
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
	time.Sleep(2 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 0, exitCode)
}

func TestAppMain_Disabled_Everything(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDK1_BASE_URL", "https://test-cdn-global.configcat.com")
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

func TestAppMain_GRPCOnly(t *testing.T) {
	resetFlags()
	t.Setenv("CONFIGCAT_SDK1_BASE_URL", "https://test-cdn-global.configcat.com")
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_GRPC_PORT", "5092")
	t.Setenv("CONFIGCAT_HTTP_ENABLED", "false")
	t.Setenv("CONFIGCAT_DIAG_ENABLED", "false")

	var exitCode int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		exitCode = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(2 * time.Second)
	closeSignal <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 0, exitCode)
}

func TestAppMain_Cache(t *testing.T) {
	resetFlags()
	s := miniredis.RunT(t)
	t.Setenv("CONFIGCAT_SDK1_BASE_URL", "https://test-cdn-global.configcat.com")
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1":"XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"}`)
	t.Setenv("CONFIGCAT_HTTP_PORT", "5101")
	t.Setenv("CONFIGCAT_GRPC_PORT", "5102")
	t.Setenv("CONFIGCAT_DIAG_PORT", "5103")
	t.Setenv("CONFIGCAT_CACHE_REDIS_ENABLED", "true")
	t.Setenv("CONFIGCAT_CACHE_REDIS_ADDRESSES", "[\""+s.Addr()+"\"]")

	var exitCode int
	closeSignal := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		exitCode = run(closeSignal)
		wg.Done()
	}()
	time.Sleep(2 * time.Second)
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
	wg.Wait()

	assert.Equal(t, 1, exitCode)
}

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
}
