import { fetchWithCache } from '../src/lib/api';
import { decryptQueuePayload, KEY_STORE } from '../src/lib/crypto';

const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock IDB with deduplication tracking. Store-aware so the sync-keys store
// (queue encryption, WP-0.1) never pollutes the queue/cache mocks.
const mockQueueStore: any[] = [];
const mockKeyStore: any[] = [];
const mockDeletedKeys: string[] = [];

jest.mock('idb', () => ({
  openDB: jest.fn().mockResolvedValue({
    get: jest.fn().mockImplementation((store: string, key: string) => {
      if (store === 'sync-keys') {
        const found = mockKeyStore.find((k) => k.name === key);
        return Promise.resolve(found ? found.key : undefined);
      }
      // Return cached data with TTL metadata for TTL tests
      if (key === '/test-stale') {
        return { data: { stale: true }, cachedAt: Date.now() - (25 * 60 * 60 * 1000) }; // 25 hours old
      }
      if (key === '/test-fresh') {
        return { data: { fresh: true }, cachedAt: Date.now() - (1 * 60 * 60 * 1000) }; // 1 hour old
      }
      // Legacy format (no cachedAt) for backwards compat test
      if (key === '/test-legacy') {
        return { test: 'legacy-data' };
      }
      return { data: { test: 'data' }, cachedAt: Date.now() };
    }),
    add: jest.fn().mockImplementation((store: string, item: any) => {
      item.id = mockQueueStore.length + 1;
      mockQueueStore.push(item);
      return Promise.resolve(item.id);
    }),
    put: jest.fn().mockImplementation((store: string, item: any) => {
      if (store === 'sync-keys') {
        mockKeyStore.push({ name: 'active', key: item });
        return Promise.resolve(item.name);
      }
      const idx = mockQueueStore.findIndex((e) => e.id === item.id);
      if (idx >= 0) mockQueueStore[idx] = item;
      else mockQueueStore.push(item);
      return Promise.resolve(item.id ?? mockQueueStore.length);
    }),
    getAll: jest.fn().mockImplementation((store: string) => {
      return Promise.resolve(store === 'sync-keys' ? [...mockKeyStore] : [...mockQueueStore]);
    }),
    delete: jest.fn().mockImplementation((_store: string, key: string) => {
      mockDeletedKeys.push(key);
      return Promise.resolve(true);
    }),
    clear: jest.fn().mockResolvedValue(true),
    close: jest.fn(),
    objectStoreNames: { contains: (name: string) => name === 'api-cache' || name === 'sync-queue' || name === KEY_STORE },
  }),
}));

jest.mock('react-hot-toast', () => ({
  __esModule: true,
  default: Object.assign(jest.fn(), {
    success: jest.fn(),
    error: jest.fn(),
  }),
}));

