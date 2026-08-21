'use client';

import { useEffect, useState, Fragment } from 'react';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';
import toast from 'react-hot-toast';
import { BookOpenCheck, Download, StickyNote, ChevronDown, ChevronUp } from 'lucide-react';
import { GradebookStudent, GradebookRow } from '@/lib/types';

// WP-2.3 RC-08: the honest gradebook. Every number is a real stored
// LearnerActivity row (accuracy + attempts); a learner with no rows renders
// the honest "Not yet assessed" state — never an invented grade. The CSV
// export downloads the backend's sanitized file (real data only).
export default function GradebookOverview({
  token,
  selectedClass,
}: {
  token: string;
  selectedClass: string;
}) {
  const t = useTranslations('gradebook');
  const [students, setStudents] = useState<GradebookStudent[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [noteOpen, setNoteOpen] = useState<string | null>(null);
  const [noteText, setNoteText] = useState<Record<string, string>>({});
  const [noteSaved, setNoteSaved] = useState<Record<string, string>>({});
  const [savingNote, setSavingNote] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedClass) return;
    setLoading(true);
    setStudents(null);
    fetchWithCache(`/moderator/gradebook?class_id=${selectedClass}`)
      .then((d) => setStudents(d.students || []))
      .catch(() => setStudents([]))
      .finally(() => setLoading(false));
  }, [selectedClass]);

  const exportCSV = async () => {
    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/moderator/gradebook.csv?class_id=${selectedClass}`,
        { headers: { Authorization: `Bearer ${token}` }, cache: 'no-store' }
      );
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'gradebook.csv';
      a.click();
      URL.revokeObjectURL(url);
      toast.success(t('csvOk'), { icon: '📄' });
    } catch {
      toast.error(t('csvError'));
    }
  };

  const openNote = async (studentId: string) => {
    if (noteOpen === studentId) {
      setNoteOpen(null);
      return;
    }
    setNoteOpen(studentId);
    if (noteText[studentId] !== undefined) return;
    try {
      const res = await fetchWithCache(`/moderator/students/${studentId}/note`);
      // WP-2.3: honest null — no note yet, the editor starts empty.
      setNoteText((prev) => ({ ...prev, [studentId]: typeof res?.note === 'string' ? res.note : '' }));
      if (typeof res?.updated_at === 'string') {
        setNoteSaved((prev) => ({ ...prev, [studentId]: res.updated_at }));
      }
    } catch {
      setNoteText((prev) => ({ ...prev, [studentId]: '' }));
    }
  };

  const saveNote = async (studentId: string) => {
    const text = (noteText[studentId] || '').trim();
    if (!text) {
      toast.error(t('noteRequired'));
      return;
    }
    setSavingNote(studentId);
    try {
      const res = await fetchWithCache(`/moderator/students/${studentId}/note`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ note: text }),
      });
      setNoteSaved((prev) => ({ ...prev, [studentId]: res?.updated_at || new Date().toISOString() }));
      toast.success(t('noteSaved'), { icon: '🗒️' });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('noteError'));
    } finally {
      setSavingNote(null);
    }
  };

  // Columns are the union of all activity ids in the class (the backend
  // returns every activity per student; all students share the same set).
  const activities: string[] = students && students.length > 0
    ? students[0].rows.map((r) => r.activity_id)
    : [];
  const activityTitles: Record<string, string> = {};
  if (students && students.length > 0) {
    students[0].rows.forEach((r) => {
      activityTitles[r.activity_id] = r.title;
    });
  }

  const cellFor = (row: GradebookRow | undefined) => {
    if (!row || row.attempts === 0) {
      return {
        label: t('notAssessed'),
        cls: 'text-white/30',
        dot: 'bg-white/10',
      };
    }
    const pct = Math.round(row.accuracy * 100);
    const statusCls =
      row.status === 'completed' ? 'text-brand-neon' :
      row.status === 'needs-practice' ? 'text-brand-amber' : 'text-white/60';
    return {
      label: `${pct}% · ${row.attempts}×`,
      cls: statusCls,
      dot: row.status === 'completed' ? 'bg-brand-neon' : row.status === 'needs-practice' ? 'bg-brand-amber' : 'bg-white/30',
    };
  };

  const fmtUpdated = (iso?: string) => {
    if (!iso) return '';
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    return `${d.toLocaleDateString()} ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  };

  return (
    <div className="card-glow p-6 border border-white/10">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <h2 className="text-2xl font-bold text-white tracking-tight flex items-center">
          <BookOpenCheck className="w-6 h-6 mr-3 text-brand-neon" /> {t('title')}
        </h2>
        <button
          onClick={exportCSV}
          disabled={!selectedClass || loading}
          className="bg-white/5 hover:bg-white/10 text-white transition-all px-4 py-2 rounded-full flex items-center gap-2 backdrop-blur-xl border border-white/10 font-semibold text-sm disabled:opacity-40"
        >
          <Download className="w-4 h-4" /> {t('exportCsv')}
        </button>
      </div>

      {!selectedClass && <p className="text-white/50 text-sm">{t('pickClass')}</p>}

      {selectedClass && loading && <p className="text-white/50 text-sm">{t('loading')}</p>}

      {selectedClass && !loading && students && students.length === 0 && (
        <p className="text-white/50 text-sm">{t('noStudents')}</p>
      )}

      {selectedClass && !loading && students && students.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm min-w-[560px]">
            <thead>
              <tr className="border-b border-white/10 text-brand-muted uppercase tracking-wider text-[10px] font-bold">
                <th className="pb-3 pr-4">{t('student')}</th>
                {activities.map((id) => (
                  <th key={id} className="pb-3 pr-4 max-w-[140px] truncate" title={activityTitles[id]}>
                    {activityTitles[id]}
                  </th>
                ))}
                <th className="pb-3">{t('note')}</th>
              </tr>
            </thead>
            <tbody>
              {students.map((s) => {
                const rowMap: Record<string, GradebookRow> = {};
                s.rows.forEach((r) => { rowMap[r.activity_id] = r; });
                const isOpen = noteOpen === s.student_id;
                return (
                  <Fragment key={s.student_id}>
                    <tr className="border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors">
                      <td className="py-3 pr-4 font-medium text-white whitespace-nowrap">{s.name}</td>
                      {activities.map((id) => {
                        const cell = cellFor(rowMap[id]);
                        return (
                          <td key={id} className={`py-3 pr-4 whitespace-nowrap ${cell.cls}`}>
                            <span className="inline-flex items-center gap-1.5">
                              <span className={`w-2 h-2 rounded-full ${cell.dot}`} />
                              {cell.label}
                            </span>
                          </td>
                        );
                      })}
                      <td className="py-3">
                        <button
                          onClick={() => openNote(s.student_id)}
                          aria-expanded={isOpen}
                          className="inline-flex items-center gap-1.5 text-brand-neon hover:text-white transition-colors text-xs font-semibold"
                        >
                          <StickyNote className="w-4 h-4" />
                          {isOpen ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                        </button>
                      </td>
                    </tr>
                    {isOpen && (
                      <tr className="border-b border-white/5 bg-black/20">
                        <td colSpan={activities.length + 2} className="py-3">
                          <div className="flex flex-col sm:flex-row gap-3 items-stretch sm:items-center">
                            <textarea
                              value={noteText[s.student_id] || ''}
                              onChange={(e) => setNoteText((prev) => ({ ...prev, [s.student_id]: e.target.value }))}
                              placeholder={t('notePlaceholder')}
                              maxLength={500}
                              rows={2}
                              className="flex-1 px-3 py-2 bg-black/40 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30 text-sm"
                            />
                            <div className="flex flex-col sm:items-end gap-1">
                              <button
                                onClick={() => saveNote(s.student_id)}
                                disabled={savingNote === s.student_id}
                                className="btn-primary px-4 py-2 text-sm flex items-center gap-2 disabled:opacity-50"
                              >
                                <StickyNote className="w-4 h-4" />
                                {savingNote === s.student_id ? t('saving') : t('saveNote')}
                              </button>
                              {noteSaved[s.student_id] && (
                                <span className="text-[10px] text-white/40">{t('savedAt')} {fmtUpdated(noteSaved[s.student_id])}</span>
                              )}
                            </div>
                          </div>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
          <p className="text-[11px] text-white/40 mt-3">{t('honestNote')}</p>
        </div>
      )}
    </div>
  );
}