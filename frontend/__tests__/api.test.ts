import { fetchWithCache } from '../src/lib/api';

const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock IDB with deduplication tracking
const mockQueueStore: any[] = [];

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
    put: jest.fn().mockResolvedValue(true),
    add: jest.fn().mockImplementation((_store: string, item: any) => {
      mockQueueStore.push(item);
      return Promise.resolve(mockQueueStore.length);
    }),
    getAll: jest.fn().mockImplementation(() => {
      return Promise.resolve([...mockQueueStore]);
    }),
    delete: jest.fn().mockResolvedValue(true),
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
});
