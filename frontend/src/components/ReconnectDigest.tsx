'use client';

import { useState, useEffect } from 'react';
import { CheckCircle2, AlertTriangle, X } from 'lucide-react';
import {
  getReconnectDigest,
  clearReconnectDigest,
  ReconnectDigest as ReconnectDigestData,
} from '@/lib/api';
import { useTranslations } from 'next-intl';

// WP-2.4: the reconnect digest card. After the offline queue flushes (online
// event, boot, or the SyncIsland "Sync now" button) api.ts writes an honest
// summary to localStorage and dispatches log:digest-ready. This card reads
// it — including a digest produced before this component mounted — and shows
// exactly what came back online and what still waits. Dismissing clears the
// record; there is no fake "all synced" state.
export default function ReconnectDigest() {
  const t = useTranslations('digest');
  const [digest, setDigest] = useState<ReconnectDigestData | null>(null);

  useEffect(() => {
    const read = () => setDigest(getReconnectDigest());
    read();
    window.addEventListener('log:digest-ready', read);
    return () => window.removeEventListener('log:digest-ready', read);
  }, []);

  if (!digest) return null;

  const time = (() => {
    const d = new Date(digest.at);
    if (Number.isNaN(d.getTime())) return '';
    return `${d.toLocaleDateString()} ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  })();

  const dismiss = () => {
    clearReconnectDigest();
    setDigest(null);
  };

  return (
    <div
      className="card-glow p-5 border border-brand-teal/30 bg-gradient-to-br from-brand-teal/10 to-transparent"
      role="status"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          {digest.failed > 0 ? (
            <AlertTriangle className="w-6 h-6 text-brand-amber flex-shrink-0 mt-0.5" />
          ) : (
            <CheckCircle2 className="w-6 h-6 text-brand-neon flex-shrink-0 mt-0.5" />
          )}
          <div>
            <p className="font-bold text-white text-lg">
              {digest.failed > 0 ? t('partialTitle') : t('title')}
            </p>
            <p className="text-sm text-white/70 mt-1">
              {t('synced')}: <span className="text-brand-neon font-semibold">{digest.synced}</span>
              {' · '}
              {t('failed')}: <span className="text-brand-amber font-semibold">{digest.failed}</span>
            </p>
            {time && <p className="text-[11px] text-white/40 mt-1">{t('at', { time })}</p>}
          </div>
        </div>
        <button
          type="button"
          onClick={dismiss}
          aria-label={t('dismiss')}
          className="text-white/40 hover:text-white transition-colors p-1 rounded-lg hover:bg-white/10"
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}