"use client";

import { useEffect, useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';
import toast from 'react-hot-toast';
import { HeartHandshake, ChevronDown, ChevronUp, CheckCircle2, Flame, Medal, Mail } from 'lucide-react';
import { ParentChild, ChildDigest } from '@/lib/types';

// WP-2.1 RC-04: the school-verified parent portal. Read-only digest of linked
// learners (id + name only, no contacts), with an honest opt-in for the
// weekly digest. Every number comes from the backend; there are no invented
// placeholders.
export default function ParentPortal() {
  const { user, isParent } = useAuth();
  const router = useRouter();
  const t = useTranslations('parent');
  const td = useTranslations('parentDigest');

  const [children, setChildren] = useState<ParentChild[] | null>(null);
  const [digests, setDigests] = useState<Record<string, ChildDigest>>({});
  const [openChild, setOpenChild] = useState<string | null>(null);
  const [loadingDigest, setLoadingDigest] = useState<string | null>(null);

  useEffect(() => {
    if (!user || !isParent) {
      router.push('/dashboard');
      return;
    }
    fetchWithCache('/parents/children')
      .then((d) => setChildren(d.children || []))
      .catch(() => setChildren([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user, isParent, router]);

  if (!isParent) return null;

  const loadDigest = (childId: string) => {
    if (openChild === childId) {
      setOpenChild(null);
      return;
    }
    setOpenChild(childId);
    if (digests[childId]) return;
    setLoadingDigest(childId);
    fetchWithCache(`/parents/children/${childId}/digest`)
      .then((d) => setDigests((prev) => ({ ...prev, [childId]: d })))
      .catch(() => toast.error(t('digestError')))
      .finally(() => setLoadingDigest(null));
  };

  const toggleOptIn = async (child: ParentChild) => {
    try {
      const res = await fetchWithCache(`/parents/children/${child.id}/opt-in`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !child.digest_opt_in }),
      });
      setChildren((prev) =>
        (prev || []).map((c) =>
          c.id === child.id ? { ...c, digest_opt_in: res.digest_opt_in ?? !child.digest_opt_in } : c
        )
      );
      toast.success(res.digest_opt_in ? t('optInOn') : t('optInOff'), { icon: '💌' });
    } catch {
      toast.error(t('optInError'));
    }
  };

  const statusLabels: Record<string, string> = {
    'not-started': td('notStarted'),
    'active': td('active'),
    'needs-practice': td('needsPractice'),
    'completed': td('completed'),
  };

  const empty = children !== null && children.length === 0;

  return (
    <div className="max-w-4xl mx-auto w-full space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8 border-b border-white/10 pb-6">
        <div>
          <h1 className="text-3xl font-bold text-white flex items-center tracking-tight">
            <HeartHandshake className="w-8 h-8 text-brand-neon mr-3" /> {t('title')}
          </h1>
          <p className="text-white/60 mt-2 text-lg">{t('subtitle')}</p>
        </div>
        {user && (
          <p className="text-sm text-white/40 flex items-center gap-2">
            <Mail className="w-4 h-4" /> {user.email}
          </p>
        )}
      </div>

      {children === null && (
        <div className="card-glow p-8 text-center text-white/50">{t('loading')}</div>
      )}

      {empty && (
        <div className="card-glow border border-brand-neon/30 bg-gradient-to-br from-brand-neon/10 to-transparent rounded-3xl p-8">
          <h2 className="text-2xl font-bold text-white tracking-tight mb-3">{t('emptyTitle')}</h2>
          <p className="text-white/60 leading-relaxed">{t('emptyHint')}</p>
        </div>
      )}

      {children && children.length > 0 && (
        <div className="space-y-4">
          {children.map((child) => {
            const digest = digests[child.id];
            const isOpen = openChild === child.id;
            return (
              <div key={child.id} className="card-glow p-6 border border-white/10">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div className="flex items-center gap-4">
                    <span className="w-12 h-12 rounded-2xl bg-brand-neon/15 border border-brand-neon/30 text-brand-neon flex items-center justify-center font-bold text-lg">
                      {child.name.charAt(0).toUpperCase()}
                    </span>
                    <div>
                      <p className="font-bold text-white text-lg">{child.name}</p>
                      <p className="text-xs text-white/40">{t('linkedChild')}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <label className="flex items-center gap-2 text-sm text-white/60 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        checked={child.digest_opt_in}
                        onChange={() => toggleOptIn(child)}
                        className="w-4 h-4 accent-brand-teal"
                      />
                      {t('digestOptIn')}
                    </label>
                    <button
                      type="button"
                      onClick={() => loadDigest(child.id)}
                      className="btn-secondary flex items-center gap-2 text-sm px-4 py-2 rounded-full bg-white/5 border-white/10 hover:bg-white/10"
                      aria-expanded={isOpen}
                    >
                      {isOpen ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                      {isOpen ? t('hideDigest') : t('viewDigest')}
                    </button>
                  </div>
                </div>

                {isOpen && (
                  <div className="mt-6 pt-6 border-t border-white/10">
                    {loadingDigest === child.id && <p className="text-white/50 text-sm">{td('loading')}</p>}
                    {digest && (
                      <div className="space-y-6">
                        {/* Honest progress numbers */}
                        <div className="grid grid-cols-3 gap-3">
                          <div className="p-4 rounded-2xl bg-white/5 border border-white/10">
                            <p className="text-[11px] uppercase tracking-widest text-white/40 font-bold">{td('topicsCompleted')}</p>
                            <p className="text-2xl font-bold text-brand-neon mt-1">
                              {digest.progress.completed}<span className="text-white/40 text-sm font-normal"> / {digest.progress.total_topics}</span>
                            </p>
                          </div>
                          <div className="p-4 rounded-2xl bg-white/5 border border-white/10">
                            <p className="text-[11px] uppercase tracking-widest text-white/40 font-bold">{td('dayStreak')}</p>
                            <p className="text-2xl font-bold text-brand-amber mt-1 flex items-center gap-2">
                              <Flame className="w-5 h-5" /> {digest.progress.current_streak}
                            </p>
                          </div>
                          <div className="p-4 rounded-2xl bg-white/5 border border-white/10">
                            <p className="text-[11px] uppercase tracking-widest text-white/40 font-bold">{td('overallScore')}</p>
                            <p className="text-2xl font-bold text-white mt-1 flex items-center gap-2">
                              <Medal className="w-5 h-5 text-brand-neon" /> {Math.round(digest.progress.overall_score)}%
                            </p>
                          </div>
                        </div>

                        {/* Activity digest with canonical supportive statuses */}
                        <div>
                          <p className="font-bold text-white mb-3">{td('journeyTitle')}</p>
                          {digest.activities.length === 0 ? (
                            <p className="text-white/50 text-sm">{td('noActivity')}</p>
                          ) : (
                            <div className="space-y-2">
                              {digest.activities.map((act) => (
                                <div key={act.id} className="flex items-center justify-between gap-3 p-3 rounded-xl bg-white/5 border border-white/10">
                                  <div className="min-w-0">
                                    <p className="text-white font-medium text-sm truncate">{act.title}</p>
                                    {act.topic && <p className="text-white/40 text-xs">{act.topic}</p>}
                                  </div>
                                  <span className={`text-xs font-semibold whitespace-nowrap ${
                                    act.status === 'completed' ? 'text-brand-neon' :
                                    act.status === 'needs-practice' ? 'text-brand-amber' : 'text-white/50'
                                  }`}>
                                    {statusLabels[act.status] ?? act.status}
                                  </span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        {/* Guidance — supportive phrasing, the same the learner sees */}
                        {digest.guidance.length > 0 && (
                          <div>
                            <p className="font-bold text-white mb-3">{td('guidanceTitle')}</p>
                            <div className="space-y-2">
                              {digest.guidance.map((g) => (
                                <div key={g.id} className="p-3 rounded-xl bg-brand-amber/10 border border-brand-amber/20">
                                  <p className="text-white/90 text-sm leading-relaxed">{g.text}</p>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {digest.as_of && (() => {
                          const d = new Date(digest.as_of);
                          if (Number.isNaN(d.getTime())) return null;
                          return (
                            <p className="text-[11px] text-white/40">
                              {td('updated')} {d.toLocaleDateString()} {td('at')} {d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                            </p>
                          );
                        })()}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Supportive closing line — RC-04 tone */}
      {children !== null && children.length > 0 && (
        <p className="text-center text-white/40 text-sm flex items-center justify-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-brand-neon" /> {t('readOnlyNote')}
        </p>
      )}
    </div>
  );
}