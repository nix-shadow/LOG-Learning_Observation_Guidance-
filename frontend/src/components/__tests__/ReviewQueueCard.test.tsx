import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import ReviewQueueCard from '../ReviewQueueCard';
import { getAllReviewStates } from '@/lib/reviewStore';

jest.mock('@/lib/api', () => ({
  fetchWithCache: jest.fn(),
}));

jest.mock('@/lib/reviewStore', () => ({
  getAllReviewStates: jest.fn(),
}));

const mockAll = getAllReviewStates as jest.Mock;

describe('ReviewQueueCard (honesty — WP-1.4)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders an honest "nothing due" state when the schedule is empty', async () => {
    mockAll.mockResolvedValue([]);
    render(<ReviewQueueCard />);
    await waitFor(() => {
      expect(screen.getByText(/Nothing is due right now/i)).toBeInTheDocument();
    });
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('never lists an activity that is not due yet', async () => {
    const future = new Date(Date.now() + 86400000 * 3).toISOString().slice(0, 10);
    mockAll.mockResolvedValue([
      {
        activityId: 'act-9',
        ease: 2.5,
        intervalDays: 6,
        repetition: 2,
        lastReviewedAt: new Date().toISOString(),
        dueDate: future,
      },
    ]);
    render(<ReviewQueueCard />);
    await waitFor(() => {
      expect(screen.getByText(/Nothing is due right now/i)).toBeInTheDocument();
    });
    expect(screen.queryByText('act-9')).not.toBeInTheDocument();
  });

  it('lists a due activity from the real schedule with a due chip', async () => {
    const today = new Date().toISOString().slice(0, 10);
    mockAll.mockResolvedValue([
      {
        activityId: 'act-1',
        ease: 2.5,
        intervalDays: 1,
        repetition: 1,
        lastReviewedAt: new Date().toISOString(),
        dueDate: today,
      },
    ]);
    const api = jest.requireMock('@/lib/api') as { fetchWithCache: jest.Mock };
    api.fetchWithCache.mockResolvedValue({
      activities: [{ id: 'act-1', title: 'Logic Foundations' }],
    });
    render(<ReviewQueueCard />);
    await waitFor(() => {
      expect(screen.getByText('Logic Foundations')).toBeInTheDocument();
    });
    expect(screen.getByText(/Due today/i)).toBeInTheDocument();
  });
});