export const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api';

export type CreateSecretRequest = {
  senderEmail: string;
  recipientEmail?: string;
  encryptedPayload: string;
  iv: string;
  algorithm: 'AES-256-GCM';
  version: 1;
  expiresInMinutes: number;
  oneTime: boolean;
  passphraseEnabled: boolean;
  sendEmail: boolean;
  manualLink: boolean;
  notifyOnReveal: boolean;
  revealProof?: string;
  recaptchaToken?: string;
  wrappedKey?: string;
  wrappingIv?: string;
  kdfSalt?: string;
  kdfIterations?: number;
  kdfAlgorithm?: 'PBKDF2-SHA-256';
};

export type CreateSecretResponse = {
  publicId: string;
  deleteToken: string;
  emailSent: boolean;
};

export type SecretPayload = {
  encryptedPayload: string;
  iv: string;
  algorithm: 'AES-256-GCM';
  version: 1;
  oneTime: boolean;
  passphraseEnabled: boolean;
  wrappedKey?: string;
  wrappingIv?: string;
  kdfSalt?: string;
  kdfIterations?: number;
  kdfAlgorithm?: 'PBKDF2-SHA-256';
};

export async function createSecret(request: CreateSecretRequest): Promise<CreateSecretResponse> {
  const response = await fetch(`${apiBaseUrl}/secrets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request)
  });
  return parseResponse(response);
}

export async function fetchSecret(publicId: string): Promise<SecretPayload> {
  const response = await fetch(`${apiBaseUrl}/secrets/${encodeURIComponent(publicId)}`);
  return parseResponse(response);
}

export async function reportSecretRevealed(publicId: string, revealProof: string): Promise<void> {
  const response = await fetch(`${apiBaseUrl}/secrets/${encodeURIComponent(publicId)}/revealed`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ revealProof })
  });
  await parseResponse(response);
}

export async function deleteSecret(publicId: string, deleteToken: string): Promise<void> {
  const response = await fetch(`${apiBaseUrl}/secrets/${encodeURIComponent(publicId)}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ deleteToken })
  });
  await parseResponse(response);
}

async function parseResponse<T>(response: Response): Promise<T> {
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || 'Request failed');
  }
  return data as T;
}
