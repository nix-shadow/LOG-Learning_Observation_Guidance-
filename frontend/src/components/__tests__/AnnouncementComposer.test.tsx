import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import toast from 'react-hot-toast';
import AnnouncementComposer from '../admin/AnnouncementComposer';
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

describe('AnnouncementComposer', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockResolvedValue({});
  });

  it('blocks publishing until title and message are filled', async () => {
    const user = userEvent.setup();
    render(<AnnouncementComposer token="t" endpoint="/admin/announcements" />);

    await user.click(screen.getByRole('button', { name: /Publish/ }));
    expect(toast.error).toHaveBeenCalledWith('Title and message are required');
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it('publishes through the fetchWithCache seam with the given endpoint', async () => {
    const user = userEvent.setup();
    render(<AnnouncementComposer token="t" endpoint="/moderator/announcements" />);

    await user.type(screen.getByPlaceholderText('Announcement title'), 'Exam Week');
    await user.type(screen.getByPlaceholderText('Message for all students & teachers…'), 'Study hard!');
    await user.click(screen.getByRole('button', { name: /Publish/ }));

    expect(mockFetch).toHaveBeenCalledWith('/moderator/announcements', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ title: 'Exam Week', body: 'Study hard!' }),
    }));
    expect(toast.success).toHaveBeenCalledWith('Announcement published');
    expect((screen.getByPlaceholderText('Announcement title') as HTMLInputElement).value).toBe('');
  });

  it('surfaces publish failures honestly', async () => {
    const user = userEvent.setup();
    mockFetch.mockRejectedValue(new Error('offline'));
    render(<AnnouncementComposer token="t" endpoint="/admin/announcements" />);

    await user.type(screen.getByPlaceholderText('Announcement title'), 'Offline Test');
    await user.type(screen.getByPlaceholderText('Message for all students & teachers…'), 'queued later');
    await user.click(screen.getByRole('button', { name: /Publish/ }));

    await screen.findByText('Publish');
    expect(toast.error).toHaveBeenCalledWith('offline');
  });
});