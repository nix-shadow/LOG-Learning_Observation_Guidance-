"use client";

import { useEffect, useState } from 'react';
import { fetchWithCache, logout } from '@/lib/api';
import Link from 'next/link';
import { CheckCircle2, ArrowRight, Activity, TrendingUp, Medal, Flame, Download, Upload, LogOut } from 'lucide-react';
import { DashboardData, Activity as ActivityType, Guidance as GuidanceType, Observation as ObservationType } from "@/lib/types";
import { CircularProgressbar, buildStyles } from 'react-circular-progressbar';
import 'react-circular-progressbar/dist/styles.css';
import SkeletonLoader from '@/components/SkeletonLoader';
import { motion } from 'framer-motion';
import { downloadSyncFile, importSyncFile } from '@/lib/syncExport';
import toast from 'react-hot-toast';

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
    <div className="space-y-8">
      <SkeletonLoader type="card" count={1} />
      <SkeletonLoader type="stats" count={3} />
      <SkeletonLoader type="card" count={2} />
    </div>
  );

  if (!data || !data.learner) return (
    <motion.div 
      initial={{ opacity: 0, scale: 0.95, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 25 }}
      className="text-center py-20 card flex flex-col items-center justify-center max-w-2xl mx-auto mt-10"
    >
      <div className="w-16 h-16 bg-brand-gray/50 rounded-full flex items-center justify-center mb-4">
        <Activity className="w-8 h-8 text-brand-teal opacity-50" />
      </div>
      <h2 className="text-2xl font-bold text-brand-blue mb-2">Welcome to LOG</h2>
      <p className="text-gray-600 mb-6">Your learning journey starts here. (Offline mode active, no cached data found).</p>
      <Link href="/learning" className="btn-primary">Start Learning</Link>
    </motion.div>
  );

  // Daily goal = actual completion progress, derived from backend data.
  const dailyGoalPercentage = data.progress.total_topics > 0
    ? Math.min(100, Math.round((data.progress.completed / data.progress.total_topics) * 100))
    : 0;

  return (
    <motion.div 
      initial="hidden"
      animate="visible"
      variants={{
        hidden: { opacity: 0 },
        visible: { opacity: 1, transition: { staggerChildren: 0.1 } }
      }}
      className="space-y-6"
    >
      {/* Welcome Area (Hero Bento) */}
      <motion.section 
        variants={{ hidden: { opacity: 0, y: 20 }, visible: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 25 } } }}
        className="bg-brand-blue text-white rounded-[24px] p-8 shadow-bento relative overflow-hidden flex flex-col md:flex-row items-center justify-between gap-8"
      >
        <div className="relative z-10 w-full md:w-2/3">
          <h1 className="text-3xl font-bold mb-2">Hello, {data.learner.name}</h1>
          <p className="text-brand-gray/80 text-lg mb-4">You are on a {data.progress.current_streak}-day learning streak. Keep it up!</p>
          <div className="flex flex-wrap gap-3 items-center">
             <div className="bg-white/10 px-4 py-2 rounded-full flex items-center gap-2 backdrop-blur-md border border-white/20 shadow-sm">
                <Flame className="w-5 h-5 text-brand-amber" />
                <span className="font-semibold">{data.progress.current_streak} Day Streak</span>
             </div>
             <div className="bg-white/10 px-4 py-2 rounded-full flex items-center gap-2 backdrop-blur-md border border-white/20 shadow-sm">
                <Medal className="w-5 h-5 text-brand-teal" />
                <span className="font-semibold">Logic Master Badge</span>
             </div>
             <button
               onClick={() => logout()}
               className="bg-white/10 hover:bg-white/20 transition-colors px-4 py-2 rounded-full flex items-center gap-2 backdrop-blur-md border border-white/20 text-white text-sm font-semibold shadow-sm"
               title="Logout"
             >
               <LogOut className="w-4 h-4" />
               <span>Logout</span>
             </button>
          </div>
        </div>

        <div className="relative z-10 w-32 h-32 flex-shrink-0 bg-white/5 backdrop-blur-md rounded-full p-2 border border-white/10 shadow-glow">
            <CircularProgressbar
              value={dailyGoalPercentage}
              text={`${dailyGoalPercentage}%`}
              styles={buildStyles({
                pathColor: '#00B4D8',
                textColor: '#F8F9FA',
                trailColor: 'rgba(255,255,255,0.1)',
                textSize: '24px',
                pathTransitionDuration: 1.5
              })}
            />
            <div className="text-center mt-3 text-xs font-bold uppercase text-brand-teal absolute -bottom-6 left-0 right-0 tracking-wider">Daily Goal</div>
        </div>

        <div className="absolute top-0 right-0 -mt-16 -mr-16 w-64 h-64 bg-brand-teal/20 rounded-full blur-3xl pointer-events-none"></div>
      </motion.section>

      {/* Bento Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content Area (Left 2 columns) */}
        <div className="lg:col-span-2 space-y-6 flex flex-col">

          {/* Current Focus (Guidance) */}
          <motion.section variants={{ hidden: { opacity: 0, y: 20 }, visible: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 25 } } }}>
            <h2 className="text-xl font-bold text-brand-blue mb-4 flex items-center">
              <TrendingUp className="w-5 h-5 mr-2 text-brand-teal" /> Current Focus
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {data.guidance.map((g: GuidanceType) => (
                <div
                  key={g.id}
                  className="card border-l-4 border-l-brand-amber hover:-translate-y-1 transition-transform duration-300"
                >
                  <p className="text-brand-text font-medium mb-3">{g.text}</p>
                  {g.action && (
                    <Link href={g.action} className="text-brand-teal text-sm font-semibold flex items-center hover:underline group">
                      Take action <ArrowRight className="w-4 h-4 ml-1 group-hover:translate-x-1 transition-transform" />
                    </Link>
                  )}
                </div>
              ))}
            </div>
          </motion.section>

          {/* Learning Journey Overview */}
          <motion.section variants={{ hidden: { opacity: 0, y: 20 }, visible: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 25 } } }} className="flex-1 flex flex-col">
            <div className="flex justify-between items-end mb-4">
              <h2 className="text-xl font-bold text-brand-blue flex items-center">
                <Activity className="w-5 h-5 mr-2 text-brand-teal" /> Learning Journey
              </h2>
              <Link href="/learning" className="text-sm font-semibold text-brand-teal hover:underline">View all</Link>
            </div>
            <div className="card flex-1">
              <div className="space-y-4">
                {data.activities.slice(0, 3).map((act: ActivityType) => (
                  <div key={act.id} className="flex items-start pb-4 border-b border-brand-gray last:border-0 last:pb-0">
                    <div className={`mt-1 flex-shrink-0 ${act.status === 'Completed' ? 'text-brand-teal' : 'text-brand-gray/80'}`}>
                      <CheckCircle2 className="w-5 h-5" />
                    </div>
                    <div className="ml-3">
                      <p className="text-sm font-bold text-brand-blue">{act.title}</p>
                      <p className="text-xs text-gray-500 font-medium mt-0.5">{act.status}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </motion.section>

        </div>

        {/* Sidebar (Right column) - Observations & Sync */}
        <div className="space-y-6 flex flex-col">
          <motion.section variants={{ hidden: { opacity: 0, x: 20 }, visible: { opacity: 1, x: 0, transition: { type: "spring", stiffness: 300, damping: 25 } } }}>
            <h2 className="text-xl font-bold text-brand-blue mb-4">Recent Observations</h2>
            <div className="card space-y-4 bg-brand-gray/30 border-none shadow-none">
              {data.observations.map((obs: ObservationType) => (
                <div key={obs.id} className="pb-3 border-b border-brand-white last:border-0 last:pb-0">
                  <span className="text-xs font-bold uppercase text-gray-500 tracking-wider mb-1 block">{obs.category}</span>
                  <p className="text-sm text-brand-text">{obs.text}</p>
                </div>
              ))}
              <div className="pt-2">
                <Link href="/observation" className="text-sm font-semibold text-brand-teal hover:underline">View full observation</Link>
              </div>
            </div>
          </motion.section>

          {/* Sync Offline Progress */}
          <motion.section variants={{ hidden: { opacity: 0, x: 20 }, visible: { opacity: 1, x: 0, transition: { type: "spring", stiffness: 300, damping: 25 } } }} className="flex-1">
            <h2 className="text-xl font-bold text-brand-blue mb-4">Offline Sync</h2>
            <div className="card space-y-4 h-[calc(100%-2.5rem)] flex flex-col justify-between">
              <p className="text-sm text-gray-600">
                Working offline? Download your progress and bring it to school on a USB drive.
              </p>
              <div className="space-y-3 mt-auto">
                <button 
                  onClick={async () => {
                    try {
                      await downloadSyncFile();
                      toast.success('Sync file downloaded successfully!');
                    } catch (e: unknown) {
                      toast.error((e as Error).message || 'Failed to download sync file.');
                    }
                  }}
                  className="btn-primary w-full flex items-center justify-center gap-2"
                >
                  <Download className="w-4 h-4" /> Export Progress
                </button>
                
                <div className="relative">
                  <input 
                    type="file" 
                    accept=".logsync"
                    className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                    onChange={async (e) => {
                      const file = e.target.files?.[0];
                      if (!file) return;
                      try {
                        const count = await importSyncFile(file);
                        toast.success(`Imported ${count} actions from sync file!`);
                      } catch {
                        toast.error('Failed to import sync file.');
                      }
                      e.target.value = ''; // Reset
                    }}
                  />
                  <button className="btn-secondary w-full flex items-center justify-center gap-2">
                    <Upload className="w-4 h-4" /> Import Progress
                  </button>
                </div>
              </div>
            </div>
          </motion.section>
        </div>
      </div>
    </motion.div>
  );
}
