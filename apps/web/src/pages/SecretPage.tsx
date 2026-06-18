import { useEffect, useState } from 'react';
import { fetchSecret, SecretPayload } from '../lib/api';
import { decryptSecret, readFragmentKey } from '../lib/crypto';

export function SecretPage({ publicId }: { publicId: string }) {
  const [payload, setPayload] = useState<SecretPayload | null>(null);
  const [passphrase, setPassphrase] = useState('');
  const [secret, setSecret] = useState('');
  const [error, setError] = useState('');
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
    } catch {
      setError('Could not decrypt the secret. Check the link key and passphrase.');
    } finally {
      setBusy(false);
    }
  }

  async function copySecret() {
    await navigator.clipboard.writeText(secret);
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
              {secret ? <button className="button secondary" onClick={copySecret}>Copy</button> : null}
              {secret ? <button className="button secondary" onClick={() => setSecret('')}>Clear</button> : null}
            </div>
            {secret ? <pre className="secret-output">{secret}</pre> : null}
          </>
        ) : null}
        {!payload && error ? <div className="alert error">{error}</div> : null}
      </div>
    </section>
  );
}

