"use client";

import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import Link from 'next/link';
import { CheckCircle2, ArrowRight, Activity, TrendingUp, Medal, Flame } from 'lucide-react';
import { DashboardData, Activity as ActivityType, Guidance as GuidanceType, Observation as ObservationType } from "@/lib/types";
import { CircularProgressbar, buildStyles } from 'react-circular-progressbar';
import 'react-circular-progressbar/dist/styles.css';
import { motion } from 'framer-motion';

export default function Dashboard() {
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

  if (!data) return (
    <div className="text-center py-20">
      <h2 className="text-2xl font-bold text-brand-blue mb-2">Welcome to LOG</h2>
      <p className="text-gray-600 mb-6">Your learning journey starts here. (Offline mode active, no cached data found).</p>
    </div>
  );

  const dailyGoalPercentage = 75; // Hardcoded for MVP, ideally fetched from backend

  return (
    <div className="space-y-8">
      {/* Welcome Area */}
      <section className="bg-brand-blue text-white rounded-3xl p-8 shadow-lg relative overflow-hidden flex flex-col md:flex-row items-center justify-between gap-8">
        <div className="relative z-10 w-full md:w-2/3">
          <h1 className="text-3xl font-bold mb-2">Hello, {data.learner.name}</h1>
          <p className="text-brand-gray/80 text-lg mb-4">You are on a {data.progress.current_streak}-day learning streak. Keep it up!</p>

          <div className="flex flex-wrap gap-3">
             <div className="bg-white/10 px-4 py-2 rounded-full flex items-center gap-2 backdrop-blur-sm border border-white/20">
                <Flame className="w-5 h-5 text-brand-amber" />
                <span className="font-semibold">{data.progress.current_streak} Day Streak</span>
             </div>
             <div className="bg-white/10 px-4 py-2 rounded-full flex items-center gap-2 backdrop-blur-sm border border-white/20">
                <Medal className="w-5 h-5 text-brand-teal" />
                <span className="font-semibold">Logic Master Badge</span>
             </div>
          </div>
        </div>

        <div className="relative z-10 w-32 h-32 flex-shrink-0 bg-white rounded-full p-2">
            <CircularProgressbar
              value={dailyGoalPercentage}
              text={`${dailyGoalPercentage}%`}
              styles={buildStyles({
                pathColor: '#00B4D8',
                textColor: '#0A2540',
                trailColor: '#E9ECEF',
                textSize: '24px',
                pathTransitionDuration: 1.5
              })}
            />
            <div className="text-center mt-2 text-xs font-bold uppercase text-brand-teal absolute -bottom-6 left-0 right-0">Daily Goal</div>
        </div>

        <div className="absolute top-0 right-0 -mt-16 -mr-16 w-64 h-64 bg-brand-teal/20 rounded-full blur-3xl"></div>
      </section>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Content Area (Left 2 columns) */}
        <div className="lg:col-span-2 space-y-8">

          {/* Current Focus (Guidance) */}
          <section>
            <h2 className="text-xl font-bold text-brand-blue mb-4 flex items-center">
              <TrendingUp className="w-5 h-5 mr-2 text-brand-teal" /> Current Focus
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {data.guidance.map((g: GuidanceType, i) => (
                <motion.div
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.1 }}
                  key={g.id}
                  className="card border-l-4 border-l-brand-amber hover:shadow-md transition-shadow"
                >
                  <p className="text-brand-text font-medium mb-3">{g.text}</p>
                  {g.action && (
                    <Link href={g.action} className="text-brand-teal text-sm font-semibold flex items-center hover:underline">
                      Take action <ArrowRight className="w-4 h-4 ml-1" />
                    </Link>
                  )}
                </motion.div>
              ))}
            </div>
          </section>

          {/* Learning Journey Overview */}
          <section>
            <div className="flex justify-between items-end mb-4">
              <h2 className="text-xl font-bold text-brand-blue flex items-center">
                <Activity className="w-5 h-5 mr-2 text-brand-teal" /> Learning Journey
              </h2>
              <Link href="/learning" className="text-sm font-semibold text-brand-teal hover:underline">View all</Link>
            </div>
            <div className="card">
              <div className="space-y-4">
                {data.activities.slice(0, 3).map((act: ActivityType) => (
                  <div key={act.id} className="flex items-start pb-4 border-b border-brand-gray last:border-0 last:pb-0">
                    <div className={`mt-1 flex-shrink-0 ${act.status === 'Completed' ? 'text-brand-teal' : 'text-gray-300'}`}>
                      <CheckCircle2 className="w-5 h-5" />
                    </div>
                    <div className="ml-3">
                      <p className="text-sm font-bold text-brand-blue">{act.title}</p>
                      <p className="text-xs text-gray-500">{act.status}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </section>

        </div>

        {/* Sidebar (Right column) - Observations */}
        <div className="space-y-8">
          <section>
            <h2 className="text-xl font-bold text-brand-blue mb-4">Recent Observations</h2>
            <div className="card space-y-4 bg-brand-gray/10 border-none shadow-none">
              {data.observations.map((obs: ObservationType) => (
                <div key={obs.id} className="pb-3 border-b border-white last:border-0 last:pb-0">
                  <span className="text-xs font-bold uppercase text-gray-500 tracking-wider mb-1 block">{obs.category}</span>
                  <p className="text-sm text-brand-text">{obs.text}</p>
                </div>
              ))}
              <div className="pt-2">
                <Link href="/observation" className="text-sm font-semibold text-brand-teal hover:underline">View full observation</Link>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
