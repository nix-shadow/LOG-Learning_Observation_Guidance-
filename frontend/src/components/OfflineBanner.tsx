"use client";

import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { WifiOff } from 'lucide-react';

export default function OfflineBanner() {
  const [isOffline, setIsOffline] = useState(false);

  useEffect(() => {
    // Check initial state
    if (typeof navigator !== 'undefined') {
      setIsOffline(!navigator.onLine);
    }

    const handleOnline = () => setIsOffline(false);
    const handleOffline = () => setIsOffline(true);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return (
    <AnimatePresence>
      {isOffline && (
        <>
          <motion.div
            initial={{ y: -50, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -50, opacity: 0 }}
            transition={{ type: "spring", stiffness: 300, damping: 25 }}
            className="bg-brand-amber text-brand-blue font-medium text-sm py-2 px-4 flex items-center justify-center shadow-md relative z-50"
          >
            <WifiOff className="w-4 h-4 mr-2" />
            You are currently offline. Changes will be saved locally and synced when you reconnect.
          </motion.div>
          
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed bottom-4 left-4 z-40 opacity-50 pointer-events-none flex items-center gap-1 font-bold text-xs bg-gray-900 text-white px-2 py-1 rounded shadow"
          >
            <WifiOff className="w-3 h-3 text-brand-amber" />
            OFFLINE MODE - CACHING LOCALLY
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
