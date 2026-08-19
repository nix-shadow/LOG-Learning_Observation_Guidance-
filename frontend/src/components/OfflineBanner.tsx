"use client";

import { useState, useEffect } from 'react';
import { m as motion, AnimatePresence } from 'framer-motion';
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
            transition={{ type: "spring", stiffness: 400, damping: 25 }}
            className="bg-brand-amber/10 backdrop-blur-md border-b border-brand-amber/30 text-brand-blue font-medium text-sm py-3 px-4 flex items-center justify-center shadow-sm relative z-50"
          >
            <WifiOff className="w-4 h-4 mr-2 text-brand-amber" />
            <span className="opacity-90">You are currently offline. Changes will be saved locally and synced when you reconnect.</span>
          </motion.div>
          
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            transition={{ type: "spring", stiffness: 400, damping: 20 }}
            className="fixed bottom-6 left-6 z-40 flex items-center gap-2 font-bold text-xs bg-white/80 backdrop-blur-md border border-brand-gray/50 text-brand-text px-3 py-2 rounded-full shadow-bento"
          >
            <div className="relative flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-brand-amber opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-brand-amber"></span>
            </div>
            <span>OFFLINE MODE</span>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
