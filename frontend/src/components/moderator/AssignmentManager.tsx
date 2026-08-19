"use client";
import { useCallback, useEffect, useRef, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { Assignment, Submission } from '@/lib/types';
import { ClipboardList, Loader2, Send } from 'lucide-react';
import toast from 'react-hot-toast';

interface AssignmentManagerProps {
  token: string;
  classId: string;
  className: string;
}

// AssignmentManager owns the create-assignment form, the assignment table for
// the selected class, and the per-assignment submissions panel. Honest empty
// states are rendered whenever the backend has no data.
export default function AssignmentManager({ token, classId, className }: AssignmentManagerProps) {
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [viewing, setViewing] = useState<Assignment | null>(null);
  const [newAssignment, setNewAssignment] = useState({ title: '', description: '', activity_id: '', due_date: '' });
  const [isCreating, setIsCreating] = useState(false);
  // F10: monotonic request id — a slow response for a previously opened
  // assignment/class must never overwrite the data of a newer selection.
  const requestSeq = useRef(0);

  const authHeaders = () => ({ headers: { 'Authorization': `Bearer ${token}` } });

  const loadAssignments = useCallback(() => {
    if (!classId) {
      setAssignments([]);
      return;
    }
    const seq = ++requestSeq.current;
    fetchWithCache(`/moderator/classes/${classId}/assignments`, authHeaders())
      .then((d) => {
        if (seq === requestSeq.current) setAssignments(d.assignments || []);
      })
      .catch(() => {
        if (seq !== requestSeq.current) return;
        toast.error('Failed to load assignments');
        setAssignments([]);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [classId, token]);

  useEffect(() => {
    loadAssignments();
    setViewing(null);
    setSubmissions([]);
  }, [loadAssignments]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!classId || !newAssignment.title) {
      toast.error('Select a class and enter a title');
      return;
    }
    setIsCreating(true);
    try {
      const body: Record<string, string> = {
        title: newAssignment.title,
        description: newAssignment.description,
      };
      if (newAssignment.activity_id) body.activity_id = newAssignment.activity_id;
      if (newAssignment.due_date) body.due_date = new Date(newAssignment.due_date).toISOString();
      const res = await fetchWithCache(`/moderator/classes/${classId}/assignments`, {
        method: 'POST',
        ...authHeaders(),
        headers: { ...authHeaders().headers, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      // F8: a queued write is NOT created yet — say so instead of claiming success.
      if (res && res.queued) {
        toast.success('Assignment saved offline. Will sync when back online.', { icon: '💾' });
      } else {
        toast.success('Assignment created');
      }
      setNewAssignment({ title: '', description: '', activity_id: '', due_date: '' });
      loadAssignments();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create assignment');
    } finally {
      setIsCreating(false);
    }
  };

  const loadSubmissions = async (assignment: Assignment) => {
    setViewing(assignment);
    const seq = ++requestSeq.current;
    try {
      const d = await fetchWithCache(`/moderator/classes/${assignment.class_id}/assignments/${assignment.id}/submissions`, authHeaders());
      if (seq !== requestSeq.current) return;
      setSubmissions(d.submissions || []);
    } catch {
      if (seq !== requestSeq.current) return;
      toast.error('Failed to load submissions');
      setSubmissions([]);
    }
  };

  // M8: a zero-value due date ("0001-01-01T00:00:00Z") renders as "1/1/1" —
  // guard it so only real deadlines are shown.
  const formatDueDate = (due?: string | null) => {
    if (!due) return 'No deadline';
    const d = new Date(due);
    if (isNaN(d.getTime()) || d.getFullYear() < 2000) return 'No deadline';
    return d.toLocaleDateString();
  };

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-white mb-6 tracking-tight flex items-center">
        <ClipboardList className="w-6 h-6 mr-3 text-brand-amber" /> Assignments
        {className && <span className="text-brand-muted text-base ml-3">— {className}</span>}
      </h2>

      <form onSubmit={handleCreate} className="grid grid-cols-1 md:grid-cols-4 gap-3 mb-8">
        <input aria-label="Assignment title" value={newAssignment.title} onChange={e => setNewAssignment({...newAssignment, title: e.target.value})}
          className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30 md:col-span-2"
          placeholder="Assignment title (e.g. Homework 1)" />
        <input aria-label="Assignment description" value={newAssignment.description} onChange={e => setNewAssignment({...newAssignment, description: e.target.value})}
          className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30 md:col-span-2"
          placeholder="Description / instructions" />
        <input aria-label="Linked activity ID (optional)" value={newAssignment.activity_id} onChange={e => setNewAssignment({...newAssignment, activity_id: e.target.value})}
          className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30"
          placeholder="Linked activity ID (optional)" />
        <input aria-label="Due date" type="date" value={newAssignment.due_date} onChange={e => setNewAssignment({...newAssignment, due_date: e.target.value})}
          className="px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30" />
        <button type="submit" disabled={!classId || isCreating}
          className="btn-primary flex items-center justify-center gap-2 disabled:opacity-40 md:col-span-4">
          {isCreating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
          {isCreating ? 'Creating...' : 'Create Assignment'}
        </button>
      </form>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-white/10 text-white/50 bg-white/5">
              <th className="pb-4 pt-4 px-6 font-semibold rounded-tl-lg uppercase tracking-wider text-xs">Assignment</th>
              <th className="pb-4 pt-4 px-6 font-semibold uppercase tracking-wider text-xs">Due</th>
              <th className="pb-4 pt-4 px-6 font-semibold uppercase tracking-wider text-xs">Submissions</th>
              <th className="pb-4 pt-4 px-6 font-semibold rounded-tr-lg uppercase tracking-wider text-xs">Action</th>
            </tr>
          </thead>
          <tbody>
            {assignments.length === 0 ? (
              <tr><td colSpan={4} className="py-12 px-6 text-center text-white/50">No assignments for this class yet.</td></tr>
            ) : assignments.map(a => (
              <tr key={a.id} className="border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors">
                <td className="py-5 px-6 font-medium text-white">{a.title}
                  {a.description && <p className="text-white/50 text-xs mt-1">{a.description}</p>}
                </td>
                <td className="py-5 px-6 text-white/60">{formatDueDate(a.due_date)}</td>
                <td className="py-5 px-6 text-white/60">{a.submissions ?? 0} submitted</td>
                <td className="py-5 px-6">
                  <button onClick={() => loadSubmissions(a)} className="text-sm text-brand-neon font-bold hover:text-white transition-colors">
                    View Submissions
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {viewing && (
        <div className="card-glow p-6 border border-white/10">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-2xl font-bold text-white tracking-tight">Submissions: <span className="text-brand-neon">{viewing.title}</span></h3>
            <button onClick={() => { setViewing(null); setSubmissions([]); }} className="text-white/50 hover:text-white text-sm font-bold" aria-label="Close submissions panel">Close ✕</button>
          </div>
          {submissions.length === 0 ? (
            <p className="text-white/50 text-center py-8">No submissions yet.</p>
          ) : (
            <div className="space-y-4">
              {submissions.map(s => (
                <div key={s.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
                  <div className="flex items-center justify-between">
                    <p className="font-bold text-white">{s.learner_id.slice(0, 12)}…</p>
                    <p className="text-xs text-brand-muted">{new Date(s.submitted_at).toLocaleString()}</p>
                  </div>
                  <p className="text-white/70 mt-2 text-sm whitespace-pre-wrap">{s.note}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}