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
};

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
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [deleted, setDeleted] = useState(false);

  const appBaseUrl = useMemo(() => import.meta.env.VITE_APP_BASE_URL || window.location.origin, []);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError('');
    setDeleted(false);
    setResult(null);

    if (!senderEmail.trim() || !secretText.trim()) {
      setError('Sender email and secret text are required.');
      return;
    }
    if (sendEmail && !recipientEmail.trim()) {
      setError('Recipient email is required when email delivery is enabled.');
      return;
    }
    if (passphraseEnabled && passphrase.length < 8) {
      setError('Use at least 8 characters for the passphrase.');
      return;
    }

    setBusy(true);
    try {
      const encrypted = await encryptSecret(secretText, passphraseEnabled ? passphrase : undefined);
      const response = await createSecret({
        senderEmail,
        recipientEmail: recipientEmail || undefined,
        encryptedPayload: encrypted.encryptedPayload,
        iv: encrypted.iv,
        algorithm: 'AES-256-GCM',
        version: 1,
        expiresInMinutes,
        oneTime,
        passphraseEnabled,
        sendEmail,
        manualLink,
        wrappedKey: encrypted.wrappedKey,
        wrappingIv: encrypted.wrappingIv,
        kdfSalt: encrypted.kdfSalt,
        kdfIterations: encrypted.kdfIterations,
        kdfAlgorithm: encrypted.kdfAlgorithm
      });
      const secureLink = `${appBaseUrl}/s/${response.publicId}#key=${encrypted.fragmentKey}`;
      setResult({ ...response, secureLink });
      setSecretText('');
      setPassphrase('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create the secret.');
    } finally {
      setBusy(false);
    }
  }

  async function copy(value: string) {
    await navigator.clipboard.writeText(value);
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

  return (
    <section className="workspace">
      <div className="section-heading">
        <p className="eyebrow">Create</p>
        <h1>Encrypt a secret in this browser</h1>
      </div>

      <form className="tool-surface" onSubmit={onSubmit}>
        <div className="form-grid">
          <Field label="Sender email">
            <input value={senderEmail} onChange={(event) => setSenderEmail(event.target.value)} type="email" required />
          </Field>
          <Field label="Recipient email" hint={sendEmail ? 'Used for sending only, never stored.' : undefined}>
            <input value={recipientEmail} onChange={(event) => setRecipientEmail(event.target.value)} type="email" placeholder="Optional" />
          </Field>
        </div>

        <Field label="Secret text">
          <textarea value={secretText} onChange={(event) => setSecretText(event.target.value)} required rows={8} />
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
            <label><input type="checkbox" checked={passphraseEnabled} onChange={(event) => setPassphraseEnabled(event.target.checked)} /> Require passphrase</label>
          </div>
        </div>

        {passphraseEnabled ? (
          <Field label="Passphrase" hint="The passphrase is used only in this browser and is never sent.">
            <input value={passphrase} onChange={(event) => setPassphrase(event.target.value)} type="password" autoComplete="new-password" />
          </Field>
        ) : null}

        {error ? <div className="alert error">{error}</div> : null}

        <div className="actions">
          <button className="button primary" disabled={busy}>{busy ? 'Working...' : 'Create encrypted secret'}</button>
        </div>
      </form>

      {result ? (
        <section className="result-panel" aria-live="polite">
          <h2>Secret created</h2>
          <p>The full secure link exists only in this browser. It includes the decrypt key after `#`.</p>
          <div className="copy-row">
            <input readOnly value={result.secureLink} />
            <button className="button secondary" onClick={() => copy(result.secureLink)}>Copy</button>
          </div>
          <p className="muted">Delete token: <code>{result.deleteToken}</code></p>
          <div className="actions">
            <button className="button danger" onClick={onDelete} disabled={busy || deleted}>{deleted ? 'Deleted' : 'Delete now'}</button>
          </div>
          {sendEmail ? (
            <div className="alert notice">
              {result.emailSent
                ? 'The email notice was sent without the decrypt key. Share the full secure link or fragment key separately.'
                : 'The secret was created, but the email notice could not be sent. Copy the secure link instead.'}
            </div>
          ) : null}
        </section>
      ) : null}
    </section>
  );
}
