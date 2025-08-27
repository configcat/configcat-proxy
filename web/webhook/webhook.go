package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
)

const signatureHeader = "X-ConfigCat-Webhook-Signature-V1"
const idHeader = "X-ConfigCat-Webhook-ID"
const timestampHeader = "X-ConfigCat-Webhook-Timestamp"

type Server struct {
	sdkRegistrar  sdk.Registrar
	logger        log.Logger
	autoSdkConfig *config.ProfileConfig
}

func NewServer(autoSdkConfig *config.ProfileConfig, sdkRegistrar sdk.Registrar, log log.Logger) *Server {
	whLogger := log.WithPrefix("webhook")
	return &Server{
		sdkRegistrar:  sdkRegistrar,
		logger:        whLogger,
		autoSdkConfig: autoSdkConfig,
	}
}

func (s *Server) ServeWebhookTest(w http.ResponseWriter, r *http.Request) {
	if s.autoSdkConfig.WebhookSigningKey != "" {
		if !s.validateSignature(s.autoSdkConfig.WebhookSigningKey, s.autoSdkConfig.WebhookSignatureValidFor, r) {
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
	}

	s.logger.Infof("webhook request received")
}

func (s *Server) ServeWebhookSdkId(w http.ResponseWriter, r *http.Request) {
	sdkId := r.PathValue("sdkId")
	if sdkId == "" {
		http.Error(w, "'sdkId' path parameter must be set", http.StatusBadRequest)
		return
	}
	sdkClient := s.sdkRegistrar.GetSdkOrNil(sdkId)
	if sdkClient == nil {
		http.Error(w, "SDK not found for identifier: '"+sdkId+"'", http.StatusNotFound)
		return
	}

	if sdkClient.WebhookSigningKey() != "" {
		if !s.validateSignature(sdkClient.WebhookSigningKey(), sdkClient.WebhookSignatureValidFor(), r) {
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
	}

	// Everything OK, refresh
	s.logger.Infof("webhook request received, refreshing")
	_ = sdkClient.Refresh()
}

func (s *Server) validateSignature(signingKey string, validFor int, r *http.Request) bool {
	signatures := r.Header.Get(signatureHeader)
	webhookId := r.Header.Get(idHeader)
	timestampStr := r.Header.Get(timestampHeader)
	if signatures == "" || webhookId == "" || timestampStr == "" {
		s.logger.Debugf("request missing a signature validation header")
		return false
	}
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil || timestamp < (time.Now().Unix()-int64(validFor)) {
		s.logger.Debugf("request is too old, rejecting")
		return false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Debugf("reading request body failed, rejecting")
		return false
	}
	payloadToSign := webhookId + timestampStr + string(body)
	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(payloadToSign))

	calcSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	receivedSignatures := strings.Split(signatures, ",")
	var found = false
	for _, sig := range receivedSignatures {
		if sig == calcSignature {
			found = true
		}
	}
	if !found {
		s.logger.Debugf("no matching signatures found")
		return false
	}
	s.logger.Debugf("signature validation passed")
	return true
}
