"use client";

// WP-1.4: review queue card. Reads the local SM-2 schedule and shows what is
// genuinely due today/overdue. Empty schedule → honest "nothing due" state,
// never invented items (AGENTS.md — no fabricated data).

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { RefreshCcw, CalendarCheck } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { getAllReviewStates } from '@/lib/reviewStore';
import { isDue, ReviewState } from '@/lib/spacedRepetition';
import { fetchWithCache } from '@/lib/api';

interface DueItem {
  activityId: string;
  title: string;
  overdue: boolean;
  dueDate: string;
}

export default function ReviewQueueCard() {
  const t = useTranslations('review');
  const [dueItems, setDueItems] = useState<DueItem[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const states = await getAllReviewStates();
      const now = new Date();
      const due = states.filter((s) => isDue(s, now));
      if (due.length === 0) {
        if (!cancelled) setDueItems([]);
        return;
      }
      try {
        const journey = await fetchWithCache('/learning-journey');
        const titles = new Map<string, string>(
          (journey.activities || []).map((a: { id: string; title: string }) => [a.id, a.title])
        );
        const items: DueItem[] = due
          .map((s: ReviewState) => ({
            activityId: s.activityId,
            title: titles.get(s.activityId) ?? s.activityId,
            overdue: s.dueDate < now.toISOString().slice(0, 10),
            dueDate: s.dueDate,
          }))
          .sort((a, b) => (a.overdue === b.overdue ? 0 : a.overdue ? -1 : 1));
        if (!cancelled) setDueItems(items);
      } catch {
        if (!cancelled) setDueItems(due.map((s) => ({
          activityId: s.activityId,
          title: s.activityId,
          overdue: s.dueDate < now.toISOString().slice(0, 10),
          dueDate: s.dueDate,
        })));
      }
    })();
    return () => { cancelled = true; };
  }, []);

  return (
    <div className="card-glow border border-white/5 bg-black/20 rounded-2xl p-6 h-full">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-white flex items-center gap-2">
          <RefreshCcw className="w-4 h-4 text-brand-neon" />
          {t('title')}
        </h3>
        <span className="text-[10px] font-bold uppercase tracking-[0.2em] text-white/40">
          {t('badge')}
        </span>
      </div>
      {dueItems === null ? (
        <p className="text-sm text-white/40">{t('loading')}</p>
      ) : dueItems.length === 0 ? (
        <div className="flex items-start gap-3 py-2">
          <span className="p-2 rounded-full bg-brand-dark text-brand-neon shrink-0">
            <CalendarCheck className="w-5 h-5" />
          </span>
          <div>
            <p className="text-sm text-white/80">{t('empty')}</p>
            <p className="text-xs text-white/40 mt-1">{t('emptyHint')}</p>
          </div>
        </div>
      ) : (
        <ul className="space-y-2">
          {dueItems.map((item) => (
            <li key={item.activityId}>
              <Link
                href={`/learning/${item.activityId}`}
                className="flex items-center justify-between gap-3 px-3 py-2 rounded-xl bg-white/5 hover:bg-white/10 transition-colors"
              >
                <span className="text-sm text-white/80 line-clamp-1">{item.title}</span>
                <span
                  className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full shrink-0 ${
                    item.overdue ? 'bg-amber-500/15 text-amber-300' : 'bg-brand-dark text-brand-neon'
                  }`}
                >
                  {item.overdue ? t('overdue') : t('dueToday')}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}