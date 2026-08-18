"use client";
import { DashboardData, ChartDataPoint } from "@/lib/types";

import { useEffect, useState, useRef } from 'react';
import { fetchWithCache } from '@/lib/api';
import { LineChart as LucideLineChart, BarChart2, Star, TrendingUp } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';

export default function Observation() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    Promise.all([
      fetchWithCache('/dashboard'),
      fetchWithCache('/chart-data')
    ])
      .then(([dashboardRes, chartRes]) => {
        setData(dashboardRes);
        setChartData(chartRes.activity_data);
      })
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
        <p className="text-white/60 text-lg">A clear reflection of your learning habits and progress.</p>
      </div>

      {/* KPI Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
          <div className="absolute top-0 right-0 p-6 opacity-5 text-white group-hover:scale-125 transition-transform duration-500"><BarChart2 className="w-20 h-20"/></div>
          <p className="text-[11px] font-bold text-white/50 uppercase tracking-[0.2em] mb-3 relative z-10">Topics Mastered</p>
          <p className="text-5xl font-extrabold text-white relative z-10 tracking-tight">{data?.progress.completed} <span className="text-xl text-white/30 font-medium">/ {data?.progress.total_topics}</span></p>
        </div>
        
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-brand-neon/30 shadow-[0_0_30px_rgba(0,240,255,0.1)] text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
          <div className="absolute top-0 right-0 p-6 opacity-10 text-brand-neon group-hover:scale-125 transition-transform duration-500"><LucideLineChart className="w-20 h-20"/></div>
          <p className="text-[11px] font-bold text-brand-neon uppercase tracking-[0.2em] mb-3 relative z-10">Current Streak</p>
          <p className="text-5xl font-extrabold text-white relative z-10 tracking-tight">{data?.progress.current_streak} <span className="text-xl text-white/50 font-medium">days</span></p>
        </div>
        
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
          <div className="absolute top-0 right-0 p-6 opacity-5 text-brand-amber group-hover:scale-125 transition-transform duration-500"><Star className="w-20 h-20"/></div>
          <p className="text-[11px] font-bold text-white/50 uppercase tracking-[0.2em] mb-3 relative z-10">Overall Score</p>
          <p className="text-5xl font-extrabold text-brand-amber relative z-10 tracking-tight">{data?.progress.overall_score}%</p>
        </div>
      </div>

      {/* Charts Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
           <h3 className="text-xl font-bold text-white mb-8 flex items-center tracking-tight">
             <TrendingUp className="w-6 h-6 mr-3 text-brand-neon"/> Performance Trend
           </h3>
           <div className="h-72 w-full">
             <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                  <defs>
                    <linearGradient id="colorScore" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#00F0FF" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="#00F0FF" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="name" tick={{fontSize: 12, fill: 'rgba(255,255,255,0.4)'}} axisLine={false} tickLine={false} />
                  <YAxis tick={{fontSize: 12, fill: 'rgba(255,255,255,0.4)'}} axisLine={false} tickLine={false} />
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.05)" />
                  <Tooltip 
                    contentStyle={{ borderRadius: '16px', border: '1px solid rgba(255,255,255,0.1)', backgroundColor: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(16px)', color: '#fff' }} 
                    itemStyle={{ color: '#00F0FF', fontWeight: 'bold' }}
                  />
                  <Area type="monotone" dataKey="score" stroke="#00F0FF" strokeWidth={3} fillOpacity={1} fill="url(#colorScore)" />
                </AreaChart>
             </ResponsiveContainer>
           </div>
        </div>

        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
           <h3 className="text-xl font-bold text-white mb-8 flex items-center tracking-tight">
             <BarChart2 className="w-6 h-6 mr-3 text-brand-amber"/> Daily Engagement
           </h3>
           <div className="h-72 w-full">
             <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                  <XAxis dataKey="name" tick={{fontSize: 12, fill: 'rgba(255,255,255,0.4)'}} axisLine={false} tickLine={false} />
                  <YAxis tick={{fontSize: 12, fill: 'rgba(255,255,255,0.4)'}} axisLine={false} tickLine={false} />
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.05)" />
                  <Tooltip 
                    cursor={{fill: 'rgba(255,255,255,0.05)'}} 
                    contentStyle={{ borderRadius: '16px', border: '1px solid rgba(255,255,255,0.1)', backgroundColor: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(16px)', color: '#fff' }} 
                    itemStyle={{ color: '#FFB703', fontWeight: 'bold' }}
                  />
                  <Bar dataKey="duration" fill="#FFB703" radius={[6, 6, 0, 0]} />
                </BarChart>
             </ResponsiveContainer>
           </div>
        </div>
      </div>
    </div>
  );
}
