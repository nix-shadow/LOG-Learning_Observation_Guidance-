import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import QRLanding from '@/app/qr/[activityId]/page';
import { fetchWithCache } from '@/lib/api';

jest.mock('@/lib/api', () => ({
  fetchWithCache: jest.fn(),
}));

// next/navigation in jsdom needs the real router shape for useParams/useRouter.
jest.mock('next/navigation', () => ({
  useParams: () => ({ activityId: 'act-4' }),
  useRouter: () => ({ push: jest.fn() }),
}));

const mockFetchWithCache = fetchWithCache as jest.Mock;

describe('QRLanding (WP-3.3 RC-10)', () => {
  const originalFetch = global.fetch;
  const originalLocation = window.location;

  afterEach(() => {
    global.fetch = originalFetch;
    jest.clearAllMocks();
    Object.defineProperty(window, 'location', { value: originalLocation, writable: true });
  });

  it('records the scan, warms the offline cache, and shows the lesson', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ scan_id: 42 }),
    }) as unknown as typeof fetch;
    mockFetchWithCache.mockImplementation((endpoint: string) => {
      if (endpoint.includes('/modules')) {
        return Promise.resolve({
          activity: { title: 'SEE Science: Electricity' },
          modules: [{ id: 'mm-37' }],
        });
      }
      return Promise.resolve({ activities: [] });
    });

    render(<QRLanding />);

    expect(await screen.findByText('SEE Science: Electricity')).toBeInTheDocument();
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/pilot/scans'),
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('act-4'),
      })
    );
    expect(mockFetchWithCache).toHaveBeenCalledWith('/activities/act-4/modules');
    // Offline cache warming also pulls the journey.
    expect(mockFetchWithCache).toHaveBeenCalledWith('/learning-journey', expect.anything());
  });

  it('marks the scan as started when the learner clicks through', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ scan_id: 7 }),
    }) as unknown as typeof fetch;
    mockFetchWithCache.mockResolvedValue({ activity: { title: 'T' }, modules: [{ id: 'x' }] });

    render(<QRLanding />);
    const startBtn = await screen.findByText('Start the lesson');
    await act(async () => {
      startBtn.click();
    });

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/pilot/scans/7/start'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  it('shows an honest offline state — the scan is not counted, no fabrication', async () => {
    global.fetch = jest.fn().mockRejectedValue(new TypeError('Failed to fetch')) as unknown as typeof fetch;
    mockFetchWithCache.mockResolvedValue({ activity: { title: 'SEE Science: Electricity' }, modules: [] });

    render(<QRLanding />);

    expect(await screen.findByText(/You are offline/)).toBeInTheDocument();
    expect(global.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining('/start'),
      expect.anything()
    );
  });
});