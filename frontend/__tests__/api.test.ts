import { fetchWithCache } from '../src/lib/api';

// Mock IndexedDB and fetch for testing
const mockFetch = jest.fn();
global.fetch = mockFetch;

jest.mock('idb', () => ({
  openDB: jest.fn().mockResolvedValue({
    get: jest.fn().mockResolvedValue({ test: 'data' }),
    put: jest.fn().mockResolvedValue(true),
    objectStoreNames: { contains: () => true },
  }),
}));

describe('fetchWithCache', () => {
  beforeEach(() => {
    mockFetch.mockClear();
  });

  it('attempts to fetch from network first when online', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: 'success' }),
    });

    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true });

    const result = await fetchWithCache('/test');
    expect(mockFetch).toHaveBeenCalled();
    expect(result).toEqual({ status: 'success' });
  });

  it('falls back to cache when network fails', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true });

    const result = await fetchWithCache('/test');
    expect(mockFetch).toHaveBeenCalled();
    expect(result).toEqual({ test: 'data' });
  });

  it('uses cache immediately when offline', async () => {
    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });

    const result = await fetchWithCache('/test');
    // fetch should NOT be called if we know we are offline
    expect(mockFetch).not.toHaveBeenCalled();
    expect(result).toEqual({ test: 'data' });
  });
});
