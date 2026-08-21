import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import SupportInbox from '../SupportInbox';
import { fetchWithCache } from '@/lib/api';
import toast from 'react-hot-toast';

jest.mock('@/lib/api', () => ({
  fetchWithCache: jest.fn(),
}));

jest.mock('react-hot-toast', () => ({
  __esModule: true,
  default: Object.assign(jest.fn(), { success: jest.fn(), error: jest.fn() }),
}));

const mockFetch = fetchWithCache as jest.Mock;

const escalatedIssue = {
  id: 'iss-1',
  user_id: 'stu-9',
  category: 'connectivity',
  description: 'The app shows an error every time I open a lesson on the school tablet.',
  escalated: true,
  status: 'open',
  created_at: '2026-08-20T08:00:00.000Z',
};

describe('SupportInbox (WP-2.2)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
  });

  it('shows an honest empty inbox when nothing is escalated', async () => {
    mockFetch.mockResolvedValue({ issues: [] });
    render(<SupportInbox />);
    await waitFor(() => {
      expect(screen.getByText(/Nothing needs attention right now/)).toBeInTheDocument();
    });
    expect(mockFetch).toHaveBeenCalledWith('/support/inbox');
  });

  it('lists open escalated issues only (self-served issues never appear)', async () => {
    mockFetch.mockResolvedValue({ issues: [escalatedIssue] });
    render(<SupportInbox />);
    await waitFor(() => {
      expect(screen.getByText(/Internet not working/)).toBeInTheDocument();
    });
    expect(screen.getByText(/app shows an error every time/i)).toBeInTheDocument();
  });

  it('resolving requires a note and then removes the issue from the inbox', async () => {
    mockFetch.mockResolvedValueOnce({ issues: [escalatedIssue] });
    render(<SupportInbox />);

    await waitFor(() => {
      expect(screen.getByText(/Internet not working/)).toBeInTheDocument();
    });

    // No note yet — resolving must be refused.
    fireEvent.click(screen.getByRole('button', { name: /resolve/i }));
    expect(toast.error).toHaveBeenCalledWith('Write a short resolution note first.');
    expect(mockFetch).toHaveBeenCalledTimes(1);

    // With a note, the PUT fires and the issue leaves the inbox.
    mockFetch.mockResolvedValueOnce({ ...escalatedIssue, status: 'resolved' });
    fireEvent.change(screen.getByPlaceholderText(/Resolution note/), {
      target: { value: 'Restarted the tablet and it works now.' },
    });
    fireEvent.click(screen.getByRole('button', { name: /resolve/i }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/support/issue/iss-1',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ resolution_note: 'Restarted the tablet and it works now.' }),
        })
      );
    });
    await waitFor(() => {
      expect(screen.getByText(/Nothing needs attention right now/)).toBeInTheDocument();
    });
  });
});