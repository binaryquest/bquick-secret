# bQuick Secret Security Notes

## Core Invariant

The backend accepts and stores encrypted payloads only. It must never receive:

- plaintext secret text
- decrypt keys
- passphrases
- full URLs containing fragments
- recipient email after the send operation completes
- sender notification email after the reveal notice is claimed
- request bodies in logs

## Link Fragments

Decrypt material is stored after `#key=` in the browser URL. URL fragments are not sent to servers in HTTP requests, so the frontend composes final secure links only after the API returns a `publicId`.

## Email

SES email is intentionally keyless. It contains the `/s/{publicId}` route without `#key=...`. The sender must share the full secure link or fragment key through another channel. This preserves the zero-knowledge backend guarantee.

If the sender opts into reveal notification, the backend stores the sender email only as a notification target. After the browser reports a successful local decrypt, the backend claims the one-time notice and clears the stored notification email.

## Passphrase Mode

Passphrase mode encrypts the secret with a random data encryption key. That data key is wrapped in the browser with an AES-GCM key derived from:

- the user passphrase
- a random server-stored KDF salt
- the browser-only URL fragment key

This means neither the link alone nor the passphrase alone is enough to decrypt the secret.

## Logging

Allowed log fields are event type, status code, generic error category, and timestamp. Logs must not include request bodies, ciphertext, plaintext, recipient email, full URLs, user agent fingerprints, keys, or passphrases.

## Phase 2 Placeholder

Encrypted file sharing should use browser-side chunk encryption, S3 presigned PUT/GET URLs, private buckets, block public access, server-side encryption, lifecycle cleanup, and a 7 day maximum retention.
