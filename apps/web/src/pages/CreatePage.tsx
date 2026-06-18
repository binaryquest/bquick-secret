import { FormEvent, useMemo, useState } from 'react';
import { Field } from '../components/Field';
import { createSecret, deleteSecret } from '../lib/api';
import { encryptSecret } from '../lib/crypto';

const expiries = [
  ['15 minutes', 15],
  ['1 hour', 60],
  ['24 hours', 1440],
  ['3 days', 4320],
  ['7 days', 10080]
] as const;

type Result = {
  publicId: string;
  deleteToken: string;
  secureLink: string;
  emailSent: boolean;
  notifyOnReveal: boolean;
};

type FieldErrors = Partial<Record<'senderEmail' | 'recipientEmail' | 'secretText' | 'passphrase', string>>;

export function CreatePage() {
  const [senderEmail, setSenderEmail] = useState('');
  const [recipientEmail, setRecipientEmail] = useState('');
  const [secretText, setSecretText] = useState('');
  const [expiresInMinutes, setExpiresInMinutes] = useState(1440);
  const [oneTime, setOneTime] = useState(true);
  const [passphraseEnabled, setPassphraseEnabled] = useState(false);
  const [passphrase, setPassphrase] = useState('');
  const [sendEmail, setSendEmail] = useState(false);
  const [manualLink, setManualLink] = useState(true);
  const [notifyOnReveal, setNotifyOnReveal] = useState(false);
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [busy, setBusy] = useState(false);
  const [deleted, setDeleted] = useState(false);
  const [copyFeedback, setCopyFeedback] = useState('');

  const appBaseUrl = useMemo(() => import.meta.env.VITE_APP_BASE_URL || window.location.origin, []);

  function validateForm() {
    const nextErrors: FieldErrors = {};
    if (!looksLikeEmail(senderEmail)) {
      nextErrors.senderEmail = 'Enter a valid sender email.';
    }
    if (sendEmail && !looksLikeEmail(recipientEmail)) {
      nextErrors.recipientEmail = 'Recipient email is required for keyless email notice.';
    }
    if (!secretText.trim()) {
      nextErrors.secretText = 'Secret text is required.';
    }
    if (passphraseEnabled && passphrase.length < 8) {
      nextErrors.passphrase = 'Use at least 8 characters.';
    }
    setFieldErrors(nextErrors);
    return Object.keys(nextErrors).length === 0;
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError('');
    setDeleted(false);
    setCopyFeedback('');

    if (!validateForm()) {
      setError('Please fix the highlighted fields.');
      return;
    }

    setBusy(true);
    try {
      const encrypted = await encryptSecret(secretText, passphraseEnabled ? passphrase : undefined);
      const response = await createSecret({
        senderEmail: senderEmail.trim(),
        recipientEmail: recipientEmail.trim() || undefined,
        encryptedPayload: encrypted.encryptedPayload,
        iv: encrypted.iv,
        algorithm: 'AES-256-GCM',
        version: 1,
        expiresInMinutes,
        oneTime,
        passphraseEnabled,
        sendEmail,
        manualLink,
        notifyOnReveal,
        wrappedKey: encrypted.wrappedKey,
        wrappingIv: encrypted.wrappingIv,
        kdfSalt: encrypted.kdfSalt,
        kdfIterations: encrypted.kdfIterations,
        kdfAlgorithm: encrypted.kdfAlgorithm
      });
      const secureLink = `${appBaseUrl}/s/${response.publicId}#key=${encrypted.fragmentKey}`;
      setResult({ ...response, secureLink, notifyOnReveal });
      setSecretText('');
      setPassphrase('');
      setRecipientEmail('');
      setFieldErrors({});
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create the secret.');
    } finally {
      setBusy(false);
    }
  }

  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopyFeedback('Secure link copied.');
    } catch {
      setCopyFeedback('Could not copy automatically. Select the link and copy it manually.');
    }
  }

  async function onDelete() {
    if (!result) return;
    setBusy(true);
    setError('');
    try {
      await deleteSecret(result.publicId, result.deleteToken);
      setDeleted(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not delete the secret.');
    } finally {
      setBusy(false);
    }
  }

  function createAnother() {
    setResult(null);
    setDeleted(false);
    setError('');
    setFieldErrors({});
    setCopyFeedback('');
    setSecretText('');
    setPassphrase('');
    setRecipientEmail('');
  }

  return (
    <section className="workspace">
      <div className="section-heading">
        <p className="eyebrow">Create</p>
        <h1>Encrypt a secret in this browser</h1>
      </div>

      {result ? (
        <section className="result-panel" aria-live="polite">
          <h2>Secret created</h2>
          <p>The full secure link exists only in this browser. It includes the decrypt key after `#`.</p>
          <div className="copy-row">
            <input readOnly value={result.secureLink} aria-label="Secure secret link" />
            <button className="button secondary" onClick={() => copy(result.secureLink)}>
              {copyFeedback === 'Secure link copied.' ? 'Copied' : 'Copy'}
            </button>
          </div>
          {copyFeedback ? <div className="alert success">{copyFeedback}</div> : null}
          <p className="muted">Delete token: <code>{result.deleteToken}</code></p>
          <div className="actions">
            <button className="button danger" onClick={onDelete} disabled={busy || deleted}>{deleted ? 'Deleted' : 'Delete now'}</button>
            <button className="button secondary" onClick={createAnother}>Create another secret</button>
          </div>
          {sendEmail ? (
            <div className="alert notice">
              {result.emailSent
                ? 'The email notice was sent without the decrypt key. Share the full secure link or fragment key separately.'
                : 'The secret was created, but the email notice could not be sent. Copy the secure link instead.'}
            </div>
          ) : null}
          {result.notifyOnReveal ? (
            <div className="alert notice">
              The sender will be notified once after the recipient successfully reveals the secret.
            </div>
          ) : null}
          {error ? <div className="alert error">{error}</div> : null}
        </section>
      ) : (
        <form className="tool-surface" onSubmit={onSubmit} noValidate>
          <div className="form-grid">
            <Field label="Sender email" required error={fieldErrors.senderEmail}>
              <input
                value={senderEmail}
                onChange={(event) => setSenderEmail(event.target.value)}
                type="email"
                aria-invalid={fieldErrors.senderEmail ? 'true' : 'false'}
              />
            </Field>
            <Field
              label="Recipient email"
              required={sendEmail}
              error={fieldErrors.recipientEmail}
              hint={sendEmail ? 'Required for sending only, never stored after the email is sent.' : undefined}
            >
              <input
                value={recipientEmail}
                onChange={(event) => setRecipientEmail(event.target.value)}
                type="email"
                placeholder={sendEmail ? 'recipient@example.com' : 'Optional'}
                aria-invalid={fieldErrors.recipientEmail ? 'true' : 'false'}
              />
            </Field>
          </div>

          <Field label="Secret text" required error={fieldErrors.secretText}>
            <textarea
              value={secretText}
              onChange={(event) => setSecretText(event.target.value)}
              aria-invalid={fieldErrors.secretText ? 'true' : 'false'}
              rows={8}
            />
          </Field>

          <div className="form-grid">
            <Field label="Expiry">
              <select value={expiresInMinutes} onChange={(event) => setExpiresInMinutes(Number(event.target.value))}>
                {expiries.map(([label, value]) => (
                  <option value={value} key={value}>{label}</option>
                ))}
              </select>
            </Field>
            <div className="toggle-stack" aria-label="Secret options">
              <label><input type="checkbox" checked={oneTime} onChange={(event) => setOneTime(event.target.checked)} /> One-time view</label>
              <label><input type="checkbox" checked={manualLink} onChange={(event) => setManualLink(event.target.checked)} /> Manual copy link</label>
              <label><input type="checkbox" checked={sendEmail} onChange={(event) => setSendEmail(event.target.checked)} /> Send keyless email notice</label>
              <label><input type="checkbox" checked={notifyOnReveal} onChange={(event) => setNotifyOnReveal(event.target.checked)} /> Notify sender when revealed</label>
              <label><input type="checkbox" checked={passphraseEnabled} onChange={(event) => setPassphraseEnabled(event.target.checked)} /> Require passphrase</label>
            </div>
          </div>

          {passphraseEnabled ? (
            <Field label="Passphrase" required error={fieldErrors.passphrase} hint="The passphrase is used only in this browser and is never sent.">
              <input
                value={passphrase}
                onChange={(event) => setPassphrase(event.target.value)}
                type="password"
                autoComplete="new-password"
                aria-invalid={fieldErrors.passphrase ? 'true' : 'false'}
              />
            </Field>
          ) : null}

          {notifyOnReveal ? (
            <div className="alert notice">
              To send this notice, bQuick Secret stores the sender email only until the first successful reveal notification is claimed.
            </div>
          ) : null}

          {error ? <div className="alert error">{error}</div> : null}

          <div className="actions">
            <button className="button primary" disabled={busy}>{busy ? 'Working...' : 'Create encrypted secret'}</button>
          </div>
        </form>
      )}
    </section>
  );
}

function looksLikeEmail(value: string) {
  const trimmed = value.trim();
  return trimmed.length <= 254 && trimmed.includes('@') && trimmed.includes('.');
}
