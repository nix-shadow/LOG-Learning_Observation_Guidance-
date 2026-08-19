import { openDB } from 'idb';

/**
 * WP-0.1 — offline sync queue encryption at rest.
 *
 * IndexedDB stores plaintext on disk, so a stolen/lost device exposes queued
 * learner work (including Authorization headers). This module encrypts queue
 * payloads with AES-GCM-256 via WebCrypto:
 *
 *   - per-record random 12-byte IV (GCM's recommended nonce size)
 *   - the CryptoKey object is persisted in the `sync-keys` store (CryptoKey is
 *     structured-cloneable, so it survives reloads without ever leaving the
 *     device — this is why the key outlives logout, letting records queued
 *     under an expired session sync after re-login)
 *   - the key is non-extractable: it cannot be exported from the browser
 *   - endpoint/method/timestamp stay plaintext (routing + queue bookkeeping
 *     need them; they carry no learner data)
 *
 * Threat model (honest): this defends against casual/forensic device access —
 * it does NOT defend against same-origin XSS, which can read IDB anyway.
 *
 * When crypto.subtle is unavailable (non-secure context), records fall back
 * to plaintext with `enc: null` — an honest flag, never a silent upgrade.
 */

export const KEY_STORE = 'sync-keys';
const ACTIVE_KEY_NAME = 'active';

const enc = (input: string): Uint8Array => new TextEncoder().encode(input);
const dec = (bytes: Uint8Array): string => new TextDecoder().decode(bytes);

/** Returns a standalone ArrayBuffer copy (newer TS typed-array generics
 *  reject Uint8Array<ArrayBufferLike> where BufferSource is expected). */
const asBuffer = (bytes: Uint8Array): ArrayBuffer =>
  bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;

export const bufToB64 = (buf: ArrayBuffer | Uint8Array): string => {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let binary = '';
  // Chunked to avoid call-stack limits on large payloads.
  for (let i = 0; i < bytes.length; i += 0x8000) {
    const chunk = bytes.subarray(i, i + 0x8000);
    let part = '';
    for (let j = 0; j < chunk.length; j++) part += String.fromCharCode(chunk[j]);
    binary += part;
  }
  return btoa(binary);
};

export const b64ToBuf = (b64: string): Uint8Array => {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
};

export const cryptoSubtleAvailable = (): boolean =>
  typeof globalThis !== 'undefined' && !!globalThis.crypto && !!globalThis.crypto.subtle;

/**
 * Returns the queue encryption key, generating and persisting it on first use.
 * The key never leaves the device (non-extractable) and survives logins.
 */
export async function ensureQueueKey(): Promise<CryptoKey> {
  if (!cryptoSubtleAvailable()) {
    throw new Error('crypto.subtle unavailable — cannot encrypt queue');
  }
  const db = await openDB('log-db', 4, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(KEY_STORE)) {
        db.createObjectStore(KEY_STORE);
      }
    },
  });
  try {
    const existing = await db.get(KEY_STORE, ACTIVE_KEY_NAME);
    if (existing) return existing as CryptoKey;
    const key = await crypto.subtle.generateKey(
      { name: 'AES-GCM', length: 256 },
      false, // non-extractable: the key cannot be exported from the browser
      ['encrypt', 'decrypt']
    );
    await db.put(KEY_STORE, key, ACTIVE_KEY_NAME);
    return key;
  } finally {
    db.close();
  }
}

/**
 * SHA-256 fingerprint of a queued action. Used for queue deduplication of
 * encrypted records: the plaintext body is only ever compared in memory.
 * Endpoint/method are plaintext routing metadata, so their inclusion in the
 * fingerprint leaks nothing beyond what the record already stores.
 */
export async function fingerprint(
  endpoint: string,
  method: string,
  body: string | null | undefined
): Promise<string> {
  const material = `${method}|${endpoint}|${body ?? ''}`;
  if (!cryptoSubtleAvailable()) {
    // Non-cryptographic fallback (djb2) — only reached when the queue itself
    // is in plaintext fallback mode, so there is nothing to protect.
    let hash = 5381;
    for (let i = 0; i < material.length; i++) {
      hash = ((hash << 5) + hash + material.charCodeAt(i)) >>> 0;
    }
    return `djb2-${hash.toString(16)}`;
  }
  const digest = await crypto.subtle.digest('SHA-256', asBuffer(enc(material)));
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * SHA-256 hex of the exact consent notice text the learner was shown.
 * The backend stores this as disclosure_hash so the school can prove what
 * the guardian actually saw at consent time (COPPA 16 CFR §312.5 practice).
 * Non-secure contexts (plain-HTTP school LANs) get the documented `djb2-`
 * fallback — weaker, but honestly labeled, matching the queue's enc:null
 * fallback philosophy. Never blocks consent on a missing subtle.
 */
export async function disclosureHash(text: string): Promise<string> {
  if (!cryptoSubtleAvailable()) {
    let hash = 5381;
    for (let i = 0; i < text.length; i++) {
      hash = ((hash << 5) + hash + text.charCodeAt(i)) >>> 0;
    }
    return `djb2-${hash.toString(16)}`;
  }
  const digest = await crypto.subtle.digest('SHA-256', asBuffer(enc(text)));
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export interface EncryptedPayload {
  v: 1;
  alg: 'AES-GCM';
  iv: string; // base64, 12 bytes
  ct: string; // base64 ciphertext
}

export interface QueueCryptoResult {
  /** Present when encryption is active; null = honest plaintext fallback. */
  enc: EncryptedPayload | null;
  /** Dedup fingerprint of endpoint|method|body. */
  fp: string;
}

/** Normalizes any HeadersInit shape into a plain record. */
export const headersToRecord = (h: RequestInit['headers']): Record<string, string> => {
  if (!h) return {};
  if (h instanceof Headers) {
    const out: Record<string, string> = {};
    h.forEach((value, key) => {
      out[key] = value;
    });
    return out;
  }
  if (Array.isArray(h)) return Object.fromEntries(h as Array<[string, string]>);
  return { ...(h as Record<string, string>) };
};

/**
 * Encrypts the sensitive part of a queued request (headers — including the
 * Authorization token — plus the body). The plaintext exists only in memory.
 */
export async function encryptQueuePayload(
  endpoint: string,
  method: string,
  headers: RequestInit['headers'] | null,
  body: string | null | undefined
): Promise<QueueCryptoResult> {
  const fp = await fingerprint(endpoint, method, body);
  if (!cryptoSubtleAvailable()) {
    return { enc: null, fp };
  }
  const key = await ensureQueueKey();
  const plaintext = enc(JSON.stringify({ headers, body }));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, asBuffer(plaintext));
  return {
    enc: { v: 1, alg: 'AES-GCM', iv: bufToB64(iv), ct: bufToB64(ct) },
    fp,
  };
}

/**
 * Decrypts a queued record's payload. Returns null when the record is corrupt
 * or the key is missing (record is preserved — never deleted on decrypt
 * failure, because that would silently lose learner work).
 */
export async function decryptQueuePayload(
  record: { enc?: EncryptedPayload | null; headers?: RequestInit['headers']; body?: string | null }
): Promise<{ headers: RequestInit['headers']; body: string | null } | null> {
  if (!record.enc) {
    // Legacy or fallback plaintext record.
    return { headers: record.headers, body: record.body ?? null };
  }
  if (!cryptoSubtleAvailable()) {
    return null;
  }
  try {
    const key = await ensureQueueKey();
    const plaintext = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: asBuffer(b64ToBuf(record.enc.iv)) },
      key,
      asBuffer(b64ToBuf(record.enc.ct))
    );
    const parsed = JSON.parse(dec(new Uint8Array(plaintext))) as {
      headers?: RequestInit['headers'];
      body?: string | null;
    };
    return { headers: parsed.headers, body: parsed.body ?? null };
  } catch (e) {
    console.error('Failed to decrypt queued record — preserving it in the queue', e);
    return null;
  }
}

/** Wipes all locally stored user data: queue, cache, and the queue key. */
export async function wipeLocalData(): Promise<void> {
  const db = await openDB('log-db', 4, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(KEY_STORE)) {
        db.createObjectStore(KEY_STORE);
      }
    },
  });
  try {
    for (const store of ['api-cache', 'sync-queue', KEY_STORE]) {
      if (db.objectStoreNames.contains(store)) {
        await db.clear(store);
      }
    }
  } finally {
    db.close();
  }
}