"use client";
import { DashboardData, Guidance as GuidanceType } from "@/lib/types";

import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import Link from 'next/link';
import { Compass, ArrowRight, Lightbulb, RefreshCw, BookOpen } from 'lucide-react';

export default function Guidance() {
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

  const getIconForType = (type: string) => {
    switch (type.toLowerCase()) {
      case 'next_step': return <ArrowRight className="w-6 h-6 text-brand-teal" />;
      case 'practice': return <RefreshCw className="w-6 h-6 text-brand-amber" />;
      case 'insight': return <Lightbulb className="w-6 h-6 text-brand-blue" />;
      default: return <BookOpen className="w-6 h-6 text-gray-400" />;
    }
  };

  return (
    <div className="max-w-4xl mx-auto w-full">
      <div className="flex items-center mb-2">
        <Compass className="w-8 h-8 text-brand-teal mr-3" />
        <h1 className="text-3xl font-bold text-brand-blue">Guidance</h1>
      </div>
      <p className="text-gray-600 mb-8">Actionable recommendations based on your recent observations.</p>

      <div className="space-y-6">
        {data?.guidance.map((g: GuidanceType) => (
          <div key={g.id} className="card relative overflow-hidden group hover:border-brand-teal/50 transition-colors">
            <div className="absolute top-0 left-0 w-1.5 h-full bg-brand-teal group-hover:bg-brand-amber transition-colors"></div>
            <div className="pl-4 flex flex-col sm:flex-row gap-6 items-start sm:items-center justify-between">
              <div className="flex items-start gap-4">
                <div className="mt-1 flex-shrink-0 bg-brand-gray/20 p-3 rounded-xl">
                  {getIconForType(g.type)}
                </div>
                <div>
                  <span className="text-xs font-bold uppercase tracking-wider text-gray-500 mb-1 block">
                    {g.type.replace('_', ' ')}
                  </span>
                  <p className="text-lg text-brand-text font-medium">{g.text}</p>
                </div>
              </div>

              {g.action && (
                <Link href={g.action} className="btn-primary whitespace-nowrap flex-shrink-0 mt-4 sm:mt-0">
                  Take Action
                </Link>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
