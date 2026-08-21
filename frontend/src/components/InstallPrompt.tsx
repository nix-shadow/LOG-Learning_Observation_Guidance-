"use client";

import { useEffect, useState } from 'react';
import { Download, X, Share } from 'lucide-react';
import { m as motion, AnimatePresence } from 'framer-motion';

// WP-0.3 a11y research round: the dismiss button had no accessible name, and
// iOS Safari never fires beforeinstallprompt — without an iOS fallback the
// prompt simply never appeared for the biggest school-device fleet. Now the
// prompt shows "Share → Add to Home Screen" instructions on iOS instead.
export default function InstallPrompt() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [deferredPrompt, setDeferredPrompt] = useState<any>(null);
  const [showPrompt, setShowPrompt] = useState(false);
  const [isIOS, setIsIOS] = useState(false);

  useEffect(() => {
    // Only show if we haven't dismissed it recently (wait 7 days)
    const dismissed = localStorage.getItem('log_pwa_dismissed');
    if (dismissed && Date.now() - parseInt(dismissed) < 7 * 24 * 60 * 60 * 1000) {
      return;
    }

    // iOS Safari (and iPadOS) do not fire beforeinstallprompt — detect and
    // show the manual Share → Add to Home Screen flow instead.
    const ua = navigator.userAgent || '';
    const iOS = /iphone|ipad|ipod/i.test(ua) && !(window as unknown as { MSStream?: unknown }).MSStream;
    setIsIOS(iOS);
    if (iOS) {
      setShowPrompt(true);
      return;
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handleBeforeInstallPrompt = (e: any) => {
      // Prevent the mini-infobar from appearing on mobile
      e.preventDefault();
      // Stash the event so it can be triggered later
      setDeferredPrompt(e);
      // Update UI to notify the user they can install the PWA
      setShowPrompt(true);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    };
  }, []);

  const handleInstallClick = async () => {
    if (!deferredPrompt) return;

    // Show the install prompt
    deferredPrompt.prompt();

    // Wait for the user to respond to the prompt
    await deferredPrompt.userChoice;

    // We've used the prompt, and can't use it again, throw it away
    setDeferredPrompt(null);
    setShowPrompt(false);
  };

  const handleDismiss = () => {
    setShowPrompt(false);
    localStorage.setItem('log_pwa_dismissed', Date.now().toString());
  };

  return (
    <AnimatePresence>
      {showPrompt && (
        <motion.div
          initial={{ y: 100, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: 100, opacity: 0 }}
          className="fixed bottom-20 md:bottom-6 right-4 left-4 md:left-auto md:w-96 bg-brand-blue text-white p-4 rounded-xl shadow-2xl z-[100] flex items-center justify-between border border-white/20"
        >
          <div className="flex items-center gap-3">
            <div className="bg-white/20 p-2 rounded-lg">
              {isIOS ? <Share className="w-6 h-6 text-brand-teal" /> : <Download className="w-6 h-6 text-brand-teal" />}
            </div>
            <div>
              <p className="font-bold text-sm">Install LOG App</p>
              {isIOS ? (
                <p className="text-xs text-brand-gray/80">Tap Share <Share className="inline w-3 h-3" /> then &quot;Add to Home Screen&quot; for offline access</p>
              ) : (
                <p className="text-xs text-brand-gray/80">Add to home screen for offline access</p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 ml-4">
            {!isIOS && (
              <button
                onClick={handleInstallClick}
                className="px-4 py-2 bg-brand-teal text-white font-bold text-sm rounded-lg hover:bg-brand-teal/90 transition-colors"
              >
                Install
              </button>
            )}
            <button
              onClick={handleDismiss}
              aria-label="Dismiss install prompt"
              className="p-1 text-brand-gray hover:text-white transition-colors rounded-full hover:bg-white/10"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}