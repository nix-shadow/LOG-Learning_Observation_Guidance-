'use client';

import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';
import toast from 'react-hot-toast';
import { Inbox as InboxIcon, CheckCircle2, Smartphone, Wifi, KeyRound, BookMarked, HelpCircle } from 'lucide-react';
import { SupportIssue } from '@/lib/types';

const CATEGORY_ICONS: Record<string, typeof Smartphone> = {
  device: Smartphone,
  connectivity: Wifi,
  account: KeyRound,
  content: BookMarked,
  other: HelpCircle,
};

// WP-2.2 RC-06: the staff side of the support funnel — open escalated issues
// only (self-served issues never appear here). Resolving is audit-logged
// server-side; the list is honest and empty when nothing needs attention.
export default function SupportInbox() {
  const t = useTranslations('support');
  const [issues, setIssues] = useState<SupportIssue[] | null>(null);
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [resolving, setResolving] = useState<string | null>(null);

  const load = () => {
    fetchWithCache('/support/inbox')
      .then((d) => setIssues(d.issues || []))
      .catch(() => setIssues([]));
  };

  useEffect(load, []);

  const resolve = async (issue: SupportIssue) => {
    const note = notes[issue.id]?.trim();
    if (!note) {
      toast.error(t('resolutionRequired'));
      return;
    }
    setResolving(issue.id);
    try {
      await fetchWithCache(`/support/issue/${issue.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ resolution_note: note }),
      });
      toast.success(t('resolvedOk'), { icon: '✅' });
      setIssues((prev) => (prev || []).filter((i) => i.id !== issue.id));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('resolveError'));
    } finally {
      setResolving(null);
    }
  };

  return (
    <div className="card-glow p-6 border border-white/10">
      <h2 className="text-2xl font-bold text-white mb-6 tracking-tight flex items-center">
        <InboxIcon className="w-6 h-6 mr-3 text-brand-amber" /> {t('inboxTitle')}
      </h2>

      {issues === null && <p className="text-white/50">{t('loading')}</p>}

      {issues !== null && issues.length === 0 && (
        <div className="text-center py-8">
          <CheckCircle2 className="w-10 h-10 text-brand-neon mx-auto mb-3" />
          <p className="text-white/60">{t('inboxEmpty')}</p>
        </div>
      )}

      {issues && issues.length > 0 && (
        <div className="space-y-4">
          {issues.map((issue) => {
            const Icon = CATEGORY_ICONS[issue.category] || HelpCircle;
            return (
              <div key={issue.id} className="p-5 rounded-2xl bg-white/5 border border-white/10">
                <div className="flex items-start gap-3">
                  <span className="p-2.5 rounded-xl bg-brand-amber/15 text-brand-amber flex-shrink-0">
                    <Icon className="w-5 h-5" />
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center justify-between gap-2 mb-1">
                      <p className="font-bold text-white">{t(`cat.${issue.category}`)}</p>
                      <span className="text-[11px] text-white/40">
                        {new Date(issue.created_at).toLocaleDateString()} {new Date(issue.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </span>
                    </div>
                    <p className="text-white/70 text-sm leading-relaxed">{issue.description}</p>
                    <div className="flex gap-3 mt-3">
                      <input
                        value={notes[issue.id] || ''}
                        onChange={(e) => setNotes((prev) => ({ ...prev, [issue.id]: e.target.value }))}
                        placeholder={t('resolutionPlaceholder')}
                        maxLength={500}
                        className="flex-1 px-3 py-2 bg-black/40 border border-white/10 rounded-lg text-white focus:ring-2 focus:ring-brand-amber/50 outline-none placeholder-white/30 text-sm"
                      />
                      <button
                        onClick={() => resolve(issue)}
                        disabled={resolving === issue.id}
                        className="btn-primary px-4 py-2 text-sm flex items-center gap-2 disabled:opacity-50"
                      >
                        <CheckCircle2 className="w-4 h-4" /> {resolving === issue.id ? t('resolving') : t('resolve')}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}