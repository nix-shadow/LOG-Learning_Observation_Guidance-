"use client";
import { DashboardData, ChartDataPoint } from "@/lib/types";

import { useEffect, useState, useRef } from 'react';
import dynamic from 'next/dynamic';
import { fetchWithCache } from '@/lib/api';
import { BarChart2 } from 'lucide-react';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';

const ChartSection = dynamic(() => import('./ChartSection'), {
  ssr: false,
  loading: () => (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
      {[0, 1].map((i) => (
        <div key={i} className="card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
          <div className="h-72 w-full animate-pulse bg-white/5 rounded-2xl" />
        </div>
      ))}
    </div>
  ),
});

export default function Observation() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [asOf, setAsOf] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // F17: dashboard and chart-data are independent — a chart failure must
    // not blank out the KPI section (and vice versa).
    fetchWithCache('/dashboard')
      .then((res) => {
        setData(res);
        if (typeof res?.as_of === 'string') setAsOf(res.as_of);
      })
      .catch(console.error)
      .finally(() => setLoading(false));

    fetchWithCache('/chart-data')
      .then((res) => {
        setChartData(Array.isArray(res?.activity_data) ? res.activity_data : []);
        if (typeof res?.as_of === 'string') setAsOf(res.as_of);
      })
      .catch(console.error);
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

  return (
    <div ref={containerRef} className="max-w-5xl mx-auto w-full space-y-10">
      <div className="gsap-stagger">
        <h1 className="text-4xl font-bold text-white tracking-tight mb-3">Observation</h1>
        <p className="text-brand-muted text-lg">A clear reflection of your learning habits and progress.</p>
      </div>

      {/* KPI Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
          <div className="absolute top-0 right-0 p-6 opacity-5 text-white group-hover:scale-125 transition-transform duration-500"><BarChart2 className="w-20 h-20"/></div>
          <p className="text-[11px] font-bold text-brand-muted uppercase tracking-[0.2em] mb-3 relative z-10">Topics Mastered</p>
          <p className="text-5xl font-extrabold text-white relative z-10 tracking-tight">{data?.progress.completed ?? 0} <span className="text-xl text-brand-muted font-medium">/ {data?.progress.total_topics ?? 0}</span></p>
        </div>
        
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-brand-neon/30 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
          <div className="absolute top-0 right-0 p-6 opacity-10 text-brand-neon group-hover:scale-125 transition-transform duration-500"><BarChart2 className="w-20 h-20"/></div>
          <p className="text-[11px] font-bold text-brand-neon uppercase tracking-[0.2em] mb-3 relative z-10">Current Streak</p>
          <p className="text-5xl font-extrabold text-white relative z-10 tracking-tight">{data?.progress.current_streak ?? 0} <span className="text-xl text-brand-muted font-medium">days</span></p>
        </div>
        
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
          <div className="absolute top-0 right-0 p-6 opacity-5 text-brand-amber group-hover:scale-125 transition-transform duration-500"><BarChart2 className="w-20 h-20"/></div>
          <p className="text-[11px] font-bold text-brand-muted uppercase tracking-[0.2em] mb-3 relative z-10">Overall Score</p>
          <p className="text-5xl font-extrabold text-brand-amber relative z-10 tracking-tight">{data?.progress.overall_score ?? 0}%</p>
        </div>
      </div>

      {/* Charts Section — recharts chunked behind dynamic import (wp03t04) */}
      <ChartSection chartData={chartData} asOf={asOf} />
    </div>
  );
}