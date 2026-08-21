"use client";

import { useEffect, useState, useRef } from 'react';
import { fetchWithCache, logout, flushSyncQueue } from '@/lib/api';
import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { CheckCircle2, ArrowRight, Activity, TrendingUp, Medal, Flame, Download, Upload, LogOut, ClipboardList, Megaphone, Send } from 'lucide-react';
import { DashboardData, Activity as ActivityType, Guidance as GuidanceType, Observation as ObservationType, Announcement as AnnouncementType, Assignment as AssignmentType } from "@/lib/types";
import { CircularProgressbar, buildStyles } from 'react-circular-progressbar';
import 'react-circular-progressbar/dist/styles.css';
import SkeletonLoader from '@/components/SkeletonLoader';
import ReviewQueueCard from '@/components/ReviewQueueCard';
import JoinClassCard from '@/components/JoinClassCard';
import ReconnectDigest from '@/components/ReconnectDigest';
import { downloadSyncFile, importSyncFile } from '@/lib/syncExport';
import toast from 'react-hot-toast';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';

export default function Dashboard() {
  const t = useTranslations('status');
  const [data, setData] = useState<DashboardData | null>(null);
  const [asOf, setAsOf] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  const [announcements, setAnnouncements] = useState<AnnouncementType[]>([]);
  const [assignments, setAssignments] = useState<AssignmentType[]>([]);
  const [noteDrafts, setNoteDrafts] = useState<Record<string, string>>({});

  useEffect(() => {
    fetchWithCache('/dashboard')
      .then((res) => {
        setData(res);
        if (typeof res?.as_of === 'string') setAsOf(res.as_of);
      })
      .catch(console.error)
      .finally(() => setLoading(false));

    fetchWithCache('/announcements').then((d) => setAnnouncements(d.announcements || [])).catch(() => {});
    fetchWithCache('/assignments').then((d) => setAssignments(d.assignments || [])).catch(() => {});
  }, []);

  const handleSubmit = async (assignmentId: string) => {
    const note = noteDrafts[assignmentId] || '';
    if (!note.trim()) {
      toast.error('Write a short note about your submission');
      return;
    }
    try {
      const res = await fetchWithCache(`/assignments/${assignmentId}/submit`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('log_token')}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ note }),
      });
      // F8: queued submissions are not accepted yet — be honest about it.
      if (res && res.queued) {
        toast.success('Submission saved offline. Will sync when back online.', { icon: '💾' });
      } else {
        toast.success('Assignment submitted!');
      }
      setNoteDrafts(prev => ({ ...prev, [assignmentId]: '' }));
      fetchWithCache('/assignments').then((d) => setAssignments(d.assignments || [])).catch(() => {});
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to submit assignment');
    }
  };

  // M8: zero-value due dates ("0001-01-01T00:00:00Z") must not render as 1/1/1.
  const formatDueDate = (due?: string | null) => {
    if (!due) return 'No deadline';
    const d = new Date(due);
    if (isNaN(d.getTime()) || d.getFullYear() < 2000) return 'No deadline';
    return `Due ${d.toLocaleDateString()}`;
  };

  useGSAP(() => {
    if (prefersReducedMotion()) return;
    if (!loading && data) {
      gsap.fromTo(
        gsap.utils.toArray('.gsap-stagger'),
        { y: 50, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.8, stagger: 0.1, ease: 'power3.out' }
      );
    }
  }, { scope: containerRef, dependencies: [loading, data] });

  if (loading) return (
    <div className="space-y-8">
      <SkeletonLoader type="card" count={1} />
      <SkeletonLoader type="stats" count={3} />
      <SkeletonLoader type="card" count={2} />
    </div>
  );

  if (!data || !data.learner) return (
    <div className="text-center py-20 card-glow flex flex-col items-center justify-center max-w-2xl mx-auto mt-10">
      <div className="w-20 h-20 bg-white/5 rounded-full flex items-center justify-center mb-6">
        <Activity className="w-10 h-10 text-brand-neon" />
      </div>
      <h2 className="text-3xl font-bold text-white mb-3">Welcome to LOG</h2>
      <p className="text-white/60 mb-8 text-lg">Your learning journey starts here. (Offline mode active, no cached data found).</p>
      <Link href="/learning" className="btn-primary">Start Learning</Link>
    </div>
  );

  const dailyGoalPercentage = data.progress.total_topics > 0
    ? Math.min(100, Math.round((data.progress.completed / data.progress.total_topics) * 100))
    : 0;

  return (
    <div ref={containerRef} className="space-y-8">
      {/* Welcome Area (Hero Bento) */}
      <section className="gsap-stagger bg-black/40 backdrop-blur-3xl border border-white/10 rounded-[32px] p-10 relative overflow-hidden flex flex-col md:flex-row items-center justify-between gap-8">
        <div className="relative z-10 w-full md:w-2/3">
          <h1 className="text-4xl font-bold mb-3 text-white tracking-tight">Hello, {data.learner.name}</h1>
          <p className="text-white/70 text-lg mb-6">You are on a <span className="text-brand-neon font-bold">{data.progress.current_streak}-day</span> learning streak. Keep it up!</p>
          <div className="flex flex-wrap gap-4 items-center">
             <div className="bg-white/5 px-5 py-2.5 rounded-full flex items-center gap-2 backdrop-blur-xl border border-white/10 shadow-sm">
                <Flame className="w-5 h-5 text-brand-amber" />
                <span className="font-semibold text-white">{data.progress.current_streak} Day Streak</span>
             </div>
             <div className="bg-white/5 px-5 py-2.5 rounded-full flex items-center gap-2 backdrop-blur-xl border border-white/10 shadow-sm">
                <Medal className="w-5 h-5 text-brand-neon" />
                <span className="font-semibold text-white">Logic Master Badge</span>
             </div>
             <button
               onClick={() => logout()}
               className="bg-white/5 hover:bg-red-500/20 hover:border-red-500/50 hover:text-red-400 transition-all px-5 py-2.5 rounded-full flex items-center gap-2 backdrop-blur-xl border border-white/10 text-white/70 text-sm font-semibold"
               title="Logout"
             >
               <LogOut className="w-4 h-4" />
               <span>Logout</span>
             </button>
          </div>
          {asOf && (() => {
            const d = new Date(asOf);
            if (Number.isNaN(d.getTime())) return null;
            return (
              <p className="text-[11px] text-white/40 mt-3">
                Data updated {d.toLocaleDateString()} at {d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </p>
            );
          })()}
        </div>

        <div className="relative z-10 w-40 h-40 flex-shrink-0 bg-[#0B1220] backdrop-blur-xl rounded-full p-3 border border-brand-blue/30">
            <CircularProgressbar
              value={dailyGoalPercentage}
              text={`${dailyGoalPercentage}%`}
              styles={buildStyles({
                pathColor: '#60A5FA',
                textColor: '#E9F0FA',
                trailColor: 'rgba(255,255,255,0.08)',
                textSize: '20px',
                pathTransitionDuration: 2
              })}
            />
            <div className="text-center mt-4 text-[10px] font-bold uppercase text-brand-neon absolute -bottom-8 left-0 right-0 tracking-[0.2em]">Daily Goal</div>
        </div>

        {/* Ambient background glows */}
        <div className="absolute top-0 right-0 -mt-20 -mr-20 w-96 h-96 bg-brand-neon/20 rounded-full blur-[100px] pointer-events-none"></div>
        <div className="absolute bottom-0 left-0 -mb-20 -ml-20 w-96 h-96 bg-purple-500/20 rounded-full blur-[100px] pointer-events-none"></div>
      </section>

      {/* Bento Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Content Area (Left 2 columns) */}
        <div className="lg:col-span-2 space-y-8 flex flex-col">

          {/* Current Focus (Guidance) */}
          <section className="gsap-stagger">
            <h2 className="text-2xl font-bold text-white mb-6 flex items-center tracking-tight">
              <TrendingUp className="w-6 h-6 mr-3 text-brand-neon" /> Current Focus
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
              {data.guidance.map((g: GuidanceType) => (
                <div
                  key={g.id}
                  className="card-glow border-l-4 border-l-brand-amber flex flex-col h-full"
                >
                  <p className="text-white/90 font-medium mb-4 flex-1 text-lg">{g.text}</p>
                  {g.action && (
                    <Link href={g.action} className="text-brand-neon text-sm font-bold flex items-center hover:text-white transition-colors group mt-auto">
                      Take action <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-2 transition-transform" />
                    </Link>
                  )}
                </div>
              ))}
            </div>
          </section>

          {/* Learning Journey Overview */}
          <section className="gsap-stagger flex-1 flex flex-col">
            <div className="flex justify-between items-end mb-6">
              <h2 className="text-2xl font-bold text-white flex items-center tracking-tight">
                <Activity className="w-6 h-6 mr-3 text-brand-neon" /> Learning Journey
              </h2>
              <Link href="/learning" className="text-sm font-bold text-brand-neon hover:text-white transition-colors">View all</Link>
            </div>
            <div className="card-glow flex-1 p-6">
              <div className="space-y-5">
                {data.activities.slice(0, 3).map((act: ActivityType) => {
                  // WP-1.1 RC-01: canonical statuses, supportive phrasing.
                  const statusLabels: Record<string, string> = {
                    'not-started': t('notStarted'),
                    'active': t('active'),
                    'needs-practice': t('needsPractice'),
                    'completed': t('completed'),
                  };
                  const statusColor: Record<string, string> = {
                    'not-started': 'text-brand-faint',
                    'active': 'text-brand-amber',
                    'needs-practice': 'text-brand-amber',
                    'completed': 'text-brand-neon',
                  };
                  return (
                  <div key={act.id} className="flex items-start pb-5 border-b border-white/10 last:border-0 last:pb-0 group">
                    <div className={`mt-1 flex-shrink-0 transition-transform group-hover:scale-110 ${statusColor[act.status] ?? 'text-brand-faint'}`}>
                      <CheckCircle2 className="w-6 h-6" />
                    </div>
                    <div className="ml-4">
                      <p className="text-base font-bold text-white group-hover:text-brand-neon transition-colors">{act.title}</p>
                      <p className={`text-sm font-medium mt-1 ${act.status === 'completed' ? 'text-brand-neon' : act.status === 'needs-practice' ? 'text-brand-amber' : 'text-white/50'}`}>
                        {statusLabels[act.status] ?? act.status}
                      </p>
                    </div>
                  </div>
                  );
                })}
              </div>
            </div>
          </section>

          {/* Assignments */}
          <section className="gsap-stagger">
            <h2 className="text-2xl font-bold text-white mb-6 flex items-center tracking-tight">
              <ClipboardList className="w-6 h-6 mr-3 text-brand-amber" /> My Assignments
            </h2>
            <div className="card-glow p-6 space-y-5">
              {assignments.length === 0 ? (
                <p className="text-white/50">No assignments for your classes yet. Check back soon!</p>
              ) : assignments.map((a: AssignmentType) => (
                <div key={a.id} className="border border-white/10 rounded-2xl p-5 bg-white/5">
                  <div className="flex flex-wrap items-center justify-between gap-3 mb-2">
                    <p className="text-lg font-bold text-white">{a.title}</p>
                    <span className="text-xs text-white/50">
                      {a.due_date ? formatDueDate(a.due_date) : 'No deadline'}
                    </span>
                  </div>
                  {a.description && <p className="text-white/60 text-sm mb-4 whitespace-pre-wrap">{a.description}</p>}
                  <div className="flex gap-3">
                    <input
                      value={noteDrafts[a.id] || ''}
                      onChange={e => setNoteDrafts(prev => ({ ...prev, [a.id]: e.target.value }))}
                      className="flex-1 px-4 py-2.5 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-amber/50 outline-none placeholder-white/30 text-sm"
                      placeholder="Write your answer / note…"
                    />
                    <button onClick={() => handleSubmit(a.id)}
                      className="btn-primary flex items-center gap-2 text-sm disabled:opacity-40">
                      <Send className="w-4 h-4" /> Submit
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </section>

          {/* Announcements */}
          <section className="gsap-stagger">
            <h2 className="text-2xl font-bold text-white mb-6 flex items-center tracking-tight">
              <Megaphone className="w-6 h-6 mr-3 text-brand-blue" /> Announcements
            </h2>
            <div className="card-glow p-6 space-y-5">
              {announcements.length === 0 ? (
                <p className="text-white/50">No announcements yet.</p>
              ) : announcements.map((ann: AnnouncementType) => (
                <div key={ann.id} className="pb-4 border-b border-white/10 last:border-0 last:pb-0">
                  <div className="flex items-center justify-between mb-1">
                    <p className="font-bold text-white">{ann.title}</p>
                    <span className="text-xs text-brand-muted">{new Date(ann.created_at).toLocaleDateString()}</span>
                  </div>
                  <p className="text-white/70 text-sm leading-relaxed">{ann.body}</p>
                </div>
              ))}
            </div>
          </section>

        </div>

        {/* Sidebar (Right column) - Review queue, Observations & Sync */}
        <div className="space-y-8 flex flex-col">
          <section className="gsap-stagger">
            <ReconnectDigest />
          </section>
          <section className="gsap-stagger">
            <ReviewQueueCard />
          </section>
          <section className="gsap-stagger">
            <JoinClassCard />
          </section>
          <section className="gsap-stagger">
            <h2 className="text-2xl font-bold text-white mb-6 tracking-tight">Recent Observations</h2>
            <div className="card-glow space-y-5 p-6">
              {data.observations.map((obs: ObservationType) => (
                <div key={obs.id} className="pb-4 border-b border-white/10 last:border-0 last:pb-0">
                  <span className="text-[11px] font-bold uppercase text-brand-neon tracking-widest mb-2 block">{obs.category}</span>
                  <p className="text-base text-white/80 leading-relaxed">{obs.text}</p>
                </div>
              ))}
              <div className="pt-2">
                <Link href="/observation" className="text-sm font-bold text-brand-neon hover:text-white transition-colors">View full observation</Link>
              </div>
            </div>
          </section>

          {/* Sync Offline Progress */}
          <section className="gsap-stagger flex-1">
            <h2 className="text-2xl font-bold text-white mb-6 tracking-tight">Offline Sync</h2>
            <div className="card-glow space-y-6 h-[calc(100%-3rem)] flex flex-col justify-between border border-white/10">
              <p className="text-base text-white/70 leading-relaxed">
                Working offline? Download your progress and bring it to school on a USB drive.
              </p>
              <div className="space-y-4 mt-auto">
                <button 
                  onClick={async () => {
                    try {
                      await downloadSyncFile();
                      toast.success('Sync file downloaded successfully!');
                    } catch (e: unknown) {
                      toast.error((e as Error).message || 'Failed to download sync file.');
                    }
                  }}
                  className="btn-primary w-full flex items-center justify-center gap-3 py-3"
                >
                  <Download className="w-5 h-5" /> Export Progress
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
                        // F1: a sneaker-net import should try to ship the queue
                        // immediately — not wait for the next online event.
                        const { synced } = await flushSyncQueue();
                        if (synced > 0) {
                          toast.success(`Imported ${count} actions — ${synced} synced to the server!`);
                        } else {
                          toast.success(`Imported ${count} actions from sync file!`);
                        }
                      } catch {
                        toast.error('Failed to import sync file.');
                      }
                      e.target.value = '';
                    }}
                  />
                  <button className="btn-secondary w-full flex items-center justify-center gap-3 py-3 bg-white/5 border-white/10 hover:bg-white/10">
                    <Upload className="w-5 h-5" /> Import Progress
                  </button>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
