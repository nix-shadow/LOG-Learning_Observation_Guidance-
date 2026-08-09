import { fetchWithCache } from '../src/lib/api';

const mockFetch = jest.fn();
global.fetch = mockFetch;

jest.mock('idb', () => ({
  openDB: jest.fn().mockResolvedValue({
    get: jest.fn().mockResolvedValue({ test: 'data' }),
    put: jest.fn().mockResolvedValue(true),
    objectStoreNames: { contains: () => true },
  }),
}));

// Mock react-hot-toast so it doesn't complain in jest environment
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
    // Simulate being online by default
    window.dispatchEvent(new Event('online'));
  });

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
    // Simulate going offline
    window.dispatchEvent(new Event('offline'));

    const result = await fetchWithCache('/test-offline');
    // fetch should NOT be called if we know we are offline
    expect(mockFetch).not.toHaveBeenCalled();
    expect(result).toEqual({ test: 'data' });
  });
});
