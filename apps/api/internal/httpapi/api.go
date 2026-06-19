package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bquick-secret/apps/api/internal/config"
	secretcrypto "bquick-secret/apps/api/internal/crypto"
	"bquick-secret/apps/api/internal/email"
	"bquick-secret/apps/api/internal/recaptcha"
	"bquick-secret/apps/api/internal/store"
)

type API struct {
	cfg     config.Config
	store   *store.Store
	mailer  email.Sender
	logger  *slog.Logger
	captcha *recaptcha.Verifier
}

func New(cfg config.Config, store *store.Store, mailer email.Sender, logger *slog.Logger) http.Handler {
	captcha := recaptcha.New(recaptcha.Config{
		APIKey:     cfg.RecaptchaAPIKey,
		AppBaseURL: cfg.AppBaseURL,
		MinScore:   cfg.RecaptchaMinScore,
		ProjectID:  cfg.RecaptchaProjectID,
		SiteKey:    cfg.RecaptchaSiteKey,
	})
	api := &API{cfg: cfg, store: store, mailer: mailer, logger: logger, captcha: captcha}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/secrets", api.createSecret)
	mux.HandleFunc("GET /api/secrets/{publicId}", api.getSecret)
	mux.HandleFunc("POST /api/secrets/{publicId}/revealed", api.secretRevealed)
	mux.HandleFunc("DELETE /api/secrets/{publicId}", api.deleteSecret)
	mux.HandleFunc("GET /api/stats/daily", api.dailyStats)

	return securityHeaders(cfg.AppBaseURL, logging(logger, mux))
}

type createSecretRequest struct {
	SenderEmail       string `json:"senderEmail"`
	RecipientEmail    string `json:"recipientEmail"`
	EncryptedPayload  string `json:"encryptedPayload"`
	IV                string `json:"iv"`
	Algorithm         string `json:"algorithm"`
	Version           int    `json:"version"`
	ExpiresInMinutes  int    `json:"expiresInMinutes"`
	OneTime           *bool  `json:"oneTime"`
	PassphraseEnabled bool   `json:"passphraseEnabled"`
	SendEmail         bool   `json:"sendEmail"`
	ManualLink        bool   `json:"manualLink"`
	NotifyOnReveal    bool   `json:"notifyOnReveal"`
	RevealProof       string `json:"revealProof,omitempty"`
	RecaptchaToken    string `json:"recaptchaToken,omitempty"`
	WrappedKey        string `json:"wrappedKey,omitempty"`
	WrappingIV        string `json:"wrappingIv,omitempty"`
	KDFSalt           string `json:"kdfSalt,omitempty"`
	KDFIterations     int    `json:"kdfIterations,omitempty"`
	KDFAlgorithm      string `json:"kdfAlgorithm,omitempty"`
}

type createSecretResponse struct {
	PublicID    string `json:"publicId"`
	DeleteToken string `json:"deleteToken"`
	EmailSent   bool   `json:"emailSent"`
}

type secretResponse struct {
	EncryptedPayload  string `json:"encryptedPayload"`
	IV                string `json:"iv"`
	Algorithm         string `json:"algorithm"`
	Version           int    `json:"version"`
	OneTime           bool   `json:"oneTime"`
	PassphraseEnabled bool   `json:"passphraseEnabled"`
	WrappedKey        string `json:"wrappedKey,omitempty"`
	WrappingIV        string `json:"wrappingIv,omitempty"`
	KDFSalt           string `json:"kdfSalt,omitempty"`
	KDFIterations     int    `json:"kdfIterations,omitempty"`
	KDFAlgorithm      string `json:"kdfAlgorithm,omitempty"`
}

type deleteSecretRequest struct {
	DeleteToken string `json:"deleteToken"`
}

