"use client";

import { useEffect, useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { Users, BookOpen, AlertCircle, WifiOff } from 'lucide-react';
import toast from 'react-hot-toast';
import { motion } from 'framer-motion';
import { fetchWithCache } from '@/lib/api';

export default function ModeratorDashboard() {
  const { user, isModerator } = useAuth();
  const router = useRouter();
  const [loading, setLoading] = useState(true);

  const [rosterData, setRosterData] = useState<{
    class_name: string;
    active_students: number;
    needs_attention: number;
    assignments_due: number;
    roster: Array<{ id: string; name: string; completion: number; streak: number; status: string; last_active: string }>;
  } | null>(null);

  useEffect(() => {
    if (!user || !isModerator) {
      toast.error('Unauthorized access. Teachers/Moderators only.');
      router.push('/dashboard');
      return;
    }

    const token = localStorage.getItem('log_token');
    fetchWithCache('/moderator/roster', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(setRosterData)
      .catch((err: unknown) => {
        console.warn('Using cached moderator data or default roster', err);
      })
      .finally(() => setLoading(false));

  }, [user, isModerator, router]);

  if (!isModerator) return null;
  if (loading && !rosterData) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-teal"></div>
    </div>
  );

  return (
    <div className="max-w-6xl mx-auto w-full space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8 border-b border-gray-200 pb-6">
        <div>
           <h1 className="text-3xl font-bold text-brand-blue flex items-center">
             <BookOpen className="w-8 h-8 text-brand-teal mr-3" /> Teacher Portal
           </h1>
           <p className="text-gray-500 mt-2">Manage your classes and review student progress.</p>
        </div>
        <div className="flex gap-3">
           <button 
             onClick={() => {
               toast.promise(
                 fetchWithCache('/moderator/roster', { headers: { 'Authorization': `Bearer ${localStorage.getItem('log_token')}` } }),
                 {
                   loading: 'Caching class roster...',
                   success: 'Roster cached! You can now view it offline.',
                   error: 'Failed to cache roster.'
                 }
               ).then(setRosterData);
             }}
             className="btn-secondary flex items-center gap-2"
           >
             <WifiOff className="w-4 h-4" /> Pre-fetch for Offline
           </button>
           <button className="btn-primary">Create Assignment</button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="card border-t-4 border-t-brand-teal">
           <h3 className="text-gray-500 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><Users className="w-4 h-4 mr-2"/> Active Students</h3>
           <p className="text-4xl font-bold text-brand-blue">{rosterData?.active_students ?? 0}</p>
           <p className="text-sm text-gray-500 mt-2">Students in your class</p>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{delay: 0.1}} className="card border-t-4 border-t-brand-amber">
           <h3 className="text-gray-500 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><AlertCircle className="w-4 h-4 mr-2"/> Needs Attention</h3>
           <p className="text-4xl font-bold text-brand-blue">{rosterData?.needs_attention ?? 0}</p>
           <p className="text-sm text-brand-amber mt-2">Students falling behind</p>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{delay: 0.2}} className="card border-t-4 border-t-brand-blue">
           <h3 className="text-gray-500 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><BookOpen className="w-4 h-4 mr-2"/> Assignments Due</h3>
           <p className="text-4xl font-bold text-brand-blue">{rosterData?.assignments_due ?? 0}</p>
           <p className="text-sm text-gray-500 mt-2">Activities awaiting completion</p>
        </motion.div>
      </div>

      <div className="card mt-12">
        <h2 className="text-xl font-bold text-brand-blue mb-6">Class Roster: {rosterData?.class_name || 'Logic 101'}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-gray-500 bg-gray-50">
                <th className="pb-3 pt-3 px-4 font-medium rounded-tl-lg">Student Name</th>
                <th className="pb-3 pt-3 px-4 font-medium">Completion %</th>
                <th className="pb-3 pt-3 px-4 font-medium">Current Streak</th>
                <th className="pb-3 pt-3 px-4 font-medium rounded-tr-lg">Action</th>
              </tr>
            </thead>
            <tbody>
              {(rosterData?.roster || [
                { id: '1', name: 'Aisha Student', completion: 85, streak: 4 },
                { id: '2', name: 'Bikash Thapa', completion: 75, streak: 3 },
                { id: '3', name: 'Chandan Gurung', completion: 60, streak: 2 },
                { id: '4', name: 'Dawa Sherpa', completion: 90, streak: 5 },
              ]).map((st) => (
                <tr key={st.id} className="border-b border-gray-100 last:border-0 hover:bg-gray-50">
                  <td className="py-4 px-4 font-medium text-brand-blue flex items-center">
                    <div className="w-8 h-8 rounded-full bg-brand-teal/20 text-brand-teal flex items-center justify-center mr-3 font-bold">
                      {st.name.charAt(0)}
                    </div>
                    {st.name}
                  </td>
                  <td className="py-4 px-4">
                    <div className="flex items-center">
                      <div className="w-full bg-gray-200 h-2 rounded-full mr-3 max-w-[100px]">
                        <div className="bg-brand-teal h-2 rounded-full" style={{ width: `${st.completion}%` }}></div>
                      </div>
                      <span className="text-gray-600">{st.completion}%</span>
                    </div>
                  </td>
                  <td className="py-4 px-4 text-gray-600 font-medium">{st.streak} days</td>
                  <td className="py-4 px-4">
                    <button className="text-sm text-brand-teal font-medium hover:underline">Message</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

