# bQuick Secret SRS

Version: 1.0  
Date: 17 June 2026

## 1. System Overview

bQuick Secret is a public encrypted secret-sharing web application. The frontend performs all encryption and decryption in the browser. The backend stores ciphertext, sends email, enforces expiry, manages one-time view state, provides aggregate stats, and performs cleanup.

## 2. Technology Stack

- Frontend: React + Vite + TypeScript
- Crypto: Browser Web Crypto API
- Backend: Go
- Database: Postgres
- Email: Amazon SES
- Abuse protection: Google reCAPTCHA Enterprise for create-secret requests
- Deployment: Docker Compose via Coolify
- Phase 2 Storage: AWS S3

## 3. System Components

### 3.1 Web Frontend

Routes:

- `/` landing page
- `/create` create secret page
- `/s/:publicId` recipient view page
- `/privacy` privacy page
- `/how-it-works` explanation page

Responsibilities:

- UI rendering
- form validation
- reCAPTCHA Enterprise token generation for create-secret submissions
- key generation
- browser-side AES-GCM encryption/decryption
- optional passphrase handling
- link creation with key in URL fragment
- recipient decryption flow
- sender reveal notification signal after successful browser decrypt

### 3.2 Go API

Responsibilities:

- accept encrypted payloads only
- verify reCAPTCHA Enterprise assessments before creating secrets when configured
- create public IDs and delete tokens
- store ciphertext and metadata in Postgres
- send email through SES
- return encrypted payloads
- send optional one-time reveal notices to senders
- enforce expiry and one-time behavior
- update aggregate stats
- run cleanup
- expose health and protected stats endpoint

### 3.3 Postgres

Stores encrypted payloads, metadata, aggregate counters, and rate-limit buckets.

### 3.4 Worker

A cleanup worker deletes expired secrets and, in Phase 2, expired S3 objects. This can be part of the Go API process initially or a separate container later.

## 4. Security Invariants

The backend must never receive or store:

- plaintext secret
- decrypt key
- passphrase
- full URL containing the `#key` fragment
- recipient email after sending
- sender notification email after reveal notice is claimed
- request bodies in logs

The browser must encrypt before upload and decrypt after retrieval.

## 5. Encryption Specification

### 5.1 Text Secrets

- Algorithm: AES-GCM
- Key size: 256-bit
- IV: random per encryption
- Encoding: base64url for transport
- Key transport: URL fragment

Link format:

```text
https://secret.example.com/s/{publicId}#key={decryptKey}
```

### 5.2 Optional Passphrase

Passphrase mode must ensure the link alone is not enough to decrypt. Recommended MVP implementation:

- Generate random data encryption key.
- Encrypt secret with data encryption key.
- Derive wrapping key from passphrase using PBKDF2 or Argon2id through a vetted browser-side implementation.
- Wrap/encrypt data key with passphrase-derived key.
- Store wrapped key metadata in encrypted payload metadata, not the passphrase.

## 6. API Specification

### 6.1 POST /api/secrets

Creates a secret from already-encrypted payload.

Request:

```json
{
  "senderEmail": "sender@example.com",
  "recipientEmail": "recipient@example.com",
  "encryptedPayload": "base64url...",
  "iv": "base64url...",
  "algorithm": "AES-256-GCM",
  "version": 1,
  "expiresInMinutes": 1440,
  "oneTime": true,
  "passphraseEnabled": false,
  "sendEmail": true,
  "manualLink": true,
  "notifyOnReveal": false,
  "recaptchaToken": "recaptcha-enterprise-token"
}
```

Response:

```json
{
  "publicId": "abc123",
  "deleteToken": "delete-token"
}
```

Validation:

- senderEmail required
- recipientEmail required if sendEmail=true
- encryptedPayload required
- iv required
- expiry must be <= 7 days
- payload size must be within configured limit

If notifyOnReveal=true, the backend stores the sender email as the one-time notification target until the reveal notice is claimed.

If reCAPTCHA Enterprise is configured, recaptchaToken is required and must verify for action `create_secret`, the configured site key, the expected hostname, and the configured minimum score.

### 6.2 GET /api/secrets/:publicId

Returns encrypted payload if available.

Response:

```json
{
  "encryptedPayload": "base64url...",
  "iv": "base64url...",
  "algorithm": "AES-256-GCM",
  "version": 1,
  "oneTime": true,
  "passphraseEnabled": false
}
```

