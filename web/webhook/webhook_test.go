package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/configcat-proxy/utils"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWebhook_Signature_Bad(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
	client := newClient(t, h, key)
	srv := NewServer(client, config.WebhookConfig{Enabled: true, SigningKey: "test-key"}, log.NewNullLogger())

	t.Run("headers missing", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := http.Request{Method: http.MethodGet}
		srv.ServeHTTP(res, &req)
		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
	t.Run("signature wrong GET", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
		req.Header.Set("X-ConfigCat-Webhook-Signature-V1", "wrong")
		req.Header.Set("X-ConfigCat-Webhook-ID", "1")
		req.Header.Set("X-ConfigCat-Webhook-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		srv.ServeHTTP(res, req)
		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
	t.Run("signature wrong POST", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
		req.Header.Set("X-ConfigCat-Webhook-Signature-V1", "wrong")
		req.Header.Set("X-ConfigCat-Webhook-ID", "1")
		req.Header.Set("X-ConfigCat-Webhook-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		srv.ServeHTTP(res, req)
		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
}

func TestWebhook_Signature_Ok(t *testing.T) {
	t.Run("signature OK GET", func(t *testing.T) {
		key := configcattest.RandomSDKKey()
		var h = &configcattest.Handler{}
		_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
		client := newClient(t, h, key)
		srv := NewServer(client, config.WebhookConfig{Enabled: true, SigningKey: "test-key"}, log.NewNullLogger())

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
		sub := client.SubConfigChanged("hook1")
		utils.WithTimeout(2*time.Second, func() {
			<-client.Ready()
		}) // wait for the SDK to do the initialization
		_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: false}})
		srv.ServeHTTP(res, req)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		assert.Equal(t, http.StatusOK, res.Code)
	})
	t.Run("signature OK POST", func(t *testing.T) {
		key := configcattest.RandomSDKKey()
		var h = &configcattest.Handler{}
		_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
		client := newClient(t, h, key)
		srv := NewServer(client, config.WebhookConfig{Enabled: true, SigningKey: "test-key"}, log.NewNullLogger())

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
		sub := client.SubConfigChanged("hook1")
		utils.WithTimeout(2*time.Second, func() {
			<-client.Ready()
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
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
	client := newClient(t, h, key)
	srv := NewServer(client, config.WebhookConfig{Enabled: true, SigningKey: "test-key", SignatureValidFor: 1}, log.NewNullLogger())

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
	time.Sleep(2100 * time.Millisecond) // expire timestamp
	srv.ServeHTTP(res, req)
	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func newClient(t *testing.T, h *configcattest.Handler, key string) sdk.Client {
	srv := httptest.NewServer(h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client
}
