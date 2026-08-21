"use client";

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, Subtitles } from 'lucide-react';
import toast from 'react-hot-toast';
import { fetchWithCache } from '@/lib/api';
import { getReviewState, saveReviewState } from '@/lib/reviewStore';
import { scheduleReview } from '@/lib/spacedRepetition';
import MicroModuleViewer, { MicroModuleData, AttemptReport } from '@/components/MicroModuleViewer';
import SkeletonLoader from '@/components/SkeletonLoader';

export default function LessonModule() {
  const router = useRouter();
  const params = useParams();
  const activityId = params?.id as string || 'act-2';
  const [modules, setModules] = useState<MicroModuleData[]>([]);
  const [activityTitle, setActivityTitle] = useState('');
  const [activityCaption, setActivityCaption] = useState('');
  const [activityAttribution, setActivityAttribution] = useState('');
  const [activityLicense, setActivityLicense] = useState('');
  const [activityLicenseUrl, setActivityLicenseUrl] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);

  useEffect(() => {
    // Fetch the real micro-modules for this activity (cached for offline replay).
    fetchWithCache(`/activities/${activityId}/modules`)
      .then((res) => {
        setModules((res.modules || []).map((m: MicroModuleData) => ({
          id: m.id,
          title: m.title,
          content_text: m.content_text,
          media_url: m.media_url,
          question: m.question,
          options: m.options,
          correct_index: m.correct_index,
          explanation: m.explanation,
        })));
        setActivityTitle(res.activity?.title || '');
        // WP-3.1/3.4: OER metadata + NSL caption track surface here so the
        // lesson page credits real sources and flags real captions.
        setActivityCaption(res.activity?.caption_text || '');
        setActivityAttribution(res.activity?.attribution || '');
        setActivityLicense(res.activity?.license || '');
        setActivityLicenseUrl(res.activity?.license_url || '');
        setLoadError(false);
      })
      .catch(() => {
        // Honest state: no fabricated lesson content (AGENTS.md — no invented
        // numbers or placeholder content when data is unavailable).
        setLoadError(true);
      })
      .finally(() => setLoading(false));
  }, [activityId]);

  if (loading) return (
    <div className="max-w-3xl mx-auto w-full space-y-6">
      <div className="flex items-center justify-between text-sm font-bold text-white/50 tracking-wider uppercase">
        <span><ArrowLeft className="inline w-4 h-4 mr-2" /> Loading module...</span>
      </div>
      <SkeletonLoader type="card" count={2} />
    </div>
  );

  // Error vs empty are distinct states: a fetch failure offers retry; a
  // genuinely content-free activity just says so (nothing to fabricate).
  if (loadError) return (
    <div className="max-w-3xl mx-auto w-full text-center py-24 space-y-6">
      <h1 className="text-2xl font-bold text-white">Could not load this lesson</h1>
      <p className="text-white/60">Check your connection and try again. If you were offline, reconnect and reload.</p>
      <button onClick={() => { setLoading(true); setLoadError(false); window.location.reload(); }} className="btn-primary px-8 py-3 font-bold">
        Try Again
      </button>
    </div>
  );

  if (modules.length === 0) return (
    <div className="max-w-3xl mx-auto w-full text-center py-24 space-y-6">
      <h1 className="text-2xl font-bold text-white">Lesson content is not ready yet</h1>
      <p className="text-white/60">This activity has no modules yet. Please check back later.</p>
      <button onClick={() => router.push('/learning')} className="btn-primary px-8 py-3 font-bold">
        <ArrowLeft className="inline w-4 h-4 mr-2" /> Back to Journey
      </button>
    </div>
  );

  const handleComplete = async (stats: AttemptReport) => {
    // WP-1.4: every real attempt advances the SM-2 review schedule. Accuracy
    // is derived from actual answers (correct_count / total_count) — never
    // invented. The completion itself flows through the normal sync queue, so
    // reviews are always backed by honest backend completions.
    const accuracy =
      stats.total_count > 0 ? stats.correct_count / stats.total_count : 1;
    const completedAt = new Date(stats.completed_at_unix_ms ?? Date.now()).toISOString();
    // C2 (Phase 4): the review schedule advances ONLY when the completion is
    // on a real server path — an online 2xx or a confirmed offline queue
    // entry (202). A 4xx failure (e.g. consent_required) must never leave
    // client-side-only review state that the backend does not know about.
    let recorded = false;
    try {
      const res = await fetchWithCache(`/activities/${activityId}/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(stats),
      });
      recorded = res?.ok === true || (res as { queued?: boolean })?.queued === true;
    } catch (e) {
      console.warn('Completion not recorded', e);
    }
    if (recorded) {
      const prev = await getReviewState(activityId);
      await saveReviewState(scheduleReview(prev ?? null, accuracy, completedAt));
      toast.success('Lesson marked as completed! Progress recorded.', { icon: '🎉' });
    } else {
      toast.error('Progress could not be recorded. Check your connection and try again.');
    }
    router.push('/learning');
  };

  return (
    <div className="max-w-3xl mx-auto w-full">
      <button onClick={() => router.back()} className="text-white/50 hover:text-brand-neon flex items-center mb-6 transition-colors font-bold tracking-wide text-sm uppercase">
        <ArrowLeft className="w-4 h-4 mr-2" /> Back to Journey
      </button>
      {activityTitle && (
        <h1 className="text-3xl font-bold text-white mb-6 tracking-tight">{activityTitle}</h1>
      )}

      {/* WP-3.4 RC-12: NSL caption track — honest: shown only when the
          activity actually carries a caption. */}
      {activityCaption && (
        <div className="mb-6 card-glow bg-black/30 backdrop-blur-2xl border border-brand-amber/30 rounded-2xl p-4">
          <p className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-brand-amber mb-2">
            <Subtitles className="w-4 h-4" /> NSL captions available
          </p>
          <p className="text-sm text-white/70 leading-relaxed">{activityCaption}</p>
        </div>
      )}

      {/* WP-3.1 RC-07: OER credit line on the lesson page. */}
      {(activityLicense || activityAttribution) && (
        <p className="text-xs text-brand-muted mb-6">
          {activityAttribution && <>{activityAttribution}. </>}
          {activityLicense && (
            <>
              Licensed under{' '}
              <a
                href={activityLicenseUrl || '#'}
                target={activityLicenseUrl ? '_blank' : undefined}
                rel="noopener noreferrer"
                className="text-brand-teal/80 hover:text-brand-teal underline underline-offset-2"
              >
                {activityLicense}
              </a>
            </>
          )}
        </p>
      )}

      <MicroModuleViewer modules={modules} onComplete={handleComplete} />
    </div>
  );
}
