import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import toast from 'react-hot-toast';
import AuditLogTable from '../admin/AuditLogTable';
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

const sampleLogs = [
  { id: 1, user_id: 'admin-1', action: 'class.create', detail: 'cls-1 Grade 10 A', ip: '::1', created_at: '2026-08-18T10:00:00Z' },
  { id: 2, user_id: 'mod-1', action: 'assignment.create', detail: 'asg-1 class=cls-1', ip: '::1', created_at: '2026-08-18T11:00:00Z' },
];

describe('AuditLogTable', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders real audit entries', async () => {
    mockFetch.mockResolvedValue({ audit_logs: sampleLogs });
    render(<AuditLogTable token="t" />);
    expect(await screen.findByText('class.create')).toBeInTheDocument();
    expect(screen.getByText('assignment.create')).toBeInTheDocument();
    expect(screen.getByText('cls-1 Grade 10 A')).toBeInTheDocument();
  });

  it('renders an honest empty state when the audit trail has no entries', async () => {
    mockFetch.mockResolvedValue({ audit_logs: [] });
    render(<AuditLogTable token="t" />);
    expect(await screen.findByText('No audit entries yet.')).toBeInTheDocument();
  });

  it('surfaces load failures without inventing entries', async () => {
    mockFetch.mockRejectedValue(new Error('offline'));
    render(<AuditLogTable token="t" />);
    expect(await screen.findByText('No audit entries yet.')).toBeInTheDocument();
    expect(toast.error).toHaveBeenCalledWith('Failed to load audit log');
  });

  it('shows the export button only when a handler is provided', async () => {
    mockFetch.mockResolvedValue({ audit_logs: [] });
    const onExport = jest.fn();
    render(<AuditLogTable token="t" onExport={onExport} />);
    const user = userEvent.setup();
    await user.click(await screen.findByText(/Export Students CSV/));
    expect(onExport).toHaveBeenCalled();
  });
});