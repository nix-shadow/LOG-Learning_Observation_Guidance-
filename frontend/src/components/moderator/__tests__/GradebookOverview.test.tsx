import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import GradebookOverview from '../GradebookOverview';
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

const gradebook = {
  students: [
    {
      student_id: 'stu-1',
      name: 'Aisha Student',
      rows: [
        { activity_id: 'act-1', title: 'Logic Basics', topic: 'Logic', status: 'completed', accuracy: 0.9, attempts: 2 },
        { activity_id: 'act-2', title: 'Sets & Venn', topic: 'Math', status: 'not-started', accuracy: 0, attempts: 0 },
      ],
    },
  ],
};

describe('GradebookOverview (WP-2.3)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    global.fetch = jest.fn();
  });

  it('renders real accuracy/attempts and an honest "Not yet assessed" for zero attempts', async () => {
    mockFetch.mockResolvedValue(gradebook);
    render(<GradebookOverview token="tok" selectedClass="cls-1" />);

    await waitFor(() => {
      expect(screen.getByText('Aisha Student')).toBeInTheDocument();
    });

    expect(mockFetch).toHaveBeenCalledWith('/moderator/gradebook?class_id=cls-1');

    // Real stored data: 90% · 2×
    expect(screen.getByText('90% · 2×')).toBeInTheDocument();
    // Honest empty state — never an invented grade
    expect(screen.getByText('Not yet assessed')).toBeInTheDocument();
  });

  it('exports the CSV from the backend endpoint with the auth token', async () => {
    mockFetch.mockResolvedValue(gradebook);
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(new Blob(['student_id,student_name'], { type: 'text/csv' })),
    });
    // jsdom has no URL.createObjectURL — stub it for the download path.
    URL.createObjectURL = jest.fn(() => 'blob:mock');
    URL.revokeObjectURL = jest.fn();
    const clickSpy = jest.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});

    render(<GradebookOverview token="tok" selectedClass="cls-1" />);
    await waitFor(() => {
      expect(screen.getByText('Aisha Student')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /export csv/i }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/moderator/gradebook.csv?class_id=cls-1'),
        expect.objectContaining({
          headers: { Authorization: 'Bearer tok' },
        })
      );
    });
    expect(toast.success).toHaveBeenCalledWith('Gradebook exported — real data only.', { icon: '📄' });
    clickSpy.mockRestore();
  });

  it('opens a note editor with an honest empty state and saves via PUT', async () => {
    // First call: gradebook; second: GET note -> null (honest: no note yet)
    mockFetch.mockResolvedValueOnce(gradebook).mockResolvedValueOnce({ note: null });
    render(<GradebookOverview token="tok" selectedClass="cls-1" />);

    await waitFor(() => {
      expect(screen.getByText('Aisha Student')).toBeInTheDocument();
    });

    // Sticky-note toggle button
    const noteButtons = screen.getAllByRole('button');
    fireEvent.click(noteButtons[noteButtons.length - 1]);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/moderator/students/stu-1/note');
    });

    // Note editor opens empty — null note, not a placeholder string.
    const textarea = screen.getByPlaceholderText(/Supportive note for this student/);
    expect((textarea as HTMLTextAreaElement).value).toBe('');

    mockFetch.mockResolvedValueOnce({ note: 'Great focus this week — keep it up!', updated_at: '2026-08-20T10:00:00.000Z' });
    fireEvent.change(textarea, { target: { value: 'Great focus this week — keep it up!' } });
    fireEvent.click(screen.getByRole('button', { name: /save note/i }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/moderator/students/stu-1/note',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ note: 'Great focus this week — keep it up!' }),
        })
      );
    });
  });
});