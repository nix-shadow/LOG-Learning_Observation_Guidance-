import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import toast from 'react-hot-toast';
import AssignmentManager from '../moderator/AssignmentManager';
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

const sampleAssignments = [
  {
    id: 'asg-1',
    class_id: 'cls-1',
    title: 'Homework 1',
    description: 'Solve exercises 1-5',
    due_date: '2026-09-01T00:00:00Z',
    created_at: '2026-08-18',
    submissions: 2,
  },
  {
    id: 'asg-2',
    class_id: 'cls-1',
    title: 'Homework 2',
    description: '',
    created_at: '2026-08-18',
    submissions: 0,
  },
];

describe('AssignmentManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockImplementation((url: string) => {
      if (url.includes('/submissions')) return Promise.resolve({ submissions: [] });
      if (url.includes('/assignments')) return Promise.resolve({ assignments: sampleAssignments });
      return Promise.resolve({});
    });
  });

  it('renders the assignments for the selected class', async () => {
    render(<AssignmentManager token="t" classId="cls-1" className="Grade 10 A" />);
    expect(await screen.findByText('Homework 1')).toBeInTheDocument();
    expect(screen.getByText('Homework 2')).toBeInTheDocument();
    expect(screen.getByText('— Grade 10 A')).toBeInTheDocument();
  });

  it('renders an honest empty state when the class has no assignments', async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes('/assignments')) return Promise.resolve({ assignments: [] });
      return Promise.resolve({});
    });
    render(<AssignmentManager token="t" classId="cls-1" className="Grade 10 A" />);
    expect(await screen.findByText('No assignments for this class yet.')).toBeInTheDocument();
  });

  it('blocks creation without a title', async () => {
    const user = userEvent.setup();
    render(<AssignmentManager token="t" classId="cls-1" className="Grade 10 A" />);
    await screen.findByText('Homework 1');

    await user.click(screen.getByRole('button', { name: /Create Assignment/ }));
    expect(toast.error).toHaveBeenCalledWith('Select a class and enter a title');
    expect(mockFetch).not.toHaveBeenCalledWith(
      expect.stringContaining('/assignments'),
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('creates an assignment and refreshes the list', async () => {
    const user = userEvent.setup();
    mockFetch.mockImplementation((url: string, opts?: { method?: string }) => {
      if (opts?.method === 'POST' && url.includes('/assignments')) return Promise.resolve({ id: 'asg-3' });
      if (url.includes('/submissions')) return Promise.resolve({ submissions: [] });
      if (url.includes('/assignments')) return Promise.resolve({ assignments: sampleAssignments });
      return Promise.resolve({});
    });
    render(<AssignmentManager token="t" classId="cls-1" className="Grade 10 A" />);
    await screen.findByText('Homework 1');

    await user.type(screen.getByPlaceholderText('Assignment title (e.g. Homework 1)'), 'Homework 3');
    await user.click(screen.getByRole('button', { name: /Create Assignment/ }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/moderator/classes/cls-1/assignments',
        expect.objectContaining({ method: 'POST', body: JSON.stringify({ title: 'Homework 3', description: '' }) })
      );
    });
    expect(toast.success).toHaveBeenCalledWith('Assignment created');
  });

  it('shows submissions and an honest empty state inside the panel', async () => {
    const user = userEvent.setup();
    mockFetch.mockImplementation((url: string) => {
      if (url.includes('/submissions')) {
        return url.includes('asg-1')
          ? Promise.resolve({
              submissions: [
                { id: 'sub-1', assignment_id: 'asg-1', learner_id: 'learner-123', note: 'My answers', submitted_at: '2026-08-18T10:00:00Z' },
              ],
            })
          : Promise.resolve({ submissions: [] });
      }
      if (url.includes('/assignments')) return Promise.resolve({ assignments: sampleAssignments });
      return Promise.resolve({});
    });
    render(<AssignmentManager token="t" classId="cls-1" className="Grade 10 A" />);
    await screen.findByText('Homework 1');

    await user.click(screen.getAllByText('View Submissions')[0]);
    expect(await screen.findByText('My answers')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Close/ }));
    await user.click(screen.getAllByText('View Submissions')[1]);
    expect(await screen.findByText('No submissions yet.')).toBeInTheDocument();
  });
});