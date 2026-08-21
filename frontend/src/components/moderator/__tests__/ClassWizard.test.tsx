import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import ClassWizard from '../ClassWizard';
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

describe('ClassWizard (WP-1.5)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('creates a class and advances to the invite-code step', async () => {
    mockFetch.mockResolvedValue({ id: 'cls-9', invite_code: 'ABC123', name: 'Grade 9 A' });
    render(<ClassWizard token="t" onClose={() => {}} onCreated={() => {}} />);

    fireEvent.change(screen.getByPlaceholderText('Class name (e.g. Grade 10 A)'), { target: { value: 'Grade 9 A' } });
    fireEvent.change(screen.getByPlaceholderText('Grade'), { target: { value: '9' } });
    fireEvent.change(screen.getByPlaceholderText('Section'), { target: { value: 'A' } });
    fireEvent.click(screen.getByRole('button', { name: /create class/i }));

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });
    expect(mockFetch).toHaveBeenCalledWith(
      '/moderator/classes',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ name: 'Grade 9 A', grade: '9', section: 'A' }),
      })
    );
  });

  it('renders the honest import report: imported count, one-time passwords, per-row errors', async () => {
    mockFetch.mockResolvedValue({ id: 'cls-9', invite_code: 'ABC123' });
    const { container } = render(<ClassWizard token="t" onClose={() => {}} onCreated={() => {}} />);

    fireEvent.change(screen.getByPlaceholderText('Class name (e.g. Grade 10 A)'), { target: { value: 'Grade 9 A' } });
    fireEvent.change(screen.getByPlaceholderText('Grade'), { target: { value: '9' } });
    fireEvent.change(screen.getByPlaceholderText('Section'), { target: { value: 'A' } });
    fireEvent.click(screen.getByRole('button', { name: /create class/i }));
    await waitFor(() => expect(screen.getByText('ABC123')).toBeInTheDocument());

    mockFetch.mockResolvedValue({
      imported: 1,
      skipped: 1,
      passwords: { 'sita@test.local': 'Temp9xYz' },
      errors: [{ row: 3, email: 'bad@x', reason: 'missing name' }],
    });

    const file = new File(['name,email\nSita,sita@test.local'], 'roster.csv', { type: 'text/csv' });
    const input = container.querySelector('input[type=file]') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText(/Imported/)).toBeInTheDocument();
    });
    expect(screen.getByText(/sita@test.local/)).toBeInTheDocument();
    expect(screen.getByText(/Temp9xYz/)).toBeInTheDocument();
    expect(screen.getByText(/Row 3/)).toBeInTheDocument();
  });

  it('shows an honest error when the import fails', async () => {
    mockFetch.mockResolvedValue({ id: 'cls-9', invite_code: 'ABC123' });
    const { container } = render(<ClassWizard token="t" onClose={() => {}} onCreated={() => {}} />);

    fireEvent.change(screen.getByPlaceholderText('Class name (e.g. Grade 10 A)'), { target: { value: 'Grade 9 A' } });
    fireEvent.change(screen.getByPlaceholderText('Grade'), { target: { value: '9' } });
    fireEvent.change(screen.getByPlaceholderText('Section'), { target: { value: 'A' } });
    fireEvent.click(screen.getByRole('button', { name: /create class/i }));
    await waitFor(() => expect(screen.getByText('ABC123')).toBeInTheDocument());

    mockFetch.mockRejectedValue({ detail: 'could not parse CSV' });
    const file = new File(['bad'], 'roster.csv', { type: 'text/csv' });
    const input = container.querySelector('input[type=file]') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/Import failed/i));
    });
  });
});