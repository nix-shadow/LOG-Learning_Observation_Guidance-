"use client";
import { LearningJourneyData, Activity } from "@/lib/types";

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { fetchWithCache } from '@/lib/api';
import { CheckCircle2, Circle, PlayCircle, ChevronRight } from 'lucide-react';
import { motion } from 'framer-motion';
import SkeletonLoader from '@/components/SkeletonLoader';

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
    <div className="max-w-4xl mx-auto w-full space-y-6">
      <SkeletonLoader type="text" count={2} />
      <div className="ml-6 space-y-4">
        <SkeletonLoader type="card" count={4} />
      </div>
    </div>
  );

  return (
    <motion.div 
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="max-w-4xl mx-auto w-full"
    >
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: "spring", stiffness: 300, damping: 25 }}
      >
        <h1 className="text-3xl font-bold text-brand-blue mb-2">Learning Journey</h1>
        <p className="text-gray-600 mb-8">Follow your customized path to mastery.</p>
      </motion.div>

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
            <motion.div 
              key={act.id} 
              className="relative pl-8 md:pl-12 group"
              initial={{ opacity: 0, x: -50 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true, margin: "-50px" }}
              transition={{ type: "spring", stiffness: 300, damping: 25, delay: index * 0.1 }}
            >
              <motion.span 
                className={`absolute -left-[11px] bg-white text-brand-white rounded-full ${iconColor} z-10`}
                whileHover={{ scale: 1.2 }}
                transition={{ type: "spring", stiffness: 400, damping: 10 }}
              >
                <Icon className="w-5 h-5 bg-white rounded-full shadow-sm" />
              </motion.span>

              <div className={`card ${bgColor} transition-all duration-300 hover:shadow-md hover:-translate-y-1 hover:border-brand-teal/50`}>
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                  <div>
                    <span className="text-xs font-bold uppercase tracking-wider text-gray-500 mb-1 block">Module {index + 1} &bull; {act.topic}</span>
                    <h3 className="text-lg font-bold text-brand-blue">{act.title}</h3>
                    <p className="text-sm text-gray-600 mt-1">{act.description}</p>
                  </div>

                  <div>
                    <Link
                      href={`/learning/${act.id}`}
                      className={`inline-flex items-center gap-1 px-4 py-2 rounded-full text-sm font-semibold transition-colors ${
                        act.status === 'Completed' ? 'bg-gray-100 text-gray-600 hover:bg-gray-200' :
                        act.status === 'In progress' ? 'bg-brand-amber text-white hover:bg-brand-amber/90' :
                        'bg-brand-teal text-white hover:bg-brand-teal/90'
                      }`}
                    >
                      {act.status === 'Completed' ? 'Review' : 'Continue'}
                      <ChevronRight className="w-4 h-4" />
                    </Link>
                  </div>
                </div>
              </div>
            </motion.div>
          );
        })}
      </div>
    </motion.div>
  );
}
