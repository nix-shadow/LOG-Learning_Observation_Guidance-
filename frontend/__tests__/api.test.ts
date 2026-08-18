import { fetchWithCache } from '../src/lib/api';

const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock IDB with deduplication tracking
const mockQueueStore: any[] = [];
const mockDeletedKeys: string[] = [];

jest.mock('idb', () => ({
  openDB: jest.fn().mockResolvedValue({
    get: jest.fn().mockImplementation((_store: string, key: string) => {
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
    add: jest.fn().mockImplementation((_store: string, item: any) => {
      item.id = mockQueueStore.length + 1;
      mockQueueStore.push(item);
      return Promise.resolve(item.id);
    }),
    put: jest.fn().mockImplementation((_store: string, item: any) => {
      const idx = mockQueueStore.findIndex((e) => e.id === item.id);
      if (idx >= 0) mockQueueStore[idx] = item;
      else mockQueueStore.push(item);
      return Promise.resolve(item.id ?? mockQueueStore.length);
    }),
    getAll: jest.fn().mockImplementation(() => {
      return Promise.resolve([...mockQueueStore]);
    }),
    delete: jest.fn().mockImplementation((_store: string, key: string) => {
      mockDeletedKeys.push(key);
      return Promise.resolve(true);
    }),
    objectStoreNames: { contains: () => true },
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
    expect(JSON.parse(mockQueueStore[0].body)).toEqual({ note: 'revised draft' });
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
    expect(JSON.parse(mockQueueStore[0].body)).toEqual({ correct_count: 4, total_count: 4 });
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
    expect(JSON.parse(mockQueueStore[0].body)).toEqual({ correct_count: 4, total_count: 4 });
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
});
