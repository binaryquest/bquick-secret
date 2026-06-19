import { describe, expect, it } from 'vitest';
import type { CreateSecretRequest } from './api';

describe('create secret request shape', () => {
  it('does not include the browser-only fragment key', () => {
    const request: CreateSecretRequest = {
      senderEmail: 'sender@example.com',
      encryptedPayload: 'ciphertext',
      iv: 'iv',
      algorithm: 'AES-256-GCM',
      version: 1,
      expiresInMinutes: 60,
      oneTime: true,
      passphraseEnabled: false,
      sendEmail: false,
      manualLink: true,
      notifyOnReveal: false
    };

    expect(JSON.stringify(request)).not.toContain('fragmentKey');
    expect(JSON.stringify(request)).not.toContain('#key=');
  });

  it('allows a reveal proof without sending the browser-only fragment key', () => {
    const request: CreateSecretRequest = {
      senderEmail: 'sender@example.com',
      encryptedPayload: 'ciphertext',
      iv: 'iv',
      algorithm: 'AES-256-GCM',
      version: 1,
      expiresInMinutes: 60,
      oneTime: true,
      passphraseEnabled: false,
      sendEmail: false,
      manualLink: true,
      notifyOnReveal: true,
      revealProof: 'derived-proof'
    };

    expect(request.revealProof).toBe('derived-proof');
    expect(JSON.stringify(request)).not.toContain('fragmentKey');
    expect(JSON.stringify(request)).not.toContain('#key=');
  });

  it('allows a recaptcha token without changing key handling', () => {
    const request: CreateSecretRequest = {
      senderEmail: 'sender@example.com',
      encryptedPayload: 'ciphertext',
      iv: 'iv',
      algorithm: 'AES-256-GCM',
      version: 1,
      expiresInMinutes: 60,
      oneTime: true,
      passphraseEnabled: false,
      sendEmail: false,
      manualLink: true,
      notifyOnReveal: false,
      recaptchaToken: 'token'
    };

    expect(request.recaptchaToken).toBe('token');
    expect(JSON.stringify(request)).not.toContain('fragmentKey');
  });
});
