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

    // Poll every 5 seconds while mounted
    const interval = setInterval(checkQueue, 5000);

    // Also check when returning online, as a sync might have just happened
    window.addEventListener('online', checkQueue);

    return () => {
      clearInterval(interval);
      window.removeEventListener('online', checkQueue);
    };
  }, []);

  return { pendingCount };
}
