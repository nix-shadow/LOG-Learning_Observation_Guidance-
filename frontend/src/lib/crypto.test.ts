import {
  bufToB64,
  b64ToBuf,
  encryptQueuePayload,
  decryptQueuePayload,
  fingerprint,
  disclosureHash,
  ensureQueueKey,
} from './crypto';
import { KEY_STORE } from './crypto';

// jest.setup.ts patches crypto.subtle with Node's webcrypto, so these tests
// exercise the real AES-GCM implementation, not a mock. Only IndexedDB is
// mocked (jsdom exposes no indexedDB here).
const mockKeyStore: Array<{ name: string; key: CryptoKey }> = [];

jest.mock('idb', () => ({
  openDB: jest.fn().mockResolvedValue({
    get: jest.fn().mockImplementation((store: string, key: string) => {
      if (store === 'sync-keys') {
        const found = mockKeyStore.find((k) => k.name === key);
        return Promise.resolve(found ? found.key : undefined);
      }
      return Promise.resolve(undefined);
    }),
    put: jest.fn().mockImplementation((store: string, item: unknown) => {
      if (store === 'sync-keys') {
        mockKeyStore.push({ name: 'active', key: item as CryptoKey });
        return Promise.resolve((item as { name?: string }).name);
      }
      return Promise.resolve(true);
    }),
    clear: jest.fn().mockResolvedValue(true),
    close: jest.fn(),
    objectStoreNames: {
      contains: (name: string) => name === 'sync-queue' || name === 'api-cache' || name === KEY_STORE,
    },
  }),
}));

describe('queue encryption (WP-0.1)', () => {
  beforeEach(() => {
    mockKeyStore.length = 0;
  });

  it('round-trips headers and body through encryption', async () => {
    const { enc, fp } = await encryptQueuePayload(
      '/activities/act-1/complete',
      'POST',
      { Authorization: 'Bearer token-123' },
      JSON.stringify({ score: 100 })
    );

    expect(enc).toBeTruthy();
    const plain = await decryptQueuePayload({ enc, headers: undefined, body: undefined });
    expect(plain).not.toBeNull();
    expect((plain!.headers as Record<string, string>).Authorization).toBe('Bearer token-123');
    expect(JSON.parse(plain!.body!)).toEqual({ score: 100 });
    expect(fp).toBeTruthy();
  });

  it('returns null for legacy records without an enc field (passthrough)', async () => {
    const plain = await decryptQueuePayload({
      headers: { Authorization: 'Bearer old' },
      body: '{"score":50}',
    });
    expect(plain).toEqual({ headers: { Authorization: 'Bearer old' }, body: '{"score":50}' });
  });

  it('returns null when the ciphertext is tampered with', async () => {
    const { enc } = await encryptQueuePayload('/activities/a', 'POST', null, '{"score":100}');
    expect(enc).toBeTruthy();

    // Flip one byte of the IV (not the ciphertext) — GCM authenticates it.
    const tamperedIv = b64ToBuf(enc!.iv);
    tamperedIv[0] ^= 0xff;
    const tampered = { ...enc!, iv: bufToB64(tamperedIv) };

    const plain = await decryptQueuePayload({ enc: tampered });
    expect(plain).toBeNull();
  });

  it('produces a stable fingerprint for identical actions and distinct ones otherwise', async () => {
    const a1 = await fingerprint('/activities/x/complete', 'POST', '{"score":100}');
    const a2 = await fingerprint('/activities/x/complete', 'POST', '{"score":100}');
    const b = await fingerprint('/activities/x/complete', 'POST', '{"score":99}');
    const c = await fingerprint('/activities/y/complete', 'POST', '{"score":100}');

    expect(a1).toBe(a2);
    expect(a1).not.toBe(b);
    expect(a1).not.toBe(c);
  });

  it('persists a single non-extractable AES-GCM-256 key', async () => {
    const key = await ensureQueueKey();
    expect(key.algorithm.name).toBe('AES-GCM');
    expect((key.algorithm as AesKeyAlgorithm).length).toBe(256);
    expect(key.extractable).toBe(false);
    expect(key.usages).toEqual(expect.arrayContaining(['encrypt', 'decrypt']));

    // A second call must return the persisted key, not a new one.
    const again = await ensureQueueKey();
    expect(again).toBe(key);
  });

  it('produces a 64-hex sha256 disclosure hash for the consent notice', async () => {
    const hash = await disclosureHash('Guardian Consent · अभिभावकको सहमति\nbilingual notice text');
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
    // Stable: identical notice -> identical hash.
    const again = await disclosureHash('Guardian Consent · अभिभावकको सहमति\nbilingual notice text');
    expect(again).toBe(hash);
  });

  it('falls back to an honestly-labeled djb2 hash without crypto.subtle', async () => {
    const subtle = globalThis.crypto.subtle;
    // @ts-expect-error simulating a non-secure context (plain-HTTP school LAN)
    globalThis.crypto.subtle = undefined;
    try {
      const hash = await disclosureHash('notice text');
      expect(hash).toMatch(/^djb2-[0-9a-f]+$/);
    } finally {
      globalThis.crypto.subtle = subtle;
    }
  });
});