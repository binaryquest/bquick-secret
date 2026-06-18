# bQuick Secret PRD

Version: 1.0  
Date: 17 June 2026

## 1. Product Summary

bQuick Secret is a public, privacy-first web application for sharing short secrets such as passwords, API keys, tokens, recovery codes, and confidential notes. Users paste a secret in the browser, where it is encrypted before upload. The backend stores only encrypted ciphertext and cannot read the plaintext.

The recipient can receive a link by email through Amazon SES, or the sender can manually copy and share the link. Secrets are one-time view by default and expire automatically, with a maximum retention of 7 days.

Phase 2 adds browser-encrypted file sharing up to 1 GB using direct S3 upload/download with presigned URLs.

## 2. Final Architecture Decision

- Frontend: React + Vite + TypeScript
- Crypto: Browser Web Crypto API
- Backend: Go
- Database: Postgres
- Email: Amazon SES
- Deployment: Docker Compose via Coolify
- Phase 2 Files: Browser-side chunk encryption + S3 presigned upload/download
- Stats: Daily aggregate counters plus privacy-aware Google Analytics page views

Strict rule: the backend must never receive plaintext secrets, decryption keys, passphrases, or full URLs containing URL fragments.

## 3. Goals

- Replace untrusted public secret-sharing websites used by team members.
- Make secret sharing simple enough for non-technical users.
- Ensure plaintext secrets never leave the browser.
- Store encrypted data temporarily, with automatic deletion.
- Require sender email, but avoid creating accounts.
- Avoid storing recipient email after sending.
- Provide manual copy-link sharing as well as email delivery.
- Provide optional sender notification when a secret is successfully revealed.
- Provide aggregate usage statistics without tracking users.
- Prepare for encrypted large file sharing in Phase 2.

## 4. Non-Goals

- User accounts or login.
- Team workspaces.
- Long-term vault/password manager features.
- Secret history or search.
- Admin ability to view secrets.
- Permanent storage.
- Session replay, advertising pixels, or analytics that collect secret IDs, URL fragments, or plaintext.
- Public API for external developers in MVP.

## 5. Target Users

- Internal team members who need to share passwords and credentials safely.
- Clients or vendors who need to send temporary secrets.
- Public users who want a privacy-focused alternative to unknown third-party secret-sharing sites.

## 6. Core User Stories

1. As a sender, I want to paste a secret and have it encrypted in my browser so the server cannot read it.
2. As a sender, I want to email the recipient a link so sharing is easy.
3. As a sender, I want to manually copy a link so I can share it through another channel.
4. As a sender, I want one-time view by default so the link cannot be reused repeatedly.
5. As a sender, I want optional passphrase protection so email compromise alone is not enough to reveal the secret.
6. As a recipient, I want to open the link and decrypt the secret in my browser.
7. As an operator, I want daily aggregate stats so I know the app is being used without tracking users.
8. As an operator, I want expired secrets deleted automatically.
9. As a sender, I want optional notification when the recipient reveals the secret so I know it was picked up.

## 7. MVP Features

### 7.1 Landing Page

The landing page must explain browser-side encryption, temporary storage, one-time view, expiry, optional passphrase, and no tracking.

Suggested hero copy:

> Share secrets safely. Encrypted in your browser. Readable only by the person with the link.

Suggested supporting copy:

> Paste a password, API key, token, or private note. bQuick Secret encrypts it inside your browser before upload. Our server only stores encrypted data and cannot read your secret.

### 7.2 Create Secret

Inputs:

- Sender email: mandatory
- Recipient email: optional when manual copy link is enabled
- Secret text: mandatory
- Expiry: 15 minutes, 1 hour, 24 hours, 3 days, 7 days
- One-time view: enabled by default
- Notify sender when revealed: optional, disabled by default
- Optional passphrase: supported from day one

### 7.3 Email Sending

If recipient email is provided, backend sends an email through Amazon SES. The recipient email is used only for sending and is not stored after the send operation.

### 7.4 Manual Link

After creation, the sender can copy the full secure link. The link includes the decrypt key in the URL fragment after `#`.

Example:

```text
https://secret.example.com/s/{publicId}#key={decryptKey}
```

### 7.5 Retrieve Secret

The recipient page retrieves encrypted payload by `publicId`. Browser JavaScript reads the key from the URL fragment and decrypts locally.

If sender notification is enabled, the browser reports a successful reveal only after local decryption succeeds. The backend sends a one-time email notice to the sender and must not receive plaintext, decrypt keys, passphrases, or full URLs.

### 7.6 Expiry and Deletion

Secrets expire automatically. Maximum retention is 7 days. One-time secrets are consumed after first payload retrieval. Manual deletion should be supported with a delete token.

### 7.7 Aggregate Stats

Store daily counts only: secrets created, opened, expired, deleted, emails sent, manual links created, passphrase-enabled secrets, one-time secrets, and later file uploads/bytes.

## 8. Privacy Requirements

The system may use Google Analytics for basic page-view measurement only if secret IDs are masked, URL fragments are never sent, ad personalization and Google signals are disabled, and tracking cookies/client-side storage are disabled.

The system must not store plaintext secret, decrypt key, passphrase, recipient email after sending, full URLs, referrer, browser fingerprint, or user behavior trail. If the sender opts into reveal notification, the sender email may be temporarily stored as a notification target and should be cleared after the one-time notification is claimed.

## 9. Security Requirements

- HTTPS only.
- HSTS enabled.
- Strong Content-Security-Policy.
- Referrer-Policy should avoid leaking URLs.
- No request body logging.
- No full URL logging.
- Rate limit create and email operations.
- Validate payload size and expiry.
- Store sender email as a hash for rate limiting and abuse protection.
- Postgres backups should be encrypted.
- Cleanup job must purge expired rows.

## 10. Phase 2 File Sharing

Phase 2 supports encrypted files up to 1 GB, retained for 7 days maximum. Files are encrypted in the browser using chunked encryption, uploaded directly to S3 using presigned URLs, and downloaded/decrypted in the recipient browser.

S3 bucket requirements:

- Private bucket.
- Block public access.
- Server-side encryption enabled.
- Lifecycle cleanup rule.
- Presigned PUT and GET only.

## 11. Success Criteria

- Team members stop using random public secret sharing sites.
- No plaintext appears in database, logs, emails, or server code.
- Users can create and open secrets without accounts.
- One-time and expiry behavior works reliably.
- Aggregate stats show adoption without invasive tracking.
- Phase 2 can be added without redesigning the system.
