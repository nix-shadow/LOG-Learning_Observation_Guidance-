"use client";
import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { SchoolClass, Learner } from '@/lib/types';
import { GraduationCap, Loader2, PlusCircle, Users } from 'lucide-react';
import toast from 'react-hot-toast';

interface ClassManagerProps {
  token: string;
}

// ClassManager owns the class lifecycle UI: create class, enroll students, and
// the class table. All data flows through the fetchWithCache seam so the
// offline layer and honest empty/error states are exercised together.
export default function ClassManager({ token }: ClassManagerProps) {
  const [classes, setClasses] = useState<SchoolClass[]>([]);
  const [users, setUsers] = useState<Learner[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [isEnrolling, setIsEnrolling] = useState(false);
  const [newClass, setNewClass] = useState({ name: '', grade: '', section: '', teacher_id: '' });
  const [selectedClass, setSelectedClass] = useState('');
  const [selectedStudents, setSelectedStudents] = useState<string[]>([]);

  const authHeaders = () => ({ headers: { 'Authorization': `Bearer ${token}` } });

  const loadClasses = () => {
    fetchWithCache('/admin/classes', authHeaders())
      .then((d) => {
        setClasses(d.classes || []);
        setLoadError(false);
      })
      .catch(() => {
        // F14: a failed load is not an empty state — flag it distinctly.
        setLoadError(true);
        setClasses([]);
      });
  };

  useEffect(() => {
    loadClasses();
    fetchWithCache('/admin/users', authHeaders())
      .then((d) => {
        setUsers(d.users || []);
        setLoadError(false);
      })
      .catch(() => {
        setLoadError(true);
        setUsers([]);
      })
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const handleCreateClass = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newClass.name || !newClass.grade || !newClass.section || !newClass.teacher_id) {
      toast.error('All class fields are required');
      return;
    }
    setIsCreating(true);
    try {
      const res = await fetchWithCache('/admin/classes', {
        method: 'POST',
        ...authHeaders(),
        headers: { ...authHeaders().headers, 'Content-Type': 'application/json' },
        body: JSON.stringify(newClass),
      });
      // F8: queued writes are pending, not created.
      if (res && res.queued) {
        toast.success('Class saved offline. Will sync when back online.', { icon: '💾' });
      } else {
        toast.success(`Class "${res.name}" created`);
      }
      setNewClass({ name: '', grade: '', section: '', teacher_id: '' });
      loadClasses();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create class');
    } finally {
      setIsCreating(false);
    }
  };

  const handleEnroll = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedClass || selectedStudents.length === 0) {
      toast.error('Select a class and at least one student');
      return;
    }
    setIsEnrolling(true);
    try {
      const res = await fetchWithCache(`/admin/classes/${selectedClass}/enroll`, {
        method: 'POST',
        ...authHeaders(),
        headers: { ...authHeaders().headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_ids: selectedStudents }),
      });
      if (res && res.queued) {
        toast.success('Enrollment saved offline. Will sync when back online.', { icon: '💾' });
      } else {
        toast.success(`Enrolled! Class now has ${res.member_count} students`);
      }
      setSelectedStudents([]);
      loadClasses();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to enroll students');
    } finally {
      setIsEnrolling(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[30vh]">
        <div className="w-12 h-12 border-4 border-white/10 border-t-brand-neon rounded-full animate-spin"></div>
      </div>
    );
  }

  const teachers = users.filter((u) => (u as Learner & { role?: string }).role === 'MODERATOR');
  const students = users.filter((u) => (u as Learner & { role?: string }).role === 'STUDENT');

  return (
    <div className="space-y-8">
      <div className="flex items-center gap-3">
        <GraduationCap className="w-6 h-6 text-brand-neon" />
        <h2 className="text-xl font-bold text-white tracking-tight">Classes &amp; Enrollment</h2>
      </div>

      {loadError && (
        <div className="p-4 bg-red-500/10 border border-red-500/30 rounded-xl text-red-300 text-sm">
          Could not load class data. Check your connection — the page below may be out of date.
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Create class */}
        <form onSubmit={handleCreateClass} className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-white/50">Create Class</h3>
          <div className="grid grid-cols-2 gap-3">
            <input aria-label="Class name" value={newClass.name} onChange={e => setNewClass({...newClass, name: e.target.value})}
              className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30 col-span-2"
              placeholder="Class name (e.g. Grade 10 A)" />
            <input aria-label="Grade" value={newClass.grade} onChange={e => setNewClass({...newClass, grade: e.target.value})}
              className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30"
              placeholder="Grade (e.g. 10)" />
            <input aria-label="Section" value={newClass.section} onChange={e => setNewClass({...newClass, section: e.target.value})}
              className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30"
              placeholder="Section (e.g. A)" />
          </div>
          <select aria-label="Assign teacher" value={newClass.teacher_id} onChange={e => setNewClass({...newClass, teacher_id: e.target.value})}
            className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none">
            <option value="">Assign teacher…</option>
            {teachers.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
          <button type="submit" disabled={isCreating} className="btn-primary w-full flex items-center justify-center gap-2 disabled:opacity-40">
            {isCreating ? <Loader2 className="w-4 h-4 animate-spin" /> : <PlusCircle className="w-4 h-4" />}
            {isCreating ? 'Creating...' : 'Create Class'}
          </button>
        </form>

        {/* Enroll students */}
        <form onSubmit={handleEnroll} className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-white/50">Enroll Students</h3>
          <select aria-label="Select class" value={selectedClass} onChange={e => setSelectedClass(e.target.value)}
            className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none">
            <option value="">Select class…</option>
            {classes.map(c => <option key={c.id} value={c.id}>{c.name} ({c.member_count ?? 0} students)</option>)}
          </select>
          <div className="max-h-40 overflow-y-auto border border-white/10 rounded-xl p-3 space-y-2">
            {students.length === 0 && <p className="text-brand-muted text-sm">No students registered yet.</p>}
            {students.map(s => (
              <label key={s.id} className="flex items-center gap-3 text-white/80 text-sm cursor-pointer hover:text-white">
                <input type="checkbox" checked={selectedStudents.includes(s.id)}
                  onChange={e => setSelectedStudents(prev => e.target.checked ? [...prev, s.id] : prev.filter(id => id !== s.id))}
                  className="accent-brand-neon" />
                {s.name}
              </label>
            ))}
          </div>
          <button type="submit" disabled={isEnrolling}
            className="btn-secondary w-full flex items-center justify-center gap-2 bg-white/5 border-white/10 hover:bg-white/10 disabled:opacity-40">
            {isEnrolling ? <Loader2 className="w-4 h-4 animate-spin" /> : <Users className="w-4 h-4" />}
            {isEnrolling ? 'Enrolling...' : 'Enroll Selected'}
          </button>
        </form>
      </div>

      {/* Class list */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-white/10 text-brand-muted uppercase tracking-wider text-[10px] font-bold">
              <th className="pb-3">Class</th>
              <th className="pb-3">Grade</th>
              <th className="pb-3">Section</th>
              <th className="pb-3">Students</th>
            </tr>
          </thead>
          <tbody>
            {classes.length === 0 ? (
              <tr><td colSpan={4} className="py-8 text-center text-brand-muted">
                {loadError ? 'Class list unavailable — reconnect to load.' : 'No classes yet.'}
              </td></tr>
            ) : classes.map(c => (
              <tr key={c.id} className="border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors">
                <td className="py-4 font-medium text-white">{c.name}</td>
                <td className="py-4 text-white/60">{c.grade}</td>
                <td className="py-4 text-white/60">{c.section}</td>
                <td className="py-4 text-white/60">{c.member_count ?? 0}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}