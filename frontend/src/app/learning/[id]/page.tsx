"use client";

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import toast from 'react-hot-toast';
import { fetchWithCache } from '@/lib/api';
import MicroModuleViewer, { MicroModuleData, AttemptReport } from '@/components/MicroModuleViewer';
import SkeletonLoader from '@/components/SkeletonLoader';

export default function LessonModule() {
  const router = useRouter();
  const params = useParams();
  const activityId = params?.id as string || 'act-2';
  const [modules, setModules] = useState<MicroModuleData[]>([]);
  const [activityTitle, setActivityTitle] = useState('');
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
    try {
      await fetchWithCache(`/activities/${activityId}/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(stats),
      });
      toast.success('Lesson marked as completed! Progress recorded.', { icon: '🎉' });
    } catch (e) {
      console.warn('Offline completion queued or sync in progress', e);
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
      <MicroModuleViewer modules={modules} onComplete={handleComplete} />
    </div>
  );
}
