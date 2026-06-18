export function PrivacyPage() {
  return (
    <section className="content-page">
      <p className="eyebrow">Privacy</p>
      <h1>Designed to know as little as possible</h1>
      <p>
        bQuick Secret stores encrypted payloads, expiry metadata, one-time view state, delete-token hashes, and aggregate daily counters.
        It does not store plaintext secrets, decrypt keys, passphrases, full URLs, tracking cookies, or recipient email after sending.
      </p>
      <p>
        Email notices are sent without the decrypt key. The sender controls how the full secure link or fragment key is shared.
      </p>
      <p>
        If the sender chooses reveal notification, bQuick Secret temporarily stores the sender email as a notification target
        and clears it after the one-time reveal notice is claimed.
      </p>
    </section>
  );
}
