"use client";
import { AdminData, Learner as LearnerType } from "@/lib/types";

import { useEffect, useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { fetchWithCache } from '@/lib/api';
import { ShieldAlert, Users, Activity, BarChart2, PlusCircle } from 'lucide-react';
import toast from 'react-hot-toast';
import { motion } from 'framer-motion';

export default function AdminDashboard() {
  const { user, isAdmin } = useAuth();
  const router = useRouter();
  const [data, setData] = useState<AdminData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // If not authenticated or not admin, redirect
    if (!user || !isAdmin) {
      toast.error('Unauthorized access. Admin only.');
      router.push('/dashboard');
      return;
    }

    const token = localStorage.getItem('log_token');

    // Fetch admin specific data
    fetchWithCache('/admin/dashboard', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(setData)
      .catch(err => {
        console.error(err);
        toast.error('Failed to load admin data');
      })
      .finally(() => setLoading(false));
  }, [user, isAdmin, router]);

  if (!isAdmin) return null;
  if (loading) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-teal"></div>
    </div>
  );

  return (
    <div className="max-w-6xl mx-auto w-full space-y-8">
      <div className="flex items-center mb-8">
        <ShieldAlert className="w-8 h-8 text-red-500 mr-3" />
        <h1 className="text-3xl font-bold text-brand-blue">Admin Control Center</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="card bg-brand-blue text-white relative overflow-hidden">
           <div className="relative z-10">
             <h3 className="text-brand-gray/80 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><Users className="w-4 h-4 mr-2"/> Total Users</h3>
             <p className="text-4xl font-bold">{data?.analytics?.total_users}</p>
           </div>
           <div className="absolute -bottom-4 -right-4 opacity-10"><Users className="w-32 h-32"/></div>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{delay: 0.1}} className="card bg-brand-teal text-white relative overflow-hidden">
           <div className="relative z-10">
             <h3 className="text-brand-white/80 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><Activity className="w-4 h-4 mr-2"/> Active Daily</h3>
             <p className="text-4xl font-bold">{data?.analytics?.active_daily}</p>
           </div>
           <div className="absolute -bottom-4 -right-4 opacity-10"><Activity className="w-32 h-32"/></div>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{delay: 0.2}} className="card bg-brand-amber text-white relative overflow-hidden">
           <div className="relative z-10">
             <h3 className="text-brand-white/80 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><BarChart2 className="w-4 h-4 mr-2"/> Total Completions</h3>
             <p className="text-4xl font-bold">{data?.analytics?.total_completions}</p>
           </div>
           <div className="absolute -bottom-4 -right-4 opacity-10"><BarChart2 className="w-32 h-32"/></div>
        </motion.div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mt-12">
        <div className="card space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold text-brand-blue">Recent Users</h2>
            <button className="text-sm text-brand-teal font-medium hover:underline">View All</button>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-gray-500">
                  <th className="pb-3 font-medium">Name</th>
                  <th className="pb-3 font-medium">Role</th>
                  <th className="pb-3 font-medium">Email / Phone</th>
                </tr>
              </thead>
              <tbody>
                {data?.recent_users?.map((u: LearnerType & { role?: string; phone?: string }) => (
                  <tr key={u.id} className="border-b border-gray-100 last:border-0 hover:bg-gray-50">
                    <td className="py-4 font-medium text-brand-blue">{u.name || 'Unknown'}</td>
                    <td className="py-4">
                      <span className={`px-2 py-1 rounded-full text-xs font-bold uppercase tracking-wide ${u.role === 'ADMIN' ? 'bg-red-100 text-red-700' : 'bg-blue-100 text-blue-700'}`}>
                        {u.role}
                      </span>
                    </td>
                    <td className="py-4 text-gray-500">{u.email || u.phone}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="card space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold text-brand-blue">Quick Actions</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <button className="p-6 border-2 border-dashed border-gray-300 rounded-2xl hover:border-brand-teal hover:bg-brand-teal/5 transition-all flex flex-col items-center justify-center text-gray-500 hover:text-brand-teal">
              <PlusCircle className="w-8 h-8 mb-2" />
              <span className="font-medium">Create Activity</span>
            </button>
            <button className="p-6 border-2 border-dashed border-gray-300 rounded-2xl hover:border-brand-amber hover:bg-brand-amber/5 transition-all flex flex-col items-center justify-center text-gray-500 hover:text-brand-amber">
              <PlusCircle className="w-8 h-8 mb-2" />
              <span className="font-medium">Send Broadcast</span>
            </button>
            <button className="p-6 border-2 border-dashed border-gray-300 rounded-2xl hover:border-brand-blue hover:bg-brand-blue/5 transition-all flex flex-col items-center justify-center text-gray-500 hover:text-brand-blue">
              <Users className="w-8 h-8 mb-2" />
              <span className="font-medium">Manage Roles</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
