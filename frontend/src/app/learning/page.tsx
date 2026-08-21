"use client";
import { LearningJourneyData, Activity } from "@/lib/types";

import { useEffect, useState, useRef } from 'react';
import Link from 'next/link';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';
import { CheckCircle2, Circle, PlayCircle, RefreshCcw, ChevronRight, Activity as ActivityIcon } from 'lucide-react';
import SkeletonLoader from '@/components/SkeletonLoader';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';

export default function Learning() {
  const t = useTranslations('status');
  const [data, setData] = useState<LearningJourneyData | null>(null);
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchWithCache('/learning-journey')
      .then(setData)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  useGSAP(() => {
    if (prefersReducedMotion()) return;
    if (!loading && data) {
      gsap.fromTo(
        gsap.utils.toArray('.gsap-stagger'),
        { y: 50, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.8, stagger: 0.1, ease: 'power3.out' }
      );
    }
  }, { scope: containerRef, dependencies: [loading, data] });

  if (loading) return (
    <div className="max-w-6xl mx-auto w-full space-y-6">
      <SkeletonLoader type="text" count={2} />
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-8">
        <SkeletonLoader type="card" count={6} />
      </div>
    </div>
  );

  return (
    <div ref={containerRef} className="max-w-6xl mx-auto w-full">
      <div className="gsap-stagger mb-12">
        <h1 className="text-4xl font-bold text-white tracking-tight mb-3">Learning Journey</h1>
        <p className="text-white/60 text-lg">Follow your customized path to mastery.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {data?.activities.map((act: Activity) => {
          let Icon = Circle;
          let iconColor = "text-brand-faint";
          let iconBg = "bg-brand-dark";
          let cardGlow = "card-glow border border-white/5 bg-black/20";
          let actionBtn = "bg-white/5 text-white/50 hover:bg-white/10 hover:text-white";
          let statusLabel = t('notStarted');
          let statusClass = "text-brand-muted";
          let actionLabel = t('start');

          // WP-1.1 RC-01: canonical per-learner statuses with supportive
          // phrasing — "needs-practice" is a nudge, never "failed".
          if (act.status === 'completed') {
            Icon = CheckCircle2;
            iconColor = "text-brand-neon";
            iconBg = "bg-brand-dark";
            cardGlow = "card-glow border border-brand-neon/30 bg-black/40";
            actionBtn = "bg-white/10 text-white/70 hover:bg-white/20";
            statusLabel = t('completed');
            statusClass = "text-brand-neon";
            actionLabel = t('review');
          } else if (act.status === 'needs-practice') {
            Icon = RefreshCcw;
            iconColor = "text-brand-amber";
            iconBg = "bg-brand-dark";
            cardGlow = "card-glow border border-brand-amber/40 bg-black/40";
            actionBtn = "bg-brand-amber text-black hover:bg-brand-amber/90 font-bold";
            statusLabel = t('needsPractice');
            statusClass = "text-brand-amber";
            actionLabel = t('practiceAgain');
          } else if (act.status === 'active') {
            Icon = PlayCircle;
            iconColor = "text-brand-amber";
            iconBg = "bg-brand-dark";
            cardGlow = "card-glow border border-brand-amber/30 bg-black/40";
            actionBtn = "bg-brand-amber text-black hover:bg-brand-amber/90 font-bold";
            statusLabel = t('active');
            statusClass = "text-brand-amber";
            actionLabel = t('continue');
          }

          return (
            <div 
              key={act.id} 
              className={`gsap-stagger ${cardGlow} backdrop-blur-2xl transition-all duration-300 hover:-translate-y-2 flex flex-col h-full`}
            >
              <div className="flex items-center justify-between mb-6">
                <span className={`text-[10px] font-bold uppercase tracking-[0.2em] ${statusClass}`}>
                  {statusLabel}
                </span>
                <span className={`p-2 rounded-full ${iconBg} ${iconColor} transition-transform duration-300`}>
                  <Icon className="w-5 h-5" />
                </span>
              </div>
              
              <div className="flex-1 mb-6">
                <span className={`text-xs font-semibold px-3 py-1 rounded-full mb-3 inline-block bg-white/5 text-white/60`}>
                  {act.topic}
                </span>
                <h3 className="text-xl font-bold text-white mb-2 tracking-tight">{act.title}</h3>
                <p className="text-sm text-white/60 leading-relaxed line-clamp-3">{act.description}</p>

                {/* WP-3.1 RC-07: honest OER metadata — every activity names
                    its license + attribution; nothing is shown when absent. */}
                {(act.license || act.attribution) && (
                  <div className="mt-4 space-y-1">
                    {act.license && (
                      <a
                        href={act.license_url || '#'}
                        target={act.license_url ? '_blank' : undefined}
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-brand-teal/80 hover:text-brand-teal transition-colors"
                      >
                        <span className="w-1.5 h-1.5 rounded-full bg-brand-teal" />
                        {act.license}
                      </a>
                    )}
                    {act.attribution && (
                      <p className="text-[10px] text-brand-muted">
                        {act.attribution}
                      </p>
                    )}
                  </div>
                )}

                {/* WP-3.4 RC-12: NSL caption badge — only when a real caption
                    track exists on the activity. */}
                {act.caption_text && (
                  <span className="mt-2 inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-brand-amber/90">
                    <span className="w-1.5 h-1.5 rounded-full bg-brand-amber" />
                    NSL captions
                  </span>
                )}
              </div>

              <div className="mt-auto pt-6 border-t border-white/5">
                <Link
                  href={`/learning/${act.id}`}
                  className={`w-full inline-flex items-center justify-center gap-2 px-6 py-3 rounded-full text-sm transition-all ${actionBtn}`}
                >
                  {actionLabel}
                  <ChevronRight className="w-4 h-4" />
                </Link>
              </div>
            </div>
          );
        })}

        {(!data?.activities || data.activities.length === 0) && (
           <div className="gsap-stagger col-span-full card-glow bg-black/40 backdrop-blur-3xl border border-white/10 flex flex-col items-center justify-center py-24 text-center">
              <span className="bg-brand-dark rounded-full p-4 text-brand-neon mb-6">
                <ActivityIcon className="w-8 h-8" />
              </span>
              <p className="text-white/60 text-xl font-medium">Your journey hasn&apos;t started yet.</p>
           </div>
        )}
      </div>
    </div>
  );
}
