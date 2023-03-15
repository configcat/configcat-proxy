package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/julienschmidt/httprouter"
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
	sdkClients map[string]sdk.Client
	logger     log.Logger
}

func NewServer(sdkClients map[string]sdk.Client, log log.Logger) *Server {
	whLogger := log.WithPrefix("webhook")
	return &Server{
		sdkClients: sdkClients,
		logger:     whLogger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := httprouter.ParamsFromContext(r.Context())
	env := vars.ByName("env")
	if env == "" {
		http.Error(w, "'env' path parameter must be set", http.StatusBadRequest)
		return
	}
	sdkClient, ok := s.sdkClients[env]
	if !ok {
		http.Error(w, "Invalid environment identifier: '"+env+"'", http.StatusBadRequest)
		return
	}

	if sdkClient.WebhookSigningKey() != "" {
		signatures := r.Header.Get(signatureHeader)
		webhookId := r.Header.Get(idHeader)
		timestampStr := r.Header.Get(timestampHeader)
		if signatures == "" || webhookId == "" || timestampStr == "" {
			s.logger.Debugf("request missing a signature validation header")
			http.Error(w, "Signature validation failed", http.StatusBadRequest)
			return
		}
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil || timestamp < (time.Now().Unix()-int64(sdkClient.WebhookSignatureValidFor())) {
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
		mac := hmac.New(sha256.New, []byte(sdkClient.WebhookSigningKey()))
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
	_ = sdkClient.Refresh()
}