describe('fetchWithCache', () => {
  beforeEach(() => {
    mockFetch.mockClear();
    mockQueueStore.length = 0;
    mockKeyStore.length = 0;
    window.dispatchEvent(new Event('online'));
  });

  // ---- Core Functionality ----

  it('attempts to fetch from network first when online', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: 'success' }),
    });

    const result = await fetchWithCache('/test');
    expect(mockFetch).toHaveBeenCalled();
    expect(result).toEqual({ status: 'success' });
  });

  it('falls back to cache when network fails', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const result = await fetchWithCache('/test');
    expect(mockFetch).toHaveBeenCalled();
    expect(result).toEqual({ test: 'data' });
  });

  it('uses cache immediately when offline', async () => {
    window.dispatchEvent(new Event('offline'));

    const result = await fetchWithCache('/test-offline');
    expect(mockFetch).not.toHaveBeenCalled();
    expect(result).toEqual({ test: 'data' });
  });

  // ---- Offline Mutation Queueing ----

  it('queues mutating POST requests when offline and returns optimistic 202', async () => {
    window.dispatchEvent(new Event('offline'));

    const result = await fetchWithCache('/activities/act-2/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 100 }),
    });

    expect(mockFetch).not.toHaveBeenCalled();
    expect(result).toEqual({ queued: true, status: 202 });
  });

  it('queues mutating POST requests when network fails', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network drop'));

    const result = await fetchWithCache('/activities/act-2/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 100 }),
    });

    expect(mockFetch).toHaveBeenCalled();
    expect(result).toEqual({ queued: true, status: 202 });
  });

  // ---- Queue Deduplication ----

  it('deduplicates identical queued requests', async () => {
    window.dispatchEvent(new Event('offline'));

    // First request queues normally
    const result1 = await fetchWithCache('/activities/act-5/complete', {
      method: 'POST',
    });
    expect(result1).toEqual({ queued: true, status: 202 });

    // Second identical request should be deduplicated
    const result2 = await fetchWithCache('/activities/act-5/complete', {
      method: 'POST',
    });
    expect(result2).toEqual({ queued: true, status: 202, deduplicated: true });
  });

  it('replaces an older queued submission with the newest payload (last write wins)', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/assignments/sub-1/submit', {
      method: 'POST',
      body: JSON.stringify({ note: 'first draft' }),
    });
    const result = await fetchWithCache('/assignments/sub-1/submit', {
      method: 'POST',
      body: JSON.stringify({ note: 'revised draft' }),
    });

    expect(result).toEqual({ queued: true, status: 202, deduplicated: true });
    expect(mockQueueStore).toHaveLength(1);
    const plain = await decryptQueuePayload(mockQueueStore[0]);
    expect(JSON.parse(plain!.body!)).toEqual({ note: 'revised draft' });
  });

  it('keeps the best-scoring completion when a better attempt is queued later', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-6/complete', {
      method: 'POST',
      body: JSON.stringify({ correct_count: 2, total_count: 4 }),
    });
    const result = await fetchWithCache('/activities/act-6/complete', {
      method: 'POST',
      body: JSON.stringify({ correct_count: 4, total_count: 4 }),
    });

    expect(result.deduplicated).toBe(true);
    expect(mockQueueStore).toHaveLength(1);
    const plain = await decryptQueuePayload(mockQueueStore[0]);
    expect(JSON.parse(plain!.body!)).toEqual({ correct_count: 4, total_count: 4 });
  });

  it('never downgrades a queued best completion with a lower-accuracy replay', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-7/complete', {
      method: 'POST',
      body: JSON.stringify({ correct_count: 4, total_count: 4 }),
    });
    const result = await fetchWithCache('/activities/act-7/complete', {
      method: 'POST',
      body: JSON.stringify({ correct_count: 1, total_count: 4 }),
    });

    expect(result.deduplicated).toBe(true);
    expect(mockQueueStore).toHaveLength(1);
    const plain = await decryptQueuePayload(mockQueueStore[0]);
    expect(JSON.parse(plain!.body!)).toEqual({ correct_count: 4, total_count: 4 });
  });

  it('keeps distinct queued actions separate (different endpoints never collide)', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-8/complete', { method: 'POST' });
    await fetchWithCache('/activities/act-8b/complete', { method: 'POST' });

    expect(mockQueueStore).toHaveLength(2);
  });

  // ---- Cache TTL ----

  it('returns fresh cached data within TTL', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network fail'));

    const result = await fetchWithCache('/test-fresh');
    expect(result).toEqual({ fresh: true });
  });

  it('handles legacy cache entries without cachedAt', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network fail'));

    const result = await fetchWithCache('/test-legacy');
    expect(result).toEqual({ test: 'legacy-data' });
  });

  // ---- Cache TTL invalidation ----

  it('rejects stale cache entries when online (TTL invalidation)', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network fail'));

    // The 25h-old entry must NOT be served: it is deleted and treated as a miss,
    // so the failing network path rejects instead of returning stale data.
    await expect(fetchWithCache('/test-stale')).rejects.toThrow();
    expect(mockDeletedKeys).toContain('/test-stale');
  });

  // ---- Queue flush on reconnect ----

  it('flushes the queued sync queue when the app comes back online', async () => {
    window.dispatchEvent(new Event('offline'));

    const queued = await fetchWithCache('/activities/act-9/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 100 }),
    });
    expect(queued).toEqual({ queued: true, status: 202 });

    mockFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    const syncedCall = mockFetch.mock.calls.find(([url]) =>
      String(url).includes('/activities/act-9/complete')
    );
    expect(syncedCall).toBeTruthy();
  });

  it('invalidates dashboard, learning-journey and chart-data caches after a completion sync', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-10/complete', { method: 'POST' });
    expect(mockQueueStore.length).toBeGreaterThan(0);

    mockDeletedKeys.length = 0;
    mockFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    expect(mockDeletedKeys).toContain('/dashboard');
    expect(mockDeletedKeys).toContain('/learning-journey');
    expect(mockDeletedKeys).toContain('/chart-data');
  });

  // ---- WP-0.1: queue encryption at rest ----

  it('stores queued payloads encrypted at rest', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-11/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 100 }),
      headers: { Authorization: 'Bearer test-token' },
    });

    const stored = mockQueueStore[0];
    expect(stored.body).toBeUndefined();
    expect(stored.headers).toBeUndefined();
    expect(stored.enc).toBeTruthy();
    expect(stored.enc.v).toBe(1);
    expect(stored.enc.alg).toBe('AES-GCM');
    expect(stored.enc.iv).toBeTruthy();
    expect(stored.enc.ct).toBeTruthy();
    // The ciphertext must not expose the plaintext or the auth token
    expect(stored.enc.ct).not.toContain('test-token');
    expect(stored.enc.ct).not.toContain('100');

    const plain = await decryptQueuePayload(stored);
    expect(JSON.parse(plain!.body!)).toEqual({ score: 100 });
    expect((plain!.headers as Record<string, string>).Authorization).toBe('Bearer test-token');
  });

  it('uses a fresh IV for every queued record', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/a/complete', { method: 'POST', body: 'x' });
    await fetchWithCache('/activities/b/complete', { method: 'POST', body: 'x' });

    expect(mockQueueStore).toHaveLength(2);
    expect(mockQueueStore[0].enc.iv).not.toBe(mockQueueStore[1].enc.iv);
  });

  it('flushes decrypted payloads to the server', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-12/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 100 }),
    });

    mockFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    const syncedCall = mockFetch.mock.calls.find(([url]) =>
      String(url).includes('/activities/act-12/complete')
    );
    expect(syncedCall).toBeTruthy();
    expect(JSON.parse(syncedCall![1].body)).toEqual({ score: 100 });
  });

  it('still flushes legacy plaintext records (v3 migration path)', async () => {
    // A record queued by an older build has no enc field.
    mockQueueStore.push({
      id: 1,
      endpoint: '/activities/act-13/complete',
      method: 'POST',
      headers: { Authorization: 'Bearer old-token' },
      body: JSON.stringify({ score: 50 }),
      timestamp: new Date().toISOString(),
      retryCount: 0,
    });

    mockFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    const syncedCall = mockFetch.mock.calls.find(([url]) =>
      String(url).includes('/activities/act-13/complete')
    );
    expect(syncedCall).toBeTruthy();
    expect(JSON.parse(syncedCall![1].body)).toEqual({ score: 50 });
  });

  // ---- WP-0.2 C1: queue-integrity regression guarantees ----

  it('KEEPS queued records on a 401 during flush (never deletes learner work)', async () => {
    window.dispatchEvent(new Event('offline'));

    await fetchWithCache('/activities/act-14/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 90 }),
    });
    expect(mockQueueStore).toHaveLength(1);

    mockFetch.mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ error: 'Unauthorized' }),
    });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    // The record survives the failed flush and is NOT deleted.
    expect(mockQueueStore).toHaveLength(1);
    const plain = await decryptQueuePayload(mockQueueStore[0]);
    expect(JSON.parse(plain!.body!)).toEqual({ score: 90 });
  });

  it('re-attaches the CURRENT token to queued records at flush time', async () => {
    window.dispatchEvent(new Event('offline'));

    // Queued under an old/expired token (or none at all).
    await fetchWithCache('/activities/act-15/complete', {
      method: 'POST',
      body: JSON.stringify({ score: 80 }),
      headers: { Authorization: 'Bearer expired-token' },
    });

    localStorage.setItem('log_token', 'fresh-token-after-relogin');
    mockFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    const syncedCall = mockFetch.mock.calls.find(([url]) =>
      String(url).includes('/activities/act-15/complete')
    );
    expect(syncedCall).toBeTruthy();
    expect(syncedCall![1].headers.Authorization).toBe('Bearer fresh-token-after-relogin');
    localStorage.removeItem('log_token');
  });

  it('invalidates dashboard, learning-journey and chart-data caches after a /sync/bulk flush', async () => {
    window.dispatchEvent(new Event('offline'));

    // Sneakernet import ships its actions through POST /sync/bulk.
    await fetchWithCache('/sync/bulk', {
      method: 'POST',
      body: JSON.stringify({ actions: [{ endpoint: '/activities/x/complete', method: 'POST' }] }),
    });
    expect(mockQueueStore.length).toBeGreaterThan(0);

    mockDeletedKeys.length = 0;
    mockFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

    window.dispatchEvent(new Event('online'));
    await new Promise((r) => setTimeout(r, 20));

    expect(mockDeletedKeys).toContain('/dashboard');
    expect(mockDeletedKeys).toContain('/learning-journey');
    expect(mockDeletedKeys).toContain('/chart-data');
  });
});
