"use client";

// WP-1.5 per-student progress. Feeds directly from GET /moderator/students/:id
// which is built on the WP-1.1 status engine — the exact canonical statuses
// and supportive vocabulary the learner sees. Hard 404/500 surfaces are shown
// honestly; there is no fabricated progress anywhere.

import { useEffect, useState } from 'react';
import { X, Loader2, AlertCircle } from 'lucide-react';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';

export interface StudentProgressPayload {
  student: { id: string; name: string; email: string };
  progress: {
    total_topics: number;
    completed: number;
    current_streak: number;
    overall_score: number;
  };
  activities: { id: string; title: string; topic: string; status: string }[];
  guidance: { id: string; text: string; action?: string }[];
}

interface Props {
  token: string;
  studentId: string;
  studentName: string;
  onClose: () => void;
}

const STATUS_CLS: Record<string, string> = {
  completed: 'bg-brand-neon/15 text-brand-neon',
  'needs-practice': 'bg-amber-500/15 text-amber-300',
  active: 'bg-brand-amber/15 text-brand-amber',
  'not-started': 'bg-white/10 text-white/50',
};

export default function StudentProgressModal({ token, studentId, studentName, onClose }: Props) {
  const t = useTranslations('studentProgress');
  const [data, setData] = useState<StudentProgressPayload | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    setData(null);
    setError(false);
    fetchWithCache(`/moderator/students/${studentId}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(setData)
      .catch(() => setError(true));
  }, [studentId, token]);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/70 backdrop-blur-sm p-4 overflow-y-auto" onClick={onClose}>
      <div
        className="card-glow bg-[#0b0f17] border border-white/10 rounded-3xl p-8 w-full max-w-2xl my-8 relative"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-white/50 hover:text-white transition-colors"
          aria-label="Close student progress"
        >
          <X className="w-5 h-5" />
        </button>

        <h2 className="text-2xl font-bold text-white tracking-tight mb-1">{studentName}</h2>
        <p className="text-sm text-white/50 mb-6">{t('subtitle')}</p>

        {error ? (
          <div className="py-10 text-center">
            <AlertCircle className="w-8 h-8 text-brand-amber mx-auto mb-3" />
            <p className="text-white/70">{t('error')}</p>
            <button onClick={onClose} className="mt-4 text-brand-neon font-semibold hover:text-white">
              {t('close')}
            </button>
          </div>
        ) : !data ? (
          <div className="py-14 flex items-center justify-center">
            <Loader2 className="w-8 h-8 text-brand-neon animate-spin" />
          </div>
        ) : (
          <div className="space-y-6">
            <div className="grid grid-cols-3 gap-3">
              <div className="bg-white/5 rounded-2xl p-4 text-center">
                <p className="text-3xl font-bold text-white">{data.progress.completed}<span className="text-white/40 text-lg">/{data.progress.total_topics}</span></p>
                <p className="text-xs text-white/50 mt-1 uppercase tracking-wider">{t('topicsCompleted')}</p>
              </div>
              <div className="bg-white/5 rounded-2xl p-4 text-center">
                <p className="text-3xl font-bold text-brand-amber">{data.progress.current_streak}</p>
                <p className="text-xs text-white/50 mt-1 uppercase tracking-wider">{t('dayStreak')}</p>
              </div>
              <div className="bg-white/5 rounded-2xl p-4 text-center">
                <p className="text-3xl font-bold text-brand-neon">{Math.round(data.progress.overall_score)}</p>
                <p className="text-xs text-white/50 mt-1 uppercase tracking-wider">{t('overallScore')}</p>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-bold text-white/60 uppercase tracking-wider mb-3">{t('journeyTitle')}</h3>
              <ul className="space-y-2">
                {(data.activities || []).length === 0 ? (
                  <li className="text-white/40 text-sm">{t('noActivity')}</li>
                ) : data.activities.map((a) => {
                  const label = t(a.status);
                  const cls = STATUS_CLS[a.status] ?? STATUS_CLS['not-started'];
                  return (
                    <li key={a.id} className="flex items-center justify-between gap-3 px-4 py-3 bg-white/5 rounded-xl">
                      <div className="min-w-0">
                        <p className="text-white/90 font-medium text-sm truncate">{a.title}</p>
                        <p className="text-white/40 text-xs">{a.topic}</p>
                      </div>
                      <span className={`shrink-0 text-[10px] font-bold uppercase tracking-wider px-2.5 py-1 rounded-full ${cls}`}>
                        {label}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </div>

            {(data.guidance || []).length > 0 && (
              <div>
                <h3 className="text-sm font-bold text-white/60 uppercase tracking-wider mb-3">{t('guidanceTitle')}</h3>
                <ul className="space-y-2">
                  {data.guidance.map((g) => (
                    <li key={g.id} className="border-l-4 border-l-brand-amber bg-white/5 px-4 py-3 rounded-r-xl">
                      <p className="text-sm text-white/80">{g.text}</p>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}