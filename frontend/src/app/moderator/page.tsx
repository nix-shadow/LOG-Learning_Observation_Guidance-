"use client";

import { useEffect, useState, useRef } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { BookOpen, WifiOff, GraduationCap, Sparkles } from 'lucide-react';
import toast from 'react-hot-toast';
import { fetchWithCache } from '@/lib/api';
import { SchoolClass } from '@/lib/types';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';
import RosterOverview, { RosterData } from '@/components/moderator/RosterOverview';
import AssignmentManager from '@/components/moderator/AssignmentManager';
import AnnouncementComposer from '@/components/admin/AnnouncementComposer';
import ClassWizard from '@/components/moderator/ClassWizard';
import SupportInbox from '@/components/SupportInbox';
import GradebookOverview from '@/components/moderator/GradebookOverview';
import { useTranslations } from 'next-intl';

export default function ModeratorDashboard() {
  const { user, isModerator } = useAuth();
  const router = useRouter();
  const t = useTranslations('onboard');
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  const [rosterData, setRosterData] = useState<RosterData | null>(null);
  const [classes, setClasses] = useState<SchoolClass[]>([]);
  const [selectedClass, setSelectedClass] = useState('');
  const [wizardOpen, setWizardOpen] = useState(false);

  const authHeaders = () => ({ headers: { 'Authorization': `Bearer ${localStorage.getItem('log_token')}` } });

  const loadClasses = () => {
    fetchWithCache('/moderator/classes', authHeaders())
      .then((d) => {
        setClasses(d.classes || []);
        if (d.classes?.length && !selectedClass) setSelectedClass(d.classes[0].id);
      })
      .catch(() => {});
  };

  useEffect(() => {
    if (!user || !isModerator) {
      toast.error('Unauthorized access. Teachers/Moderators only.');
      router.push('/dashboard');
      return;
    }
    loadClasses();
    setLoading(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user, isModerator, router]);

  useGSAP(() => {
    if (prefersReducedMotion()) return;
    if (!loading && rosterData) {
      gsap.fromTo(
        gsap.utils.toArray('.gsap-stagger'),
        { y: 50, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.8, stagger: 0.1, ease: 'power3.out' }
      );
    }
  }, { scope: containerRef, dependencies: [loading, rosterData] });

  if (!isModerator) return null;

  const token = typeof window !== 'undefined' ? (localStorage.getItem('log_token') || '') : '';
  const selectedClassName = classes.find(c => c.id === selectedClass)?.name || '';

  return (
    <div ref={containerRef} className="max-w-6xl mx-auto w-full space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8 border-b border-white/10 pb-6 gsap-stagger">
        <div>
           <h1 className="text-3xl font-bold text-white flex items-center tracking-tight">
             <BookOpen className="w-8 h-8 text-brand-neon mr-3" /> Teacher Portal
           </h1>
           <p className="text-white/60 mt-2 text-lg">Manage your classes and review student progress.</p>
        </div>
        <div className="flex gap-3">
           <button
             onClick={() => setWizardOpen(true)}
             className="bg-brand-neon/15 hover:bg-brand-neon/25 text-brand-neon transition-all px-5 py-2.5 rounded-full flex items-center gap-2 backdrop-blur-xl border border-brand-neon/30 font-semibold text-sm"
           >
             <GraduationCap className="w-4 h-4" /> {t('newClass')}
           </button>
           <button
             onClick={() => {
               toast.promise(
                 fetchWithCache('/moderator/roster', authHeaders()),
                 {
                   loading: 'Caching class roster...',
                   success: 'Roster cached! You can now view it offline.',
                   error: 'Failed to cache roster.'
                 }
               ).then(setRosterData);
             }}
             className="bg-white/5 hover:bg-white/10 text-white transition-all px-5 py-2.5 rounded-full flex items-center gap-2 backdrop-blur-xl border border-white/10 font-semibold text-sm"
           >
             <WifiOff className="w-4 h-4" /> Pre-fetch for Offline
           </button>
        </div>
      </div>

      {classes.length === 0 && !rosterData && (
        <div className="gsap-stagger card-glow border border-brand-neon/30 bg-gradient-to-br from-brand-neon/10 to-transparent rounded-3xl p-8">
          <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
            <div className="flex items-start gap-4">
              <span className="p-3 rounded-2xl bg-brand-neon/20 text-brand-neon">
                <Sparkles className="w-7 h-7" />
              </span>
              <div>
                <h2 className="text-2xl font-bold text-white tracking-tight mb-2">{t('welcome')}</h2>
                <p className="text-white/60">{t('welcomeHint')}</p>
                <ol className="mt-4 space-y-2 text-sm text-white/70">
                  <li>1. {t('step1')}</li>
                  <li>2. {t('step2')}</li>
                  <li>3. {t('step3')}</li>
                </ol>
              </div>
            </div>
            <button
              onClick={() => setWizardOpen(true)}
              className="btn-primary px-7 py-3.5 font-bold whitespace-nowrap"
            >
              {t('startWizard')} →
            </button>
          </div>
        </div>
      )}

      <RosterOverview token={token} onLoaded={setRosterData} />

      {/* My Classes */}
      <div className="gsap-stagger card-glow p-6 border border-white/10">
        <h2 className="text-2xl font-bold text-white mb-6 tracking-tight flex items-center">
          <GraduationCap className="w-6 h-6 mr-3 text-brand-neon" /> My Classes
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {classes.length === 0 ? (
            <p className="text-brand-muted">{t('noClasses')}</p>
          ) : classes.map(c => (
            <div key={c.id} className="p-5 rounded-2xl border transition-all border-white/10 bg-white/5">
              <button onClick={() => setSelectedClass(c.id)}
                className={`text-left w-full ${selectedClass === c.id ? '' : ''}`}>
                <p className="font-bold text-white text-lg">{c.name}</p>
                <p className="text-sm text-white/50 mt-1">{c.member_count ?? 0} students</p>
              </button>
              {c.invite_code && (
                <p className="mt-3 pt-3 border-t border-white/10 flex items-center justify-between">
                  <span className="text-xs text-white/40 uppercase tracking-wider">Invite code</span>
                  <button
                    onClick={() => navigator.clipboard?.writeText(c.invite_code ?? '').then(() => toast.success('Code copied'))}
                    className="font-mono font-bold tracking-widest text-brand-neon hover:text-white transition-colors"
                  >
                    {c.invite_code}
                  </button>
                </p>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Assignments */}
      <div className="gsap-stagger card-glow p-6 border border-white/10">
        <AssignmentManager token={token} classId={selectedClass} className={selectedClassName} />
      </div>

      {/* WP-2.3: honest gradebook — real accuracy/attempts per activity,
          CSV export, and per-student supportive notes */}
      <div className="gsap-stagger">
        <GradebookOverview token={token} selectedClass={selectedClass} />
      </div>

      {/* Announcement */}
      <div className="gsap-stagger card-glow p-6 border border-white/10">
        <AnnouncementComposer token={token} endpoint="/moderator/announcements" />
      </div>

      {/* WP-2.2: support inbox — escalated learner issues, resolved here */}
      <div className="gsap-stagger">
        <SupportInbox />
      </div>

      {wizardOpen && (
        <ClassWizard token={token} onClose={() => setWizardOpen(false)} onCreated={loadClasses} />
      )}
    </div>
  );
}