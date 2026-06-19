import { base64URLToBytes, bytesToBase64URL, joinBytes } from './base64url';

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const KEY_USAGE: KeyUsage[] = ['encrypt', 'decrypt'];
export const KDF_ITERATIONS = 310000;

export type EncryptedSecret = {
  encryptedPayload: string;
  iv: string;
  fragmentKey: string;
  wrappedKey?: string;
  wrappingIv?: string;
  kdfSalt?: string;
  kdfIterations?: number;
  kdfAlgorithm?: 'PBKDF2-SHA-256';
};

export async function encryptSecret(secretText: string, passphrase?: string): Promise<EncryptedSecret> {
  const dataKey = await crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, KEY_USAGE);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, dataKey, encoder.encode(secretText));
  const rawDataKey = new Uint8Array(await crypto.subtle.exportKey('raw', dataKey));

  if (!passphrase) {
    return {
      encryptedPayload: bytesToBase64URL(encrypted),
      iv: bytesToBase64URL(iv),
      fragmentKey: bytesToBase64URL(rawDataKey)
    };
  }

  const fragmentKey = crypto.getRandomValues(new Uint8Array(32));
  const kdfSalt = crypto.getRandomValues(new Uint8Array(16));
  const wrappingKey = await deriveWrappingKey(passphrase, fragmentKey, kdfSalt, KDF_ITERATIONS);
  const wrappingIv = crypto.getRandomValues(new Uint8Array(12));
  const wrapped = await crypto.subtle.encrypt({ name: 'AES-GCM', iv: wrappingIv }, wrappingKey, rawDataKey);

  return {
    encryptedPayload: bytesToBase64URL(encrypted),
    iv: bytesToBase64URL(iv),
    fragmentKey: bytesToBase64URL(fragmentKey),
    wrappedKey: bytesToBase64URL(wrapped),
    wrappingIv: bytesToBase64URL(wrappingIv),
    kdfSalt: bytesToBase64URL(kdfSalt),
    kdfIterations: KDF_ITERATIONS,
    kdfAlgorithm: 'PBKDF2-SHA-256'
  };
}

export async function decryptSecret(args: {
  encryptedPayload: string;
  iv: string;
  fragmentKey: string;
  passphrase?: string;
  wrappedKey?: string;
  wrappingIv?: string;
  kdfSalt?: string;
  kdfIterations?: number;
}): Promise<string> {
  let rawDataKey = base64URLToBytes(args.fragmentKey);

  if (args.wrappedKey) {
    if (!args.passphrase || !args.wrappingIv || !args.kdfSalt || !args.kdfIterations) {
      throw new Error('Passphrase is required');
    }
    const fragmentBytes = base64URLToBytes(args.fragmentKey);
    const wrappingKey = await deriveWrappingKey(args.passphrase, fragmentBytes, base64URLToBytes(args.kdfSalt), args.kdfIterations);
    const unwrapped = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: toArrayBuffer(base64URLToBytes(args.wrappingIv)) },
      wrappingKey,
      toArrayBuffer(base64URLToBytes(args.wrappedKey))
    );
    rawDataKey = new Uint8Array(unwrapped);
  }

  const key = await crypto.subtle.importKey('raw', toArrayBuffer(rawDataKey), { name: 'AES-GCM' }, false, ['decrypt']);
  const decrypted = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: toArrayBuffer(base64URLToBytes(args.iv)) },
    key,
    toArrayBuffer(base64URLToBytes(args.encryptedPayload))
  );
  return decoder.decode(decrypted);
}

export async function createRevealProof(fragmentKey: string): Promise<string> {
  const keyBytes = base64URLToBytes(fragmentKey);
  const prefix = encoder.encode('bquick-secret-reveal-v1:');
  const proof = await crypto.subtle.digest('SHA-256', toArrayBuffer(joinBytes(prefix, keyBytes)));
  return bytesToBase64URL(proof);
}

export function readFragmentKey(): string {
  const hash = new URLSearchParams(window.location.hash.replace(/^#/, ''));
  return hash.get('key') || '';
}

async function deriveWrappingKey(passphrase: string, fragmentKey: Uint8Array, salt: Uint8Array, iterations: number): Promise<CryptoKey> {
  const baseKey = await crypto.subtle.importKey('raw', encoder.encode(passphrase), 'PBKDF2', false, ['deriveKey']);
  return crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: toArrayBuffer(joinBytes(salt, fragmentKey)),
      iterations,
      hash: 'SHA-256'
    },
    baseKey,
    { name: 'AES-GCM', length: 256 },
    false,
    KEY_USAGE
  );
}

function toArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
}