func (api *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) createSecret(w http.ResponseWriter, r *http.Request) {
	var req createSecretRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(api.cfg.MaxSecretBytes*2))).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.Version == 0 {
		req.Version = 1
	}
	if req.Algorithm == "" {
		req.Algorithm = "AES-256-GCM"
	}
	if req.ExpiresInMinutes == 0 {
		req.ExpiresInMinutes = api.cfg.DefaultExpiryMinutes
	}
	oneTime := api.cfg.DefaultOneTime
	if req.OneTime != nil {
		oneTime = *req.OneTime
	}

	payload, iv, wrapped, wrappingIV, kdfSalt, validationErr := api.validateCreate(req)
	if validationErr != "" {
		writeError(w, http.StatusBadRequest, validationErr)
		return
	}
	if err := api.verifyRecaptcha(r, req.RecaptchaToken, "create_secret"); err != nil {
		writeError(w, http.StatusBadRequest, "recaptcha verification failed")
		return
	}

	senderHash := secretcrypto.Hash(req.SenderEmail)
	ok, err := api.store.AllowRateLimit(r.Context(), "create:"+senderHash, api.cfg.RateLimitCreateHour, time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}
	if !ok {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	if req.SendEmail {
		emailOK, err := api.store.AllowRateLimit(r.Context(), "email:"+senderHash, api.cfg.RateLimitEmailHour, time.Hour)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "request failed")
			return
		}
		if !emailOK {
			writeError(w, http.StatusTooManyRequests, "email rate limit exceeded")
			return
		}
	}

	publicID, err := secretcrypto.RandomToken(12)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}
	deleteToken, err := secretcrypto.RandomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}
	senderNotifyEmail := ""
	revealTokenHash := ""
	if req.NotifyOnReveal {
		senderNotifyEmail = strings.TrimSpace(req.SenderEmail)
		revealTokenHash = secretcrypto.HashToken(req.RevealProof)
	}

	err = api.store.CreateSecret(r.Context(), store.CreateSecretParams{
		PublicID:               publicID,
		EncryptedPayload:       payload,
		IV:                     iv,
		Algorithm:              req.Algorithm,
		Version:                req.Version,
		ExpiresAt:              time.Now().UTC().Add(time.Duration(req.ExpiresInMinutes) * time.Minute),
		OneTime:                oneTime,
		SenderEmailHash:        senderHash,
		RecipientEmailProvided: req.SendEmail,
		ManualLinkEnabled:      req.ManualLink,
		PassphraseEnabled:      req.PassphraseEnabled,
		NotifySenderOnReveal:   req.NotifyOnReveal,
		SenderNotifyEmail:      senderNotifyEmail,
		RevealTokenHash:        revealTokenHash,
		DeleteTokenHash:        secretcrypto.Hash(deleteToken),
		PayloadSizeBytes:       len(payload),
		WrappedKey:             wrapped,
		WrappingIV:             wrappingIV,
		KDFSalt:                kdfSalt,
		KDFIterations:          req.KDFIterations,
		KDFAlgorithm:           req.KDFAlgorithm,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}

	stats := []string{"secrets_created_count"}
	if req.ManualLink {
		stats = append(stats, "manual_links_created_count")
	}
	if req.PassphraseEnabled {
		stats = append(stats, "passphrase_enabled_count")
	}
	if oneTime {
		stats = append(stats, "one_time_enabled_count")
	}
	_ = api.store.IncrementStats(r.Context(), time.Now().UTC(), stats...)

	emailSent := false
	if req.SendEmail {
		keylessURL := api.cfg.AppBaseURL + "/s/" + publicID
		if err := api.mailer.SendSecretLink(r.Context(), req.RecipientEmail, keylessURL); err == nil {
			emailSent = true
			_ = api.store.IncrementStats(r.Context(), time.Now().UTC(), "emails_sent_count")
		} else {
			api.logger.Warn("email_failed", "category", "ses")
		}
	}

	writeJSON(w, http.StatusCreated, createSecretResponse{PublicID: publicID, DeleteToken: deleteToken, EmailSent: emailSent})
}

func (api *API) verifyRecaptcha(r *http.Request, token, action string) error {
	if api.captcha == nil {
		return nil
	}
	if err := api.captcha.Verify(r.Context(), token, action); err != nil {
		var verificationErr *recaptcha.VerificationError
		if errors.As(err, &verificationErr) {
			api.logger.Warn(
				"recaptcha_failed",
				"category", "verification",
				"reason", verificationErr.Reason,
				"status", verificationErr.StatusCode,
				"score", verificationErr.Score,
				"action", verificationErr.Action,
				"expected_action", verificationErr.ExpectedAction,
				"hostname", verificationErr.Hostname,
				"expected_host", verificationErr.ExpectedHost,
				"invalid_reason", verificationErr.InvalidReason,
			)
		} else {
			api.logger.Warn("recaptcha_failed", "category", "verification", "reason", "request_failed")
		}
		return err
	}
	return nil
}

