import React from 'react';
import { render, screen } from '@testing-library/react';
import toast from 'react-hot-toast';
import RosterOverview, { RosterData } from '../moderator/RosterOverview';
import { fetchWithCache } from '@/lib/api';

jest.mock('@/lib/api', () => ({
  fetchWithCache: jest.fn(),
}));

jest.mock('react-hot-toast', () => ({
  __esModule: true,
  default: Object.assign(jest.fn(), {
    success: jest.fn(),
    error: jest.fn(),
  }),
}));

const mockFetch = fetchWithCache as jest.Mock;

const sampleRoster: RosterData = {
  class_name: 'Logic 101',
  active_students: 12,
  needs_attention: 2,
  assignments_due: 3,
  roster: [
    { id: 'stu-1', name: 'Aisha Student', completion: 75, streak: 3, status: 'Active', last_active: '2026-08-18' },
    { id: 'stu-2', name: 'Bibek Rai', completion: 0, streak: 0, status: 'New', last_active: '—' },
  ],
};

describe('RosterOverview', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders real stats and roster rows from the payload', async () => {
    mockFetch.mockResolvedValue(sampleRoster);
    render(<RosterOverview token="t" />);

    expect(await screen.findByText('Aisha Student')).toBeInTheDocument();
    expect(screen.getByText('Bibek Rai')).toBeInTheDocument();
    expect(screen.getByText('75%')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    // '3' appears twice: the Assignments Due stat and Aisha's streak badge
    expect(screen.getAllByText('3').length).toBe(2);
  });

  it('renders honest zeros and an empty row when there is no data', async () => {
    mockFetch.mockResolvedValue({
      class_name: 'Logic 101',
      active_students: 0,
      needs_attention: 0,
      assignments_due: 0,
      roster: [],
    });
    render(<RosterOverview token="t" />);

    expect(await screen.findByText('No student data available yet. Reconnect to load the latest roster.')).toBeInTheDocument();
    const zeroCounts = screen.getAllByText('0');
    expect(zeroCounts.length).toBeGreaterThanOrEqual(3);
  });

  it('never renders fabricated students when the fetch fails', async () => {
    mockFetch.mockRejectedValue(new Error('offline'));
    render(<RosterOverview token="t" />);

    expect(await screen.findByText('No student data available yet. Reconnect to load the latest roster.')).toBeInTheDocument();
    expect(toast.error).toHaveBeenCalledWith('Failed to load roster');
    expect(screen.queryByText('Aisha Student')).not.toBeInTheDocument();
  });

  it('reports loaded data to the parent through onLoaded', async () => {
    mockFetch.mockResolvedValue(sampleRoster);
    const onLoaded = jest.fn();
    render(<RosterOverview token="t" onLoaded={onLoaded} />);

    await screen.findByText('Aisha Student');
    expect(onLoaded).toHaveBeenCalledWith(sampleRoster);
  });
});