"use client";
import { AdminData, Learner as LearnerType } from "@/lib/types";

import { useEffect, useState, useRef } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { fetchWithCache } from '@/lib/api';
import { ShieldAlert, Users, Activity, BarChart2, Megaphone, Download, GraduationCap } from 'lucide-react';
import toast from 'react-hot-toast';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';
import ClassManager from '@/components/admin/ClassManager';
import AnnouncementComposer from '@/components/admin/AnnouncementComposer';
import AuditLogTable from '@/components/admin/AuditLogTable';

export default function AdminDashboard() {
  const { user, isAdmin } = useAuth();
  const router = useRouter();
  const [data, setData] = useState<AdminData | null>(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // If not authenticated or not admin, redirect
    if (!user || !isAdmin) {
      toast.error('Unauthorized access. Admin only.');
      router.push('/dashboard');
      return;
    }

    const token = localStorage.getItem('log_token');
    const authHeaders = { headers: { 'Authorization': `Bearer ${token}` } };

    // Fetch admin specific data
    fetchWithCache('/admin/dashboard', authHeaders)
      .then(setData)
      .catch(err => {
        console.error(err);
        toast.error('Failed to load admin data');
      })
      .finally(() => setLoading(false));
  }, [user, isAdmin, router]);

  useGSAP(() => {
    if (prefersReducedMotion()) return;
    if (!loading && data) {
      gsap.fromTo(
        gsap.utils.toArray('.gsap-stagger'),
        { y: 50, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.8, stagger: 0.1, ease: 'power3.out' }
      );
    }
  }, { dependencies: [loading, data], scope: containerRef });

  const handleExport = async () => {
    setExporting(true);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/admin/export/students.csv`, {
        headers: { 'Authorization': `Bearer ${localStorage.getItem('log_token')}` },
      });
      if (!res.ok) throw new Error('Export failed');
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'students_export.csv';
      document.body.appendChild(a); // F16: keep the anchor attached — immediate
      a.click();                      // revoke can abort the download in some browsers
      document.body.removeChild(a);
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      toast.success('Student export downloaded');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setExporting(false);
    }
  };

  if (!isAdmin) return null;
  if (loading) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="w-16 h-16 border-4 border-white/10 border-t-brand-neon rounded-full animate-spin"></div>
    </div>
  );

  const token = typeof window !== 'undefined' ? (localStorage.getItem('log_token') || '') : '';

  return (
    <div ref={containerRef} className="max-w-6xl mx-auto w-full space-y-8">
      <div className="gsap-stagger flex items-center mb-8">
        <ShieldAlert className="w-10 h-10 text-red-500 mr-4 animate-pulse-glow" />
        <div>
          <h1 className="text-4xl font-bold text-white tracking-tight mb-2">Admin Control Center</h1>
          <p className="text-white/60 text-lg">Platform analytics and administrative actions.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
           <div className="absolute top-0 right-0 p-6 opacity-5 text-white group-hover:scale-125 transition-transform duration-500"><Users className="w-20 h-20"/></div>
           <p className="text-[11px] font-bold text-white/50 uppercase tracking-[0.2em] mb-3 relative z-10 flex items-center justify-center"><Users className="w-4 h-4 mr-2"/> Total Users</p>
           <p className="text-5xl font-extrabold text-white relative z-10 tracking-tight">{data?.analytics?.total_users}</p>
        </div>

        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
           <div className="absolute top-0 right-0 p-6 opacity-10 text-brand-neon group-hover:scale-125 transition-transform duration-500"><Activity className="w-20 h-20"/></div>
           <p className="text-[11px] font-bold text-brand-neon uppercase tracking-[0.2em] mb-3 relative z-10 flex items-center justify-center"><Activity className="w-4 h-4 mr-2"/> Active Daily</p>
           <p className="text-5xl font-extrabold text-white relative z-10 tracking-tight">{data?.analytics?.active_daily}</p>
        </div>

        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-2xl border border-white/10 text-center relative overflow-hidden group hover:-translate-y-1 transition-transform">
           <div className="absolute top-0 right-0 p-6 opacity-5 text-brand-amber group-hover:scale-125 transition-transform duration-500"><BarChart2 className="w-20 h-20"/></div>
           <p className="text-[11px] font-bold text-white/50 uppercase tracking-[0.2em] mb-3 relative z-10 flex items-center justify-center"><BarChart2 className="w-4 h-4 mr-2"/> Total Completions</p>
           <p className="text-5xl font-extrabold text-brand-amber relative z-10 tracking-tight">{data?.analytics?.total_completions}</p>
        </div>
      </div>

      {/* Classes Management */}
      <div id="class-section" className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-8 space-y-8">
        <ClassManager token={token} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Announcements */}
        <div id="announcement-section" className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-8">
          <AnnouncementComposer token={token} endpoint="/admin/announcements" />
        </div>

        {/* Audit log + export */}
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-8">
          <AuditLogTable token={token} onExport={handleExport} exporting={exporting} />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mt-12">
        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-8 space-y-6 relative overflow-hidden">
          <div className="absolute inset-0 bg-brand-neon/5 opacity-0 hover:opacity-100 transition-opacity duration-500 pointer-events-none" />
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold text-white tracking-tight">Recent Users</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-white/10 text-brand-muted uppercase tracking-wider text-[10px] font-bold">
                  <th className="pb-3">Name</th>
                  <th className="pb-3">Role</th>
                  <th className="pb-3">Email / Phone</th>
                </tr>
              </thead>
              <tbody>
                {data?.recent_users?.map((u: LearnerType & { role?: string; phone?: string }) => (
                  <tr key={u.id} className="border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors">
                    <td className="py-4 font-medium text-white">{u.name || 'Unknown'}</td>
                    <td className="py-4">
                      <span className={`px-2 py-1 rounded-full text-[10px] font-bold uppercase tracking-widest ${u.role === 'ADMIN' ? 'bg-red-500/20 text-red-400 border border-red-500/30' : 'bg-brand-blue/20 text-brand-blue border border-brand-blue/30'}`}>
                        {u.role}
                      </span>
                    </td>
                    <td className="py-4 text-white/60">{u.email || u.phone}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="gsap-stagger card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-8 space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold text-white tracking-tight">Quick Actions</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <button onClick={() => document.querySelector('#class-section')?.scrollIntoView({ behavior: 'smooth' })}
              className="p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-brand-neon hover:bg-brand-neon/10 hover:shadow-glow transition-all flex flex-col items-center justify-center text-white/60 hover:text-brand-neon group">
              <GraduationCap className="w-8 h-8 mb-3 opacity-70 group-hover:opacity-100 group-hover:scale-110 transition-all" />
              <span className="font-bold tracking-tight">Manage Classes</span>
            </button>
            <button onClick={() => document.querySelector('#announcement-section')?.scrollIntoView({ behavior: 'smooth' })}
              className="p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-brand-amber hover:bg-brand-amber/10 hover:shadow-[0_0_20px_rgba(255,183,3,0.3)] transition-all flex flex-col items-center justify-center text-white/60 hover:text-brand-amber group">
              <Megaphone className="w-8 h-8 mb-3 opacity-70 group-hover:opacity-100 group-hover:scale-110 transition-all" />
              <span className="font-bold tracking-tight">Send Broadcast</span>
            </button>
            <button onClick={handleExport} disabled={exporting}
              className="p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-brand-blue hover:bg-brand-blue/10 hover:shadow-[0_0_20px_rgba(0,180,216,0.3)] transition-all flex flex-col items-center justify-center text-white/60 hover:text-brand-blue group disabled:opacity-50">
              <Download className="w-8 h-8 mb-3 opacity-70 group-hover:opacity-100 group-hover:scale-110 transition-all" />
              <span className="font-bold tracking-tight">{exporting ? 'Exporting…' : 'Export Students'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}