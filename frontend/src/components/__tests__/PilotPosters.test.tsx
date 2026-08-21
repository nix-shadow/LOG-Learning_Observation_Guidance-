import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PilotPosters from '../admin/PilotPosters';
import QRCode from 'qrcode';

jest.mock('qrcode', () => ({
  toDataURL: jest.fn().mockResolvedValue('data:image/png;base64,FAKEQR'),
}));

const sampleActivities = {
  activities: [
    { id: 'act-3', title: 'SEE Mathematics: Quadratic Equations', topic: 'Mathematics', order: 3 },
    { id: 'act-4', title: 'SEE Science: Electricity', topic: 'Science', order: 4 },
  ],
};

const sampleStats = {
  stats: {
    total_scans: 5,
    scans_today: 2,
    starts: 1,
    distinct_posters: 2,
    start_rate: 0.2,
    per_poster: [{ poster_id: 'act-3', scans: 4, starts: 1 }],
  },
};

const emptyStats = {
  stats: {
    total_scans: 0,
    scans_today: 0,
    starts: 0,
    distinct_posters: 0,
    start_rate: 0,
    per_poster: [],
  },
};

describe('PilotPosters (WP-3.3 RC-10)', () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    jest.clearAllMocks();
  });

  it('renders a QR poster per real activity and honest pilot numbers', async () => {
    global.fetch = jest.fn().mockImplementation((url: string) => {
      if (url.includes('/admin/pilot/stats')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(sampleStats) });
      }
      if (url.includes('/learning-journey')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(sampleActivities) });
      }
      return Promise.reject(new Error('unexpected fetch'));
    }) as unknown as typeof fetch;

    render(<PilotPosters token="t" />);

    expect(await screen.findByText('QR Pilot Posters')).toBeInTheDocument();
    expect(await screen.findByText('SEE Mathematics: Quadratic Equations')).toBeInTheDocument();
    expect(screen.getByText('SEE Science: Electricity')).toBeInTheDocument();
    expect(await screen.findByText('Total scans')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
    // Start rate rendered as a percentage derived from the real rows.
    expect(screen.getByText('20%')).toBeInTheDocument();
    expect(QRCode.toDataURL).toHaveBeenCalledTimes(2);
  });

  it('renders honest zeros when the pilot has no data yet — never invented numbers', async () => {
    global.fetch = jest.fn().mockImplementation((url: string) => {
      if (url.includes('/admin/pilot/stats')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(emptyStats) });
      }
      if (url.includes('/learning-journey')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(sampleActivities) });
      }
      return Promise.reject(new Error('unexpected fetch'));
    }) as unknown as typeof fetch;

    render(<PilotPosters token="t" />);

    expect(await screen.findByText('Total scans')).toBeInTheDocument();
    expect(screen.getAllByText('0').length).toBeGreaterThanOrEqual(4);
  });

  it('surfaces a load failure distinctly — no fabricated posters', async () => {
    global.fetch = jest.fn().mockRejectedValue(new Error('offline')) as unknown as typeof fetch;

    render(<PilotPosters token="t" />);

    expect(
      await screen.findByText('Poster list could not be loaded — check the connection and retry.')
    ).toBeInTheDocument();
    expect(screen.queryByText('SEE Mathematics: Quadratic Equations')).not.toBeInTheDocument();
  });

  it('refresh reloads real stats', async () => {
    let calls = 0;
    global.fetch = jest.fn().mockImplementation((url: string) => {
      if (url.includes('/admin/pilot/stats')) {
        calls += 1;
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve(
              calls === 1 ? emptyStats : { stats: { ...sampleStats.stats, total_scans: 9 } }
            ),
        });
      }
      if (url.includes('/learning-journey')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(sampleActivities) });
      }
      return Promise.reject(new Error('unexpected fetch'));
    }) as unknown as typeof fetch;

    render(<PilotPosters token="t" />);
    await screen.findByText('Total scans');
    await userEvent.click(screen.getByLabelText('Refresh pilot stats'));
    await waitFor(() => expect(screen.getByText('9')).toBeInTheDocument());
  });
});