import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import JoinClassCard from '../JoinClassCard';
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

describe('JoinClassCard (WP-1.5)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
  });

  it('joins with a valid code and shows the class name', async () => {
    mockFetch.mockResolvedValue({ id: 'cls-9', name: 'Grade 9 A' });
    render(<JoinClassCard />);

    fireEvent.change(screen.getByLabelText(/class invite code/i), { target: { value: 'abc123' } });
    fireEvent.click(screen.getByRole('button', { name: /join/i }));

    await waitFor(() => {
      expect(screen.getByText(/Grade 9 A/)).toBeInTheDocument();
    });
    expect(mockFetch).toHaveBeenCalledWith(
      '/classes/join',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ code: 'ABC123' }),
      })
    );
  });

  it('surfaces the honest not-found error for an unknown code', async () => {
    mockFetch.mockRejectedValue({ detail: 'No class found for this code' });
    render(<JoinClassCard />);

    fireEvent.change(screen.getByLabelText(/class invite code/i), { target: { value: 'ZZZ999' } });
    fireEvent.click(screen.getByRole('button', { name: /join/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('No class found for this code');
    });
  });

  it('does not call the API when the code is empty', async () => {
    render(<JoinClassCard />);
    fireEvent.click(screen.getByRole('button', { name: /join/i }));
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
    expect(mockFetch).not.toHaveBeenCalled();
  });
});