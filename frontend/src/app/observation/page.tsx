"use client";
import { DashboardData, Observation as ObservationType } from "@/lib/types";

import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { LineChart, BarChart2, Star, Target } from 'lucide-react';

export default function Observation() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchWithCache('/dashboard')
      .then(setData)
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
      case 'consistency': return <LineChart className="w-5 h-5 text-brand-teal" />;
      default: return <BarChart2 className="w-5 h-5 text-brand-blue" />;
    }
  };

  return (
    <div className="max-w-5xl mx-auto w-full">
      <h1 className="text-3xl font-bold text-brand-blue mb-2">Observation</h1>
      <p className="text-gray-600 mb-8">A clear reflection of your learning habits and progress.</p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
        <div className="card text-center">
          <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-2">Topics Mastered</p>
          <p className="text-4xl font-extrabold text-brand-blue">{data?.progress.completed} <span className="text-lg text-gray-400 font-medium">/ {data?.progress.total_topics}</span></p>
        </div>
        <div className="card text-center">
          <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-2">Current Streak</p>
          <p className="text-4xl font-extrabold text-brand-teal">{data?.progress.current_streak} <span className="text-lg text-gray-400 font-medium">days</span></p>
        </div>
        <div className="card text-center">
          <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-2">Overall Score</p>
          <p className="text-4xl font-extrabold text-brand-amber">{data?.progress.overall_score}%</p>
        </div>
      </div>

      <h2 className="text-xl font-bold text-brand-blue mb-4">Detailed Insights</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {data?.observations.map((obs: ObservationType) => (
          <div key={obs.id} className="card flex items-start gap-4">
            <div className="mt-1 p-2 bg-brand-gray/30 rounded-full flex-shrink-0">
              {getIconForCategory(obs.category)}
            </div>
            <div>
              <h3 className="text-sm font-bold uppercase tracking-wider text-gray-500 mb-1">{obs.category}</h3>
              <p className="text-brand-text">{obs.text}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
