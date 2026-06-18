import { useEffect, useState } from 'react';
import { fetchSecret, reportSecretRevealed, SecretPayload } from '../lib/api';
import { decryptSecret, readFragmentKey } from '../lib/crypto';

export function SecretPage({ publicId }: { publicId: string }) {
  const [payload, setPayload] = useState<SecretPayload | null>(null);
  const [passphrase, setPassphrase] = useState('');
  const [secret, setSecret] = useState('');
  const [error, setError] = useState('');
  const [copyFeedback, setCopyFeedback] = useState('');
  const [busy, setBusy] = useState(true);
  const fragmentKey = readFragmentKey();

  useEffect(() => {
    let cancelled = false;
    setBusy(true);
    fetchSecret(publicId)
      .then((data) => {
        if (!cancelled) {
          setPayload(data);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Secret not available');
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [publicId]);

  async function reveal() {
    if (!payload) return;
    setCopyFeedback('');
    if (!fragmentKey) {
      setError('The decrypt key is missing from the link.');
      return;
    }
    if (payload.passphraseEnabled && !passphrase) {
      setError('Enter the passphrase to decrypt this secret.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const plaintext = await decryptSecret({
        encryptedPayload: payload.encryptedPayload,
        iv: payload.iv,
        fragmentKey,
        passphrase: payload.passphraseEnabled ? passphrase : undefined,
        wrappedKey: payload.wrappedKey,
        wrappingIv: payload.wrappingIv,
        kdfSalt: payload.kdfSalt,
        kdfIterations: payload.kdfIterations
      });
      setSecret(plaintext);
      void reportSecretRevealed(publicId).catch(() => undefined);
    } catch {
      setError('Could not decrypt the secret. Check the link key and passphrase.');
    } finally {
      setBusy(false);
    }
  }

  async function copySecret() {
    setCopyFeedback('Copied');
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(secret);
      } else {
        fallbackCopy(secret);
      }
    } catch {
      try {
        fallbackCopy(secret);
      } catch {}
    }
  }

  return (
    <section className="workspace narrow">
      <div className="section-heading">
        <p className="eyebrow">Open</p>
        <h1>Decrypt secret</h1>
      </div>

      <div className="tool-surface">
        {busy && !payload ? <p className="muted">Loading encrypted payload...</p> : null}
        {payload ? (
          <>
            <p className="muted">
              The encrypted payload has been retrieved. Decryption happens locally in this browser.
            </p>
            {!fragmentKey ? <div className="alert error">The URL is missing `#key=...`.</div> : null}
            {payload.passphraseEnabled ? (
              <label className="field">
                <span className="field-label">Passphrase</span>
                <input value={passphrase} onChange={(event) => setPassphrase(event.target.value)} type="password" />
              </label>
            ) : null}
            {error ? <div className="alert error">{error}</div> : null}
            <div className="actions">
              <button className="button primary" disabled={busy || !payload} onClick={reveal}>{busy ? 'Working...' : 'Reveal'}</button>
              {secret ? <button className="button secondary" onClick={copySecret}>{copyFeedback === 'Copied' ? 'Copied' : 'Copy'}</button> : null}
              {secret ? <button className="button secondary" onClick={() => { setSecret(''); setCopyFeedback(''); }}>Clear</button> : null}
            </div>
            {copyFeedback ? <p className="muted" aria-live="polite">{copyFeedback}</p> : null}
            {secret ? <pre className="secret-output">{secret}</pre> : null}
          </>
        ) : null}
        {!payload && error ? <div className="alert error">{error}</div> : null}
      </div>
    </section>
  );
}

function fallbackCopy(value: string) {
  const textArea = document.createElement('textarea');
  textArea.value = value;
  textArea.setAttribute('readonly', '');
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';
  document.body.appendChild(textArea);
  textArea.select();
  const copied = document.execCommand('copy');
  document.body.removeChild(textArea);
  if (!copied) {
    throw new Error('copy failed');
  }
}
