'use client';

import { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { WifiOff, RefreshCcw, CheckCircle2 } from 'lucide-react';
import { useSyncQueue } from '@/hooks/useSyncQueue';

export default function SyncIsland() {
  const { pendingCount } = useSyncQueue();
  const [isOffline, setIsOffline] = useState(false);
  const [syncComplete, setSyncComplete] = useState(false);
  const prevPending = useRef(0);

  useEffect(() => {
    const handleOnline = () => setIsOffline(false);
    const handleOffline = () => setIsOffline(true);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    // Initial check
    if (typeof navigator !== 'undefined') {
      setIsOffline(!navigator.onLine);
    }

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  // Show "Sync Complete" when the queue drains after having pending items
  useEffect(() => {
    if (prevPending.current > 0 && pendingCount === 0 && !isOffline) {
      setSyncComplete(true);
      const t = setTimeout(() => setSyncComplete(false), 3000);
      return () => clearTimeout(t);
    }
    prevPending.current = pendingCount;
  }, [pendingCount, isOffline]);

  const isSyncing = !isOffline && pendingCount > 0;
  const isVisible = isOffline || isSyncing || syncComplete;

  return (
    <div className="fixed top-6 left-1/2 -translate-x-1/2 z-50 pointer-events-none">
      <AnimatePresence>
        {isVisible && (
          <motion.div
            initial={{ y: -50, opacity: 0, scale: 0.9, borderRadius: 24 }}
            animate={{
              y: 0,
              opacity: 1,
              scale: 1,
              width: isOffline ? 190 : isSyncing ? 170 : 180,
              borderRadius: 32
            }}
            exit={{ y: -50, opacity: 0, scale: 0.9 }}
            transition={{ type: "spring", stiffness: 400, damping: 25 }}
            className={`
              flex items-center justify-center gap-2 px-4 py-3 shadow-glow-strong
              backdrop-blur-2xl border pointer-events-auto overflow-hidden
              ${isOffline ? 'bg-black/80 border-red-500/50 text-red-400' : ''}
              ${isSyncing ? 'bg-black/80 border-brand-teal/50 text-brand-neon' : ''}
              ${syncComplete ? 'bg-black/80 border-green-500/50 text-green-400' : ''}
            `}
          >
            {isOffline && (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex items-center gap-2">
                <WifiOff className="w-4 h-4" />
                <span className="text-sm font-medium">
                  {pendingCount > 0 ? `Offline — ${pendingCount} change${pendingCount > 1 ? 's' : ''} saved` : 'Offline Mode'}
                </span>
              </motion.div>
            )}

            {isSyncing && (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex items-center gap-2">
                <RefreshCcw className="w-4 h-4 animate-spin" />
                <span className="text-sm font-medium">Syncing {pendingCount} change{pendingCount > 1 ? 's' : ''}...</span>
              </motion.div>
            )}

            {syncComplete && (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4" />
                <span className="text-sm font-medium">Sync Complete</span>
              </motion.div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}