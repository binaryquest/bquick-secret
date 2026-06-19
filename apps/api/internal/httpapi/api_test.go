package httpapi

import (
	"testing"

	"bquick-secret/apps/api/internal/config"
)

func TestValidateCreateRejectsMissingRecipientWhenEmailEnabled(t *testing.T) {
	api := &API{cfg: config.Config{MaxSecretBytes: 1024, MaxExpiryMinutes: 10080}}
	req := validCreateRequest()
	req.SendEmail = true
	req.RecipientEmail = ""

	_, _, _, _, _, message := api.validateCreate(req)
	if message != "recipient email is required when email is enabled" {
		t.Fatalf("expected recipient email validation, got %q", message)
	}
}

func TestValidateCreateRejectsOversizedPayload(t *testing.T) {
	api := &API{cfg: config.Config{MaxSecretBytes: 2, MaxExpiryMinutes: 10080}}
	req := validCreateRequest()

	_, _, _, _, _, message := api.validateCreate(req)
	if message != "encrypted payload is invalid" {
		t.Fatalf("expected payload validation, got %q", message)
	}
}

func TestValidateCreateRequiresPassphraseMetadata(t *testing.T) {
	api := &API{cfg: config.Config{MaxSecretBytes: 1024, MaxExpiryMinutes: 10080}}
	req := validCreateRequest()
	req.PassphraseEnabled = true

	_, _, _, _, _, message := api.validateCreate(req)
	if message != "wrapped key is required" {
		t.Fatalf("expected wrapped key validation, got %q", message)
	}
}

func TestValidateCreateRequiresRevealProofWhenNotifyEnabled(t *testing.T) {
	api := &API{cfg: config.Config{MaxSecretBytes: 1024, MaxExpiryMinutes: 10080}}
	req := validCreateRequest()
	req.NotifyOnReveal = true

	_, _, _, _, _, message := api.validateCreate(req)
	if message != "reveal proof is required" {
		t.Fatalf("expected reveal proof validation, got %q", message)
	}
}

func validCreateRequest() createSecretRequest {
	return createSecretRequest{
		SenderEmail:      "sender@example.com",
		RecipientEmail:   "recipient@example.com",
		EncryptedPayload: "YWJjZA",
		IV:               "MTIzNDU2Nzg5MDEy",
		Algorithm:        "AES-256-GCM",
		Version:          1,
		ExpiresInMinutes: 60,
		OneTime:          boolPtr(true),
		ManualLink:       true,
	}
}

func boolPtr(value bool) *bool {
	return &value
}
