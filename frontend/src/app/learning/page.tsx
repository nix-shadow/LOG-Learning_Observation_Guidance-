"use client";
import { LearningJourneyData, Activity } from "@/lib/types";

import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { CheckCircle2, Circle, PlayCircle } from 'lucide-react';

export default function Learning() {
  const [data, setData] = useState<LearningJourneyData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchWithCache('/learning-journey')
      .then(setData)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  if (loading) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-teal"></div>
    </div>
  );

  return (
    <div className="max-w-4xl mx-auto w-full">
      <h1 className="text-3xl font-bold text-brand-blue mb-2">Learning Journey</h1>
      <p className="text-gray-600 mb-8">Follow your customized path to mastery.</p>

      <div className="relative border-l-2 border-brand-gray ml-3 md:ml-6 space-y-12">
        {data?.activities.map((act: Activity, index: number) => {
          let Icon = Circle;
          let iconColor = "text-gray-300";
          let bgColor = "bg-white";

          if (act.status === 'Completed') {
            Icon = CheckCircle2;
            iconColor = "text-brand-teal";
          } else if (act.status === 'In progress') {
            Icon = PlayCircle;
            iconColor = "text-brand-amber";
            bgColor = "bg-brand-amber/10";
          }

          return (
            <div key={act.id} className="relative pl-8 md:pl-12">
              <span className={`absolute -left-[11px] bg-white text-brand-white rounded-full ${iconColor}`}>
                <Icon className="w-5 h-5 bg-white rounded-full" />
              </span>

              <div className={`card ${bgColor} transition-colors hover:border-brand-teal/50`}>
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                  <div>
                    <span className="text-xs font-bold uppercase tracking-wider text-gray-500 mb-1 block">Module {index + 1} &bull; {act.topic}</span>
                    <h3 className="text-lg font-bold text-brand-blue">{act.title}</h3>
                    <p className="text-sm text-gray-600 mt-1">{act.description}</p>
                  </div>

                  <div>
                    <button className={`px-4 py-2 rounded-full text-sm font-semibold transition-colors ${
                      act.status === 'Completed' ? 'bg-gray-100 text-gray-600' :
                      act.status === 'In progress' ? 'bg-brand-amber text-white' :
                      'bg-brand-teal text-white'
                    }`}>
                      {act.status === 'Completed' ? 'Review' : 'Continue'}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
