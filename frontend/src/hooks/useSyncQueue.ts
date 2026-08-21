import { useState, useEffect } from 'react';
import { getSyncQueueCount } from '@/lib/api';

export function useSyncQueue() {
  const [pendingCount, setPendingCount] = useState(0);

  useEffect(() => {
    const checkQueue = async () => {
      if (typeof window !== 'undefined') {
        const count = await getSyncQueueCount();
        setPendingCount(count);
      }
    };

    // Check immediately
    checkQueue();

    // Event-driven refresh — no constant polling:
    //   - 'log:queue-changed' fires after an enqueue or a flush (api.ts)
    //   - 'online' fires when connectivity returns (api.ts flushes on it)
    //   - visibilitychange covers cross-tab changes / returns from background,
    //     and only reads while the tab is visible
    const onQueueChanged = () => void checkQueue();
    const onOnline = () => void checkQueue();
    const onVisibility = () => {
      if (!document.hidden) void checkQueue();
    };

    window.addEventListener('log:queue-changed', onQueueChanged);
    window.addEventListener('online', onOnline);
    document.addEventListener('visibilitychange', onVisibility);

    return () => {
      window.removeEventListener('log:queue-changed', onQueueChanged);
      window.removeEventListener('online', onOnline);
      document.removeEventListener('visibilitychange', onVisibility);
    };
  }, []);

  return { pendingCount };
}