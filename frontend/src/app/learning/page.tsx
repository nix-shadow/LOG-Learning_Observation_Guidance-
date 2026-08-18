"use client";
import { LearningJourneyData, Activity } from "@/lib/types";

import { useEffect, useState, useRef } from 'react';
import Link from 'next/link';
import { fetchWithCache } from '@/lib/api';
import { CheckCircle2, Circle, PlayCircle, ChevronRight, Activity as ActivityIcon } from 'lucide-react';
import SkeletonLoader from '@/components/SkeletonLoader';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';

export default function Learning() {
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
        {data?.activities.map((act: Activity, index: number) => {
          let Icon = Circle;
          let iconColor = "text-white/30";
          let iconBg = "bg-brand-dark";
          let cardGlow = "card-glow border border-white/5 bg-black/20";
          let actionBtn = "bg-white/5 text-white/50 hover:bg-white/10 hover:text-white";

          if (act.status === 'Completed') {
            Icon = CheckCircle2;
            iconColor = "text-brand-neon";
            iconBg = "bg-brand-dark drop-shadow-[0_0_10px_rgba(0,240,255,0.8)]";
            cardGlow = "card-glow border border-brand-neon/30 bg-black/40";
            actionBtn = "bg-white/10 text-white/70 hover:bg-white/20";
          } else if (act.status === 'In progress') {
            Icon = PlayCircle;
            iconColor = "text-brand-amber drop-shadow-[0_0_10px_rgba(255,183,3,0.8)]";
            iconBg = "bg-brand-dark";
            cardGlow = "card-glow border border-brand-amber/30 bg-black/40";
            actionBtn = "bg-brand-amber text-black hover:bg-brand-amber/90 font-bold";
          }

          return (
            <div 
              key={act.id} 
              className={`gsap-stagger ${cardGlow} backdrop-blur-2xl transition-all duration-300 hover:-translate-y-2 hover:shadow-glow flex flex-col h-full`}
            >
              <div className="flex items-center justify-between mb-6">
                <span className={`text-[10px] font-bold uppercase tracking-[0.2em] ${act.status === 'Completed' ? 'text-brand-neon' : act.status === 'In progress' ? 'text-brand-amber' : 'text-white/40'}`}>
                  Module {index + 1}
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
              </div>

              <div className="mt-auto pt-6 border-t border-white/5">
                <Link
                  href={`/learning/${act.id}`}
                  className={`w-full inline-flex items-center justify-center gap-2 px-6 py-3 rounded-full text-sm transition-all ${actionBtn}`}
                >
                  {act.status === 'Completed' ? 'Review' : 'Continue'}
                  <ChevronRight className="w-4 h-4" />
                </Link>
              </div>
            </div>
          );
        })}

        {(!data?.activities || data.activities.length === 0) && (
           <div className="gsap-stagger col-span-full card-glow bg-black/40 backdrop-blur-3xl border border-white/10 flex flex-col items-center justify-center py-24 text-center">
              <span className="bg-brand-dark rounded-full p-4 text-brand-neon mb-6">
                <ActivityIcon className="w-8 h-8 animate-pulse-glow" />
              </span>
              <p className="text-white/60 text-xl font-medium">Your journey hasn&apos;t started yet.</p>
           </div>
        )}
      </div>
    </div>
  );
}
