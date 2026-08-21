"use client";
import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { AlertCircle, BookOpen, Users, ChevronRight } from 'lucide-react';
import toast from 'react-hot-toast';
import StudentProgressModal from './StudentProgressModal';

export interface RosterStudent {
  id: string;
  name: string;
  completion: number;
  streak: number;
  status: string;
  last_active: string;
}

export interface RosterData {
  class_name: string;
  active_students: number;
  needs_attention: number;
  assignments_due: number;
  roster: RosterStudent[];
}

interface RosterOverviewProps {
  token: string;
  onLoaded?: (data: RosterData) => void;
}

// RosterOverview owns the teacher's class stats and roster table. Every number
// comes from the real /moderator/roster payload — when the network is down the
// cached payload is served; when there is genuinely no data, honest zeros and
// an empty row are rendered (never invented students).
export default function RosterOverview({ token, onLoaded }: RosterOverviewProps) {
  const [data, setData] = useState<RosterData | null>(null);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<RosterStudent | null>(null);

  useEffect(() => {
    fetchWithCache('/moderator/roster', { headers: { 'Authorization': `Bearer ${token}` } })
      .then((d) => {
        setData(d);
        onLoaded?.(d);
      })
      .catch(() => {
        toast.error('Failed to load roster');
        setData(null);
      })
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[30vh]">
        <div className="w-12 h-12 border-4 border-white/10 border-t-brand-neon rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div className="card-glow border-t-4 border-t-brand-neon gsap-stagger flex flex-col p-6">
          <h3 className="text-white/60 font-medium mb-2 uppercase tracking-wider text-xs flex items-center"><Users className="w-4 h-4 mr-2" /> Active Students</h3>
          <p className="text-5xl font-bold text-white mb-2 tracking-tight">{data?.active_students ?? 0}</p>
          <p className="text-sm text-brand-muted mt-auto">Students in your class</p>
        </div>

        <div className="card-glow border-t-4 border-t-brand-amber gsap-stagger flex flex-col p-6">
          <h3 className="text-white/60 font-medium mb-2 uppercase tracking-wider text-xs flex items-center"><AlertCircle className="w-4 h-4 mr-2 text-brand-amber" /> Needs Attention</h3>
          <p className="text-5xl font-bold text-white mb-2 tracking-tight">{data?.needs_attention ?? 0}</p>
          <p className="text-sm text-brand-amber mt-auto">Students falling behind</p>
        </div>

        <div className="card-glow border-t-4 border-t-purple-500 gsap-stagger flex flex-col p-6">
          <h3 className="text-white/60 font-medium mb-2 uppercase tracking-wider text-xs flex items-center"><BookOpen className="w-4 h-4 mr-2 text-purple-400" /> Assignments Due</h3>
          <p className="text-5xl font-bold text-white mb-2 tracking-tight">{data?.assignments_due ?? 0}</p>
          <p className="text-sm text-brand-muted mt-auto">Activities awaiting completion</p>
        </div>
      </div>

      <div className="card-glow p-6 gsap-stagger border border-white/10">
        <h2 className="text-2xl font-bold text-white mb-8 tracking-tight">
          Class Roster: <span className="text-brand-neon">{data?.class_name || 'No class yet'}</span>
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-white/10 text-white/50 bg-white/5">
                <th className="pb-4 pt-4 px-6 font-semibold rounded-tl-lg uppercase tracking-wider text-xs">Student Name</th>
                <th className="pb-4 pt-4 px-6 font-semibold uppercase tracking-wider text-xs">Completion %</th>
                <th className="pb-4 pt-4 px-6 font-semibold rounded-tr-lg uppercase tracking-wider text-xs">Current Streak</th>
              </tr>
            </thead>
            <tbody>
              {(data?.roster || []).length === 0 ? (
                <tr>
                  <td colSpan={3} className="py-12 px-6 text-center text-white/50">
                    No student data available yet. Reconnect to load the latest roster.
                  </td>
                </tr>
              ) : (data?.roster || []).map((st) => (
                <tr
                  key={st.id}
                  onClick={() => setSelected(st)}
                  className="border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors cursor-pointer group"
                  title="View student progress"
                >
                  <td className="py-5 px-6 font-medium text-white flex items-center text-base">
                    <div className="w-10 h-10 rounded-full bg-brand-neon/20 text-brand-neon flex items-center justify-center mr-4 font-bold shadow-sm">
                      {st.name.charAt(0)}
                    </div>
                    {st.name}
                    <ChevronRight className="w-4 h-4 ml-2 text-white/30 group-hover:text-brand-neon group-hover:translate-x-1 transition-all" />
                  </td>
                  <td className="py-5 px-6">
                    <div className="flex items-center">
                      <div className="w-full bg-white/10 h-2 rounded-full mr-4 max-w-[120px] overflow-hidden">
                        <div className="bg-brand-neon h-full rounded-full" style={{ width: `${st.completion}%` }}></div>
                      </div>
                      <span className="text-white/70 font-medium">{st.completion}%</span>
                    </div>
                  </td>
                  <td className="py-5 px-6 text-white/70 font-medium">
                    <span className="bg-white/5 px-3 py-1 rounded-full text-brand-amber text-xs font-bold mr-2">{st.streak}</span> days
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {selected && (
        <StudentProgressModal
          token={token}
          studentId={selected.id}
          studentName={selected.name}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}