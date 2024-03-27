package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWebhook_Signature_Bad(t *testing.T) {
	reg, _, _ := newRegistrar(t, "test-key", 300)
	srv := NewServer(reg, log.NewNullLogger())

	t.Run("headers missing", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := http.Request{Method: http.MethodGet}
		testutils.AddSdkIdContextParam(&req)
		srv.ServeHTTP(res, &req)
		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
	t.Run("signature wrong GET", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
		req.Header.Set("X-ConfigCat-Webhook-Signature-V1", "wrong")
		req.Header.Set("X-ConfigCat-Webhook-ID", "1")
		req.Header.Set("X-ConfigCat-Webhook-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)
		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
	t.Run("signature wrong POST", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
		req.Header.Set("X-ConfigCat-Webhook-Signature-V1", "wrong")
		req.Header.Set("X-ConfigCat-Webhook-ID", "1")
		req.Header.Set("X-ConfigCat-Webhook-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)
		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
}

func TestWebhook_Signature_Ok(t *testing.T) {
	t.Run("signature OK GET", func(t *testing.T) {
		reg, h, key := newRegistrar(t, "test-key", 300)
		srv := NewServer(reg, log.NewNullLogger())

		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		id := "1"
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		payloadToSign := fmt.Sprintf("%s%s", id, timestamp)
		mac := hmac.New(sha256.New, []byte("test-key"))
		mac.Write([]byte(payloadToSign))
		signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-ConfigCat-Webhook-Signature-V1", signature)
		req.Header.Set("X-ConfigCat-Webhook-ID", id)
		req.Header.Set("X-ConfigCat-Webhook-Timestamp", timestamp)
		testutils.AddSdkIdContextParam(req)
		sub := reg.GetSdkOrNil("test").SubConfigChanged("hook1")
		utils.WithTimeout(2*time.Second, func() {
			<-reg.GetSdkOrNil("test").Ready()
		}) // wait for the SDK to do the initialization
		_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: false}})
		srv.ServeHTTP(res, req)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		assert.Equal(t, http.StatusOK, res.Code)
	})
	t.Run("signature OK POST", func(t *testing.T) {
		reg, h, key := newRegistrar(t, "test-key", 300)
		srv := NewServer(reg, log.NewNullLogger())

		id := "1"
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		body := "body"
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		payloadToSign := fmt.Sprintf("%s%s%s", id, timestamp, body)
		mac := hmac.New(sha256.New, []byte("test-key"))
		mac.Write([]byte(payloadToSign))
		signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-ConfigCat-Webhook-Signature-V1", signature)
		req.Header.Set("X-ConfigCat-Webhook-ID", id)
		req.Header.Set("X-ConfigCat-Webhook-Timestamp", timestamp)
		testutils.AddSdkIdContextParam(req)
		sub := reg.GetSdkOrNil("test").SubConfigChanged("hook1")
		utils.WithTimeout(2*time.Second, func() {
			<-reg.GetSdkOrNil("test").Ready()
		}) // wait for the SDK to do the initialization
		_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: false}})
		srv.ServeHTTP(res, req)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		assert.Equal(t, http.StatusOK, res.Code)
	})
}

func TestWebhook_Signature_Replay_Reject(t *testing.T) {
	reg, _, _ := newRegistrar(t, "test-key", 1)
	srv := NewServer(reg, log.NewNullLogger())

	id := "1"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body := "body"
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	payloadToSign := fmt.Sprintf("%s%s%s", id, timestamp, body)
	mac := hmac.New(sha256.New, []byte("test-key"))
	mac.Write([]byte(payloadToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-ConfigCat-Webhook-Signature-V1", signature)
	req.Header.Set("X-ConfigCat-Webhook-ID", id)
	req.Header.Set("X-ConfigCat-Webhook-Timestamp", timestamp)
	testutils.AddSdkIdContextParam(req)
	time.Sleep(2100 * time.Millisecond) // expire timestamp
	srv.ServeHTTP(res, req)
	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func newRegistrar(t *testing.T, signingKey string, validFor int) (sdk.Registrar, *configcattest.Handler, string) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
	srv := httptest.NewServer(h)
	reg := testutils.NewTestRegistrar(&config.SDKConfig{BaseUrl: srv.URL, Key: key, WebhookSigningKey: signingKey, WebhookSignatureValidFor: validFor}, nil)
	t.Cleanup(func() {
		srv.Close()
		reg.Close()
	})
	return reg, h, key
}
