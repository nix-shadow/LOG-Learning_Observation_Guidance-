"use client";

import { useEffect, useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';
import toast from 'react-hot-toast';
import { LifeBuoy, Smartphone, Wifi, KeyRound, BookMarked, HelpCircle, ChevronLeft, CheckCircle2, Send, Clock } from 'lucide-react';
import { SupportIssue } from '@/lib/types';

// WP-2.2 RC-06: the who-to-call funnel. Guidance first (bilingual, offline-
// friendly), escalation only when the guidance did not help — escalated
// issues land in the moderator/admin inbox. Every step is honest; a
// self-served issue never clutters the inbox.
export default function SupportPage() {
  const { user } = useAuth();
  const router = useRouter();
  const t = useTranslations('support');

  const [category, setCategory] = useState<string | null>(null);
  const [helped, setHelped] = useState<boolean | null>(null);
  const [description, setDescription] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState<SupportIssue | null>(null);
  const [issues, setIssues] = useState<SupportIssue[]>([]);

  const categories = [
    { id: 'device', icon: Smartphone },
    { id: 'connectivity', icon: Wifi },
    { id: 'account', icon: KeyRound },
    { id: 'content', icon: BookMarked },
    { id: 'other', icon: HelpCircle },
  ];

  useEffect(() => {
    if (!user) {
      router.push('/login');
      return;
    }
    fetchWithCache('/support/my-issues')
      .then((d) => setIssues(d.issues || []))
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user, router]);

  const categoryName = (id: string) => t(`cat.${id}`);

  const submitEscalation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!category) return;
    setSubmitting(true);
    try {
      const res = await fetchWithCache('/support/issue', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ category, description, escalated: true }),
      });
      setDone(res);
      toast.success(t('escalatedOk'), { icon: '🆘' });
      fetchWithCache('/support/my-issues')
        .then((d) => setIssues(d.issues || []))
        .catch(() => {});
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('escalationError'));
    } finally {
      setSubmitting(false);
    }
  };

  const reset = () => {
    setCategory(null);
    setHelped(null);
    setDescription('');
    setDone(null);
  };

  return (
    <div className="max-w-3xl mx-auto w-full space-y-8">
      <div className="border-b border-white/10 pb-6">
        <h1 className="text-3xl font-bold text-white flex items-center tracking-tight">
          <LifeBuoy className="w-8 h-8 text-brand-neon mr-3" /> {t('title')}
        </h1>
        <p className="text-white/60 mt-2 text-lg">{t('subtitle')}</p>
      </div>

      {/* Wizard */}
      <div className="card-glow p-8 border border-white/10">
        {done ? (
          <div className="text-center py-6">
            <CheckCircle2 className="w-14 h-14 text-brand-neon mx-auto mb-4" />
            <h2 className="text-2xl font-bold text-white mb-2">{t('doneTitle')}</h2>
            <p className="text-white/60 mb-6">{t('doneHint')}</p>
            <button onClick={reset} className="btn-primary px-6 py-3">{t('newIssue')}</button>
          </div>
        ) : !category ? (
          <>
            <h2 className="text-xl font-bold text-white mb-6">{t('pickCategory')}</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {categories.map((c) => {
                const Icon = c.icon;
                return (
                  <button
                    key={c.id}
                    type="button"
                    onClick={() => setCategory(c.id)}
                    className="p-5 rounded-2xl border border-white/10 bg-white/5 hover:bg-brand-neon/10 hover:border-brand-neon/40 transition-all text-left"
                  >
                    <Icon className="w-7 h-7 text-brand-neon mb-3" />
                    <p className="font-bold text-white">{categoryName(c.id)}</p>
                  </button>
                );
              })}
            </div>
          </>
        ) : helped === null ? (
          <>
            <button
              type="button"
              onClick={() => setCategory(null)}
              className="text-brand-muted hover:text-white flex items-center gap-1 text-sm mb-4 transition-colors"
            >
              <ChevronLeft className="w-4 h-4" /> {t('back')}
            </button>
            <h2 className="text-xl font-bold text-white mb-4">{t('guideTitle', { category: categoryName(category) })}</h2>
            <div className="rounded-2xl bg-white/5 border border-white/10 p-6 space-y-4 mb-6">
              <p className="text-white/80 leading-relaxed whitespace-pre-line">{t(`guide.${category}`)}</p>
              <p className="text-xs text-white/40">{t('guideHint')}</p>
            </div>
            <div className="flex flex-col sm:flex-row gap-3">
              <button
                type="button"
                onClick={() => setHelped(true)}
                className="btn-primary flex-1 py-3.5 flex items-center justify-center gap-2"
              >
                <CheckCircle2 className="w-5 h-5" /> {t('helped')}
              </button>
              <button
                type="button"
                onClick={() => setHelped(false)}
                className="btn-secondary flex-1 py-3.5 bg-white/5 border-white/10 hover:bg-white/10"
              >
                {t('notHelped')}
              </button>
            </div>
          </>
        ) : (
          <>
            <button
              type="button"
              onClick={() => setHelped(null)}
              className="text-brand-muted hover:text-white flex items-center gap-1 text-sm mb-4 transition-colors"
            >
              <ChevronLeft className="w-4 h-4" /> {t('back')}
            </button>
            <h2 className="text-xl font-bold text-white mb-2">{t('describeTitle')}</h2>
            <p className="text-white/60 text-sm mb-6">{t('describeHint')}</p>
            <form onSubmit={submitEscalation} className="space-y-4">
              <textarea
                required
                minLength={10}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t('describePlaceholder')}
                rows={5}
                className="w-full px-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30 focus:ring-2 focus:ring-brand-teal focus:border-transparent outline-none transition-all"
              />
              <button
                type="submit"
                disabled={submitting}
                className="btn-primary w-full py-3.5 flex items-center justify-center gap-2 disabled:opacity-50"
              >
                <Send className="w-5 h-5" /> {submitting ? t('sending') : t('sendToTeacher')}
              </button>
            </form>
          </>
        )}
      </div>

      {/* My issues */}
      <div className="card-glow p-6 border border-white/10">
        <h2 className="text-xl font-bold text-white mb-4 flex items-center">
          <Clock className="w-5 h-5 mr-3 text-brand-amber" /> {t('myIssues')}
        </h2>
        {issues.length === 0 ? (
          <p className="text-white/50 text-sm">{t('noIssues')}</p>
        ) : (
          <div className="space-y-3">
            {issues.map((issue) => (
              <div key={issue.id} className="p-4 rounded-xl bg-white/5 border border-white/10">
                <div className="flex items-center justify-between gap-3 mb-1">
                  <p className="font-bold text-white text-sm">{categoryName(issue.category)}</p>
                  <span className={`text-[11px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full ${
                    issue.status === 'resolved'
                      ? 'bg-green-500/15 text-green-400'
                      : 'bg-brand-amber/15 text-brand-amber'
                  }`}>
                    {issue.status === 'resolved' ? t('resolved') : t('open')}
                  </span>
                </div>
                <p className="text-white/70 text-sm leading-relaxed">{issue.description}</p>
                {issue.resolution_note && (
                  <p className="text-brand-neon text-xs mt-2">{t('resolution')}: {issue.resolution_note}</p>
                )}
                <p className="text-[11px] text-white/40 mt-2">
                  {new Date(issue.created_at).toLocaleDateString()} {new Date(issue.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}