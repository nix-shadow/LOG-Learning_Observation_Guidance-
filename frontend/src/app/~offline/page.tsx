import { WifiOff } from 'lucide-react';

// WP-0.3 bundle research round: previously an offline navigation attempt fell
// to the browser's native "no internet" error page. With navigateFallback the
// service worker serves this page instead — it is the ONLY page guaranteed to
// work offline (it is statically generated, no API calls, no IndexedDB).
export default function OfflinePage() {
  return (
    <main className="min-h-screen flex flex-col items-center justify-center px-6 text-center bg-brand-dark">
      <div className="card-glow bg-black/40 backdrop-blur-2xl border-white/10 rounded-2xl p-10 max-w-md">
        <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-white/5 border border-white/10 flex items-center justify-center">
          <WifiOff className="w-8 h-8 text-brand-muted" />
        </div>
        <h1 className="text-2xl font-bold text-white mb-2 tracking-tight">You&apos;re offline</h1>
        <p className="text-brand-muted text-sm leading-relaxed mb-6">
          This page was not saved for offline use. Reconnect to the internet to load it — your
          saved progress is safe and will sync automatically.
        </p>
        <a
          href="/dashboard"
          className="inline-block px-6 py-3 rounded-xl bg-brand-teal/10 border border-brand-teal/30 text-brand-teal font-semibold text-sm hover:bg-brand-teal/20 transition-colors"
        >
          Try the Dashboard
        </a>
      </div>
    </main>
  );
}