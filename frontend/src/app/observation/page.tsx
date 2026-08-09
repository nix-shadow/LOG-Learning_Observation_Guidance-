"use client";
import { DashboardData, Observation as ObservationType, ChartDataPoint } from "@/lib/types";

import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { LineChart as LucideLineChart, BarChart2, Star, Target, TrendingUp } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { motion } from 'framer-motion';

export default function Observation() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [loading, setLoading] = useState(true);

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

  if (loading) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-teal"></div>
    </div>
  );

  const getIconForCategory = (category: string) => {
    switch (category.toLowerCase()) {
      case 'strengths': return <Star className="w-5 h-5 text-brand-amber" />;
      case 'areas needing improvement': return <Target className="w-5 h-5 text-red-400" />;
      case 'consistency': return <LucideLineChart className="w-5 h-5 text-brand-teal" />;
      default: return <BarChart2 className="w-5 h-5 text-brand-blue" />;
    }
  };

  return (
    <div className="max-w-5xl mx-auto w-full">
      <h1 className="text-3xl font-bold text-brand-blue mb-2">Observation</h1>
      <p className="text-gray-600 mb-8">A clear reflection of your learning habits and progress.</p>

      {/* KPI Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
        <motion.div initial={{ scale: 0.95, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} className="card text-center relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10 text-brand-blue"><BarChart2 className="w-12 h-12"/></div>
          <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-2 relative z-10">Topics Mastered</p>
          <p className="text-4xl font-extrabold text-brand-blue relative z-10">{data?.progress.completed} <span className="text-lg text-gray-400 font-medium">/ {data?.progress.total_topics}</span></p>
        </motion.div>
        <motion.div initial={{ scale: 0.95, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} transition={{ delay: 0.1 }} className="card text-center relative overflow-hidden border-brand-teal/20">
          <div className="absolute top-0 right-0 p-4 opacity-10 text-brand-teal"><LucideLineChart className="w-12 h-12"/></div>
          <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-2 relative z-10">Current Streak</p>
          <p className="text-4xl font-extrabold text-brand-teal relative z-10">{data?.progress.current_streak} <span className="text-lg text-gray-400 font-medium">days</span></p>
        </motion.div>
        <motion.div initial={{ scale: 0.95, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} transition={{ delay: 0.2 }} className="card text-center relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10 text-brand-amber"><Star className="w-12 h-12"/></div>
          <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-2 relative z-10">Overall Score</p>
          <p className="text-4xl font-extrabold text-brand-amber relative z-10">{data?.progress.overall_score}%</p>
        </motion.div>
      </div>

      {/* Charts Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-10">
        <div className="card">
           <h3 className="text-lg font-bold text-brand-blue mb-6 flex items-center"><TrendingUp className="w-5 h-5 mr-2 text-brand-teal"/> Performance Trend</h3>
           <div className="h-64 w-full">
             <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                  <defs>
                    <linearGradient id="colorScore" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#00B4D8" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="#00B4D8" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="name" tick={{fontSize: 12, fill: '#6B7280'}} axisLine={false} tickLine={false} />
                  <YAxis tick={{fontSize: 12, fill: '#6B7280'}} axisLine={false} tickLine={false} />
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E5E7EB" />
                  <Tooltip contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }} />
                  <Area type="monotone" dataKey="score" stroke="#00B4D8" strokeWidth={3} fillOpacity={1} fill="url(#colorScore)" />
                </AreaChart>
             </ResponsiveContainer>
           </div>
        </div>

        <div className="card">
           <h3 className="text-lg font-bold text-brand-blue mb-6 flex items-center"><BarChart2 className="w-5 h-5 mr-2 text-brand-amber"/> Daily Engagement (Mins)</h3>
           <div className="h-64 w-full">
             <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                  <XAxis dataKey="name" tick={{fontSize: 12, fill: '#6B7280'}} axisLine={false} tickLine={false} />
                  <YAxis tick={{fontSize: 12, fill: '#6B7280'}} axisLine={false} tickLine={false} />
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E5E7EB" />
                  <Tooltip cursor={{fill: '#F3F4F6'}} contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }} />
                  <Bar dataKey="duration" fill="#FFB703" radius={[4, 4, 0, 0]} />
                </BarChart>
             </ResponsiveContainer>
           </div>
        </div>
      </div>

      <h2 className="text-xl font-bold text-brand-blue mb-4">Detailed Insights</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {data?.observations.map((obs: ObservationType, i) => (
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: i * 0.1 }}
            key={obs.id}
            className="card flex items-start gap-4 hover:border-brand-teal/30 transition-colors"
          >
            <div className="mt-1 p-2 bg-brand-gray/30 rounded-full flex-shrink-0">
              {getIconForCategory(obs.category)}
            </div>
            <div>
              <h3 className="text-sm font-bold uppercase tracking-wider text-gray-500 mb-1">{obs.category}</h3>
              <p className="text-brand-text">{obs.text}</p>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