If expired, deleted, or consumed, return 404 with a generic message.

For one-time secrets, the server should consume on first successful encrypted payload fetch.

### 6.3 POST /api/secrets/:publicId/revealed

Called by the browser only after local decryption succeeds. If sender notification was enabled and not already claimed, the backend sends a one-time email notice to the sender. The request must not include plaintext, decrypt keys, passphrases, or full URLs.

Response:

```json
{
  "notified": true
}
```

### 6.4 DELETE /api/secrets/:publicId

Deletes a secret using a delete token.

Request:

```json
{
  "deleteToken": "delete-token"
}
```

### 6.5 GET /api/stats/daily

Protected by admin token. Returns aggregate daily stats only.

### 6.6 GET /health

Returns service health.

## 7. Database Schema

```sql
CREATE TABLE secrets (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    encrypted_payload BYTEA NOT NULL,
    iv BYTEA NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'AES-256-GCM',
    version INT NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ NOT NULL,
    one_time BOOLEAN NOT NULL DEFAULT TRUE,
    consumed_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,
    sender_email_hash TEXT NOT NULL,
    recipient_email_provided BOOLEAN NOT NULL DEFAULT FALSE,
    manual_link_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    passphrase_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    delete_token_hash TEXT NULL,
    payload_size_bytes INT NOT NULL,
    notify_sender_on_reveal BOOLEAN NOT NULL DEFAULT FALSE,
    sender_notify_email TEXT NULL,
    sender_notified_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_secrets_public_id ON secrets(public_id);
CREATE INDEX idx_secrets_expires_at ON secrets(expires_at);

CREATE TABLE daily_stats (
    stat_date DATE PRIMARY KEY,
    secrets_created_count BIGINT NOT NULL DEFAULT 0,
    secrets_opened_count BIGINT NOT NULL DEFAULT 0,
    secrets_expired_count BIGINT NOT NULL DEFAULT 0,
    secrets_deleted_count BIGINT NOT NULL DEFAULT 0,
    emails_sent_count BIGINT NOT NULL DEFAULT 0,
    manual_links_created_count BIGINT NOT NULL DEFAULT 0,
    passphrase_enabled_count BIGINT NOT NULL DEFAULT 0,
    one_time_enabled_count BIGINT NOT NULL DEFAULT 0,
    files_uploaded_count BIGINT NOT NULL DEFAULT 0,
    total_encrypted_file_bytes BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 8. Environment Variables

```env
APP_BASE_URL=https://secret.example.com
DATABASE_URL=postgres://...
SES_REGION=ap-southeast-1
SES_FROM_EMAIL=no-reply@example.com
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
MAX_SECRET_BYTES=262144
MAX_EXPIRY_MINUTES=10080
DEFAULT_EXPIRY_MINUTES=1440
DEFAULT_ONE_TIME=true
ADMIN_STATS_TOKEN=change-me
LOG_LEVEL=info
RATE_LIMIT_CREATE_PER_HOUR=20
RATE_LIMIT_EMAIL_PER_HOUR=20
RECAPTCHA_SITE_KEY=...
RECAPTCHA_PROJECT_ID=...
RECAPTCHA_API_KEY=...
RECAPTCHA_MIN_SCORE=0.5
```

## 9. Logging Requirements

Logs may include event type, status code, generic error category, and timestamp. Logs must not include request body, plaintext, ciphertext unless needed for debugging and disabled in production, recipient email, full URLs, decrypt key, passphrase, or user agent fingerprint.

## 10. Rate Limiting

Rate limit by short-lived hashed keys. Use sender email hash and short-lived IP hash only where necessary for abuse protection. Avoid long-term IP retention.

## 11. Acceptance Criteria

- Plaintext is never sent to backend.
- Decrypt key appears only in URL fragment and frontend memory.
- Recipient email is not retained after SES send.
- One-time view is default.
- Expired secrets are inaccessible and purged.
- Stats are aggregate-only.
- Sender reveal notification is opt-in and clears the notification email after the one-time notice is claimed.
- Docker Compose deployment works locally and in Coolify.

## 12. Phase 2 SRS Notes

File sharing will add:

- S3 presigned upload URL endpoint
- S3 presigned download URL endpoint
- Browser-side chunk encryption
- file manifest metadata
- 1 GB max file size
- 7 day max retention
- lifecycle cleanup
