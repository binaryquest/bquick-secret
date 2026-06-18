import { describe, expect, it } from 'vitest';
import { base64URLToBytes, bytesToBase64URL } from './base64url';

describe('base64url', () => {
  it('round trips bytes without padding', () => {
    const input = new Uint8Array([0, 1, 2, 250, 251, 252, 253, 254, 255]);
    const encoded = bytesToBase64URL(input);
    expect(encoded).not.toContain('=');
    expect(Array.from(base64URLToBytes(encoded))).toEqual(Array.from(input));
  });
});

