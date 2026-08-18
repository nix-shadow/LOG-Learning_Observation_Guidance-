"use client";

import { useEffect, useState, useRef } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { BookOpen, WifiOff, GraduationCap } from 'lucide-react';
import toast from 'react-hot-toast';
import { fetchWithCache } from '@/lib/api';
import { SchoolClass } from '@/lib/types';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import RosterOverview, { RosterData } from '@/components/moderator/RosterOverview';
import AssignmentManager from '@/components/moderator/AssignmentManager';
import AnnouncementComposer from '@/components/admin/AnnouncementComposer';

export default function ModeratorDashboard() {
  const { user, isModerator } = useAuth();
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  const [rosterData, setRosterData] = useState<RosterData | null>(null);
  const [classes, setClasses] = useState<SchoolClass[]>([]);
  const [selectedClass, setSelectedClass] = useState('');

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

      <RosterOverview token={token} onLoaded={setRosterData} />

      {/* My Classes */}
      <div className="gsap-stagger card-glow p-6 border border-white/10">
        <h2 className="text-2xl font-bold text-white mb-6 tracking-tight flex items-center">
          <GraduationCap className="w-6 h-6 mr-3 text-brand-neon" /> My Classes
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {classes.length === 0 ? (
            <p className="text-white/40">No classes assigned yet. Ask the admin to create one for you.</p>
          ) : classes.map(c => (
            <button key={c.id} onClick={() => setSelectedClass(c.id)}
              className={`text-left p-5 rounded-2xl border transition-all ${selectedClass === c.id ? 'border-brand-neon bg-brand-neon/10 shadow-glow' : 'border-white/10 bg-white/5 hover:border-white/30'}`}>
              <p className="font-bold text-white text-lg">{c.name}</p>
              <p className="text-sm text-white/50 mt-1">{c.member_count ?? 0} students</p>
            </button>
          ))}
        </div>
      </div>

      {/* Assignments */}
      <div className="gsap-stagger card-glow p-6 border border-white/10">
        <AssignmentManager token={token} classId={selectedClass} className={selectedClassName} />
      </div>

      {/* Announcement */}
      <div className="gsap-stagger card-glow p-6 border border-white/10">
        <AnnouncementComposer token={token} endpoint="/moderator/announcements" />
      </div>
    </div>
  );
}