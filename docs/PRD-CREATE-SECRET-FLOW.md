# Create Secret Flow PRD

Version: 1.0
Date: 18 June 2026

## Summary

Improve the create-secret workflow so successful creation focuses the sender on the generated share link, gives immediate copy feedback, validates required fields visibly, and supports an optional one-time sender notification after the recipient successfully reveals the secret.

## Goals

- Hide the create form after a secret is created.
- Show only the share-link result state after successful creation.
- Provide a clear "Create another secret" action from the result state.
- Give visible feedback when the secure link is copied.
- Require recipient email when "Send keyless email notice" is selected.
- Add basic field-level validation for required and invalid inputs.
- Add an opt-in "Notify sender when revealed" option.

## Non-Goals

- The backend must not receive plaintext secret text, decrypt keys, passphrases, or full URLs.
- Reveal notification is not a delivery/read receipt for email itself.
- Notification failures do not block secret reveal.

## Requirements

- Sender email is always required and must look like an email address.
- Secret text is always required.
- Recipient email becomes required only when keyless email notice is enabled.
- Passphrase must be at least 8 characters when passphrase mode is enabled.
- Copying the secure link must change the button or show nearby feedback.
- The browser reports reveal notification only after successful local decrypt.
- The backend sends at most one reveal notice per secret.
- The sender notification email is stored only when the sender opts in and is cleared after the notice is claimed.

## Acceptance Criteria

- Creating a valid secret hides the form and shows the secure link panel.
- "Create another secret" returns to the form with secret, passphrase, recipient, and generated state cleared.
- Missing sender email, recipient email when needed, secret text, or short passphrase shows field-level errors.
- Copying the secure link shows "Copied" or an equivalent success message.
- A successful recipient reveal calls the reveal endpoint without plaintext, key, passphrase, or full URL in the request.
- Existing databases are upgraded idempotently with the notification columns.
