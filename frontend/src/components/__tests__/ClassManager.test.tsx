import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import toast from 'react-hot-toast';
import ClassManager from '../admin/ClassManager';
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

const sampleClasses = [
  { id: 'cls-1', name: 'Grade 10 A', grade: '10', section: 'A', created_at: '2026-01-01', member_count: 2 },
  { id: 'cls-2', name: 'Grade 9 B', grade: '9', section: 'B', created_at: '2026-01-02', member_count: 0 },
];

const sampleUsers = [
  { id: 'mod-1', name: 'Teacher Edna', role: 'MODERATOR', email: 'edna@log.edu' },
  { id: 'stu-1', name: 'Aisha Student', role: 'STUDENT', email: 'aisha@log.edu' },
];

const renderManager = () => render(<ClassManager token="test-token" />);

describe('ClassManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockImplementation((url: string) => {
      if (url === '/admin/classes') return Promise.resolve({ classes: sampleClasses });
      if (url === '/admin/users') return Promise.resolve({ users: sampleUsers });
      return Promise.resolve({});
    });
  });

  it('renders the class table with real data', async () => {
    renderManager();
    expect(await screen.findByText('Grade 10 A')).toBeInTheDocument();
    expect(screen.getByText('Grade 9 B')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('renders an honest empty state when there are no classes', async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url === '/admin/classes') return Promise.resolve({ classes: [] });
      if (url === '/admin/users') return Promise.resolve({ users: sampleUsers });
      return Promise.resolve({});
    });
    renderManager();
    expect(await screen.findByText('No classes yet.')).toBeInTheDocument();
  });

  it('renders an honest note when no students are registered', async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url === '/admin/classes') return Promise.resolve({ classes: sampleClasses });
      if (url === '/admin/users') return Promise.resolve({ users: [] });
      return Promise.resolve({});
    });
    renderManager();
    expect(await screen.findByText('No students registered yet.')).toBeInTheDocument();
  });

  it('shows an error toast and honest empty table when the class fetch fails', async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url === '/admin/classes') return Promise.reject(new Error('offline'));
      if (url === '/admin/users') return Promise.resolve({ users: sampleUsers });
      return Promise.resolve({});
    });
    renderManager();
    expect(await screen.findByText('No classes yet.')).toBeInTheDocument();
    expect(toast.error).toHaveBeenCalledWith('Failed to load classes');
  });

  it('blocks class creation when a required field is missing', async () => {
    const user = userEvent.setup();
    renderManager();
    await screen.findByText('Grade 10 A');

    await user.click(screen.getByRole('button', { name: /Create Class/ }));
    expect(toast.error).toHaveBeenCalledWith('All class fields are required');
    expect(mockFetch).not.toHaveBeenCalledWith('/admin/classes', expect.objectContaining({ method: 'POST' }));
  });

  it('creates a class via the fetchWithCache seam and refreshes the list', async () => {
    const user = userEvent.setup();
    mockFetch.mockImplementation((url: string, opts?: { method?: string }) => {
      if (opts?.method === 'POST') return Promise.resolve({ id: 'cls-3', name: 'Grade 12 A' });
      if (url === '/admin/classes') return Promise.resolve({ classes: sampleClasses });
      if (url === '/admin/users') return Promise.resolve({ users: sampleUsers });
      return Promise.resolve({});
    });
    renderManager();
    await screen.findByText('Grade 10 A');

    await user.type(screen.getByPlaceholderText('Class name (e.g. Grade 10 A)'), 'Grade 12 A');
    await user.type(screen.getByPlaceholderText('Grade (e.g. 10)'), '12');
    await user.type(screen.getByPlaceholderText('Section (e.g. A)'), 'A');
    await user.selectOptions(screen.getAllByRole('combobox')[0], 'mod-1');
    await user.click(screen.getByRole('button', { name: /Create Class/ }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/admin/classes', expect.objectContaining({ method: 'POST' }));
    });
    expect(toast.success).toHaveBeenCalledWith('Class "Grade 12 A" created');
    expect(mockFetch).toHaveBeenCalledWith('/admin/classes', expect.not.objectContaining({ method: 'POST' }));
  });

  it('blocks enrollment until a class and students are selected', async () => {
    const user = userEvent.setup();
    renderManager();
    await screen.findByText('Grade 10 A');

    await user.click(screen.getByRole('button', { name: /Enroll Selected/ }));
    expect(toast.error).toHaveBeenCalledWith('Select a class and at least one student');
    expect(mockFetch).not.toHaveBeenCalledWith(
      expect.stringContaining('/enroll'),
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('enrolls selected students and shows the new member count', async () => {
    const user = userEvent.setup();
    mockFetch.mockImplementation((url: string, opts?: { method?: string }) => {
      if (opts?.method === 'POST' && url.includes('/enroll')) {
        return Promise.resolve({ member_count: 3 });
      }
      if (url === '/admin/classes') return Promise.resolve({ classes: sampleClasses });
      if (url === '/admin/users') return Promise.resolve({ users: sampleUsers });
      return Promise.resolve({});
    });
    renderManager();
    await screen.findByText('Grade 10 A');

    await user.selectOptions(screen.getAllByRole('combobox')[1], 'cls-1');
    await user.click(screen.getByText('Aisha Student'));
    await user.click(screen.getByRole('button', { name: /Enroll Selected/ }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/admin/classes/cls-1/enroll',
        expect.objectContaining({ method: 'POST' })
      );
    });
    expect(toast.success).toHaveBeenCalledWith('Enrolled! Class now has 3 students');
  });
});