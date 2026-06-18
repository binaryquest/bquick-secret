export function HomePage() {
  return (
    <>
      <section className="hero">
        <div className="hero-copy">
          <p className="eyebrow">Browser-side encrypted secret sharing</p>
          <h1>bQuick Secret</h1>
          <p className="hero-subtitle">Encrypted in your browser. Readable only by the person with the link.</p>
          <p>
            Paste a password, API key, token, or private note. bQuick Secret encrypts it inside your browser before upload. Our server
            only stores encrypted data and cannot read your secret.
          </p>
          <p className="product-note">A free-to-use product of Binary Quest Limited.</p>
          <div className="hero-actions">
            <a className="button primary" href="/create">Create secret</a>
            <a className="button secondary" href="/how-it-works">How it works</a>
          </div>
        </div>
        <div className="privacy-panel" aria-label="Encryption flow">
          <div className="flow-step">
            <span>1</span>
            <strong>Plaintext stays here</strong>
            <small>Your browser encrypts before upload.</small>
          </div>
          <div className="flow-line" />
          <div className="flow-step">
            <span>2</span>
            <strong>Ciphertext is stored</strong>
            <small>The server receives unreadable data only.</small>
          </div>
          <div className="flow-line" />
          <div className="flow-step">
            <span>3</span>
            <strong>Recipient decrypts</strong>
            <small>The key lives in the URL fragment.</small>
          </div>
        </div>
      </section>

      <section className="feature-grid" aria-label="bQuick Secret protections">
        {[
          ['Browser-side encryption', 'Plaintext never leaves the device.'],
          ['Server cannot read it', 'Only ciphertext and metadata are stored temporarily.'],
          ['One-time view by default', 'Links can be consumed after first retrieval.'],
          ['Automatic expiry', 'Retention is capped at seven days.'],
          ['Optional passphrase', 'Require both a link fragment and a passphrase.'],
          ['Privacy-aware analytics', 'Page views are counted without secret IDs, URL fragments, or tracking cookies.']
        ].map(([title, body]) => (
          <article className="feature-card" key={title}>
            <h2>{title}</h2>
            <p>{body}</p>
          </article>
        ))}
      </section>
    </>
  );
}
