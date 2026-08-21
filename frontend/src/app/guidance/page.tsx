"use client";
import { DashboardData, Guidance as GuidanceType } from "@/lib/types";

import { useEffect, useState, useRef } from 'react';
import { fetchWithCache } from '@/lib/api';
import Link from 'next/link';
import { Compass, ArrowRight, Lightbulb, RefreshCw, BookOpen } from 'lucide-react';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';

export default function Guidance() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchWithCache('/dashboard')
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
  }, { dependencies: [loading, data], scope: containerRef });

  if (loading) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="w-16 h-16 border-4 border-white/10 border-t-brand-neon rounded-full animate-spin"></div>
    </div>
  );

  const getIconForType = (type: string) => {
    switch (type.toLowerCase()) {
      case 'next_step': return <ArrowRight className="w-6 h-6 text-brand-neon" />;
      case 'practice': return <RefreshCw className="w-6 h-6 text-brand-amber" />;
      case 'insight': return <Lightbulb className="w-6 h-6 text-brand-teal" />;
      default: return <BookOpen className="w-6 h-6 text-white/50" />;
    }
  };

  return (
    <div ref={containerRef} className="max-w-4xl mx-auto w-full">
      <div className="gsap-stagger mb-12 flex items-center">
        <Compass className="w-10 h-10 text-brand-neon mr-4" />
        <div>
          <h1 className="text-4xl font-bold text-white tracking-tight mb-2">Guidance</h1>
          <p className="text-white/60 text-lg">Actionable recommendations based on your recent observations.</p>
        </div>
      </div>

      <div className="space-y-6">
        {data?.guidance.map((g: GuidanceType) => (
          <div key={g.id} className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 relative overflow-hidden group hover:border-brand-neon/50 transition-all duration-300 hover:-translate-y-1">
            <div className="absolute top-0 left-0 w-1.5 h-full bg-white/10 group-hover:bg-brand-neon transition-colors duration-500"></div>
            <div className="pl-8 py-4 pr-6 flex flex-col sm:flex-row gap-6 items-start sm:items-center justify-between">
              <div className="flex items-start gap-5">
                <div className="mt-1 flex-shrink-0 bg-white/5 p-4 rounded-2xl border border-white/10">
                  {getIconForType(g.type)}
                </div>
                <div>
                  <span className="text-[11px] font-bold uppercase tracking-[0.2em] text-brand-muted mb-2 block">
                    {g.type.replace('_', ' ')}
                  </span>
                  <p className="text-xl text-white font-medium leading-relaxed tracking-tight">{g.text}</p>
                </div>
              </div>

              {g.action && (
                <Link href={g.action} className="btn-primary whitespace-nowrap flex-shrink-0 mt-4 sm:mt-0 text-sm py-3 px-6">
                  Take Action
                </Link>
              )}
            </div>
          </div>
        ))}
        {(!data?.guidance || data.guidance.length === 0) && (
           <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 flex flex-col items-center justify-center py-20 text-center">
               <Lightbulb className="w-12 h-12 text-brand-faint mb-4" />
               <p className="text-white/60 text-lg">No new guidance available right now. Keep learning!</p>
           </div>
        )}
      </div>
    </div>
  );
}
