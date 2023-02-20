package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const signatureHeader = "X-ConfigCat-Webhook-Signature-V1"
const idHeader = "X-ConfigCat-Webhook-ID"
const timestampHeader = "X-ConfigCat-Webhook-Timestamp"

type Server struct {
	sdkClient sdk.Client
	config    config.WebhookConfig
	logger    log.Logger
}

func NewServer(client sdk.Client, config config.WebhookConfig, log log.Logger) *Server {
	whLogger := log.WithPrefix("webhook")
	return &Server{
		sdkClient: client,
		config:    config,
		logger:    whLogger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.config.SigningKey != "" {
		signatures := r.Header.Get(signatureHeader)
		webhookId := r.Header.Get(idHeader)
		timestampStr := r.Header.Get(timestampHeader)
		if signatures == "" || webhookId == "" || timestampStr == "" {
			s.logger.Debugf("request missing a signature validation header")
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil || timestamp < (time.Now().Unix()-int64(s.config.SignatureValidFor)) {
			s.logger.Debugf("request is too old, rejecting")
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.logger.Debugf("reading request body failed, rejecting")
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
		payloadToSign := webhookId + timestampStr + string(body)
		mac := hmac.New(sha256.New, []byte(s.config.SigningKey))
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
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
		s.logger.Debugf("signature validation passed")
	}

	// Everything OK, refresh
	s.logger.Infof("webhook request received, refreshing")
	_ = s.sdkClient.Refresh()
}
