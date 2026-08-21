import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import ReconnectDigest from '../ReconnectDigest';
import { getReconnectDigest, clearReconnectDigest } from '@/lib/api';

// Keep the localStorage-backed helpers REAL (the card reads/writes them),
// but stub the network flush — the real module boots flushSyncQueue on the
// window 'load' event, which would hit IndexedDB (absent in jsdom).
jest.mock('@/lib/api', () => {
  const actual = jest.requireActual('@/lib/api');
  return {
    ...actual,
    fetchWithCache: jest.fn(),
    flushSyncQueue: jest.fn().mockResolvedValue({ synced: 0, failed: 0 }),
  };
});

describe('ReconnectDigest (WP-2.4)', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.restoreAllMocks();
  });

  it('renders nothing when no reconnect digest exists (no fabricated state)', () => {
    render(<ReconnectDigest />);
    expect(screen.queryByText(/Back online/)).not.toBeInTheDocument();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('shows an honest digest recorded before mount (e.g. boot flush)', () => {
    localStorage.setItem(
      'log_reconnect_digest',
      JSON.stringify({ synced: 3, failed: 1, at: '2026-08-20T09:00:00.000Z' })
    );
    render(<ReconnectDigest />);
    expect(screen.getByText('Back online — some changes are still waiting')).toBeInTheDocument();
    expect(screen.getByText(/3/)).toBeInTheDocument();
    expect(screen.getByText(/1/)).toBeInTheDocument();
  });

  it('appears when the log:digest-ready event fires (flush just finished)', () => {
    render(<ReconnectDigest />);
    expect(screen.queryByRole('status')).not.toBeInTheDocument();

    localStorage.setItem(
      'log_reconnect_digest',
      JSON.stringify({ synced: 2, failed: 0, at: '2026-08-20T09:05:00.000Z' })
    );
    act(() => {
      window.dispatchEvent(new Event('log:digest-ready'));
    });

    expect(screen.getByText('Back online — changes synced')).toBeInTheDocument();
  });

  it('dismissing clears the record so the card never reappears', async () => {
    localStorage.setItem(
      'log_reconnect_digest',
      JSON.stringify({ synced: 1, failed: 0, at: '2026-08-20T09:10:00.000Z' })
    );
    render(<ReconnectDigest />);
    const dismiss = screen.getByRole('button', { name: /dismiss/i });
    fireEvent.click(dismiss);

    await waitFor(() => {
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });
    expect(getReconnectDigest()).toBeNull();
  });

  it('exports working localStorage helpers', () => {
    expect(clearReconnectDigest()).toBeUndefined();
    expect(getReconnectDigest()).toBeNull();
  });
});