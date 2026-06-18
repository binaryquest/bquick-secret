export function HowItWorksPage() {
  return (
    <section className="content-page">
      <p className="eyebrow">How it works</p>
      <h1>Plaintext never leaves the browser</h1>
      <ol className="steps">
        <li>The sender enters a secret and optional passphrase.</li>
        <li>The browser encrypts the secret with AES-GCM before upload.</li>
        <li>The API stores only ciphertext, IV, expiry, and safe metadata.</li>
        <li>The browser creates a secure link with the key after `#key=`.</li>
        <li>The recipient browser fetches ciphertext and decrypts locally.</li>
      </ol>
    </section>
  );
}