func (api *API) validateCreate(req createSecretRequest) ([]byte, []byte, []byte, []byte, []byte, string) {
	if !looksLikeEmail(req.SenderEmail) {
		return nil, nil, nil, nil, nil, "sender email is required"
	}
	if req.SendEmail && !looksLikeEmail(req.RecipientEmail) {
		return nil, nil, nil, nil, nil, "recipient email is required when email is enabled"
	}
	if req.EncryptedPayload == "" || req.IV == "" {
		return nil, nil, nil, nil, nil, "encrypted payload and iv are required"
	}
	if req.Algorithm != "AES-256-GCM" {
		return nil, nil, nil, nil, nil, "unsupported algorithm"
	}
	if req.Version != 1 {
		return nil, nil, nil, nil, nil, "unsupported payload version"
	}
	if req.ExpiresInMinutes <= 0 || req.ExpiresInMinutes > api.cfg.MaxExpiryMinutes {
		return nil, nil, nil, nil, nil, "expiry is invalid"
	}
	if req.NotifyOnReveal && !validRevealProof(req.RevealProof) {
		return nil, nil, nil, nil, nil, "reveal proof is required"
	}

	payload, err := decodeBase64URL(req.EncryptedPayload)
	if err != nil || len(payload) == 0 || len(payload) > api.cfg.MaxSecretBytes {
		return nil, nil, nil, nil, nil, "encrypted payload is invalid"
	}
	iv, err := decodeBase64URL(req.IV)
	if err != nil || len(iv) != 12 {
		return nil, nil, nil, nil, nil, "iv is invalid"
	}

	var wrapped, wrappingIV, kdfSalt []byte
	if req.PassphraseEnabled {
		var err error
		wrapped, err = decodeBase64URL(req.WrappedKey)
		if err != nil || len(wrapped) == 0 {
			return nil, nil, nil, nil, nil, "wrapped key is required"
		}
		wrappingIV, err = decodeBase64URL(req.WrappingIV)
		if err != nil || len(wrappingIV) != 12 {
			return nil, nil, nil, nil, nil, "wrapping iv is invalid"
		}
		kdfSalt, err = decodeBase64URL(req.KDFSalt)
		if err != nil || len(kdfSalt) < 16 {
			return nil, nil, nil, nil, nil, "kdf salt is invalid"
		}
		if req.KDFIterations < 100000 || req.KDFAlgorithm != "PBKDF2-SHA-256" {
			return nil, nil, nil, nil, nil, "kdf settings are invalid"
		}
	}

	return payload, iv, wrapped, wrappingIV, kdfSalt, ""
}

func (api *API) getSecret(w http.ResponseWriter, r *http.Request) {
	publicID := r.PathValue("publicId")
	if !validPublicID(publicID) {
		writeError(w, http.StatusNotFound, "secret not available")
		return
	}

	payload, err := api.store.GetSecretForOpen(r.Context(), publicID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "secret not available")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}

	_ = api.store.IncrementStats(r.Context(), time.Now().UTC(), "secrets_opened_count")

	writeJSON(w, http.StatusOK, secretResponse{
		EncryptedPayload:  encodeBase64URL(payload.EncryptedPayload),
		IV:                encodeBase64URL(payload.IV),
		Algorithm:         payload.Algorithm,
		Version:           payload.Version,
		OneTime:           payload.OneTime,
		PassphraseEnabled: payload.PassphraseEnabled,
		WrappedKey:        encodeOptional(payload.WrappedKey),
		WrappingIV:        encodeOptional(payload.WrappingIV),
		KDFSalt:           encodeOptional(payload.KDFSalt),
		KDFIterations:     payload.KDFIterations,
		KDFAlgorithm:      payload.KDFAlgorithm,
	})
}

func (api *API) secretRevealed(w http.ResponseWriter, r *http.Request) {
	publicID := r.PathValue("publicId")
	if !validPublicID(publicID) {
		writeError(w, http.StatusNotFound, "secret not available")
		return
	}

	var req struct {
		RevealProof string `json:"revealProof"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || !validRevealProof(req.RevealProof) {
		writeError(w, http.StatusBadRequest, "reveal proof is required")
		return
	}

	senderEmail, err := api.store.ClaimRevealNotification(r.Context(), publicID, secretcrypto.HashToken(req.RevealProof))
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]bool{"notified": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}

	secretURL := api.cfg.AppBaseURL + "/s/" + publicID
	if err := api.mailer.SendRevealNotice(r.Context(), senderEmail, secretURL); err != nil {
		api.logger.Warn("reveal_notice_failed", "category", "ses")
		writeJSON(w, http.StatusOK, map[string]bool{"notified": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"notified": true})
}

func (api *API) deleteSecret(w http.ResponseWriter, r *http.Request) {
	publicID := r.PathValue("publicId")
	if !validPublicID(publicID) {
		writeError(w, http.StatusNotFound, "secret not available")
		return
	}

	var req deleteSecretRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || strings.TrimSpace(req.DeleteToken) == "" {
		writeError(w, http.StatusBadRequest, "delete token is required")
		return
	}

	deleted, err := api.store.DeleteSecret(r.Context(), publicID, secretcrypto.Hash(req.DeleteToken))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "secret not available")
		return
	}
	_ = api.store.IncrementStats(r.Context(), time.Now().UTC(), "secrets_deleted_count")
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (api *API) dailyStats(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" || token != api.cfg.AdminStatsToken {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	stats, err := api.store.ListDailyStats(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeBase64URL(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

func encodeBase64URL(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func encodeOptional(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	return encodeBase64URL(value)
}

func looksLikeEmail(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) <= 254 && strings.Contains(value, "@") && strings.Contains(value, ".")
}

func validPublicID(value string) bool {
	if len(value) < 8 || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func validRevealProof(value string) bool {
	if len(value) < 32 || len(value) > 96 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
