"use client";

// WP-1.5 class-creation wizard: create a class → share the invite code →
// import the roster from CSV. Every outcome is honest: the import report
// lists per-row failures and one-time generated passwords exactly once.

import { useState, useRef } from 'react';
import { X, Copy, Check, Upload, Users, GraduationCap, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';

interface Props {
  token: string;
  onClose: () => void;
  onCreated: () => void;
}

interface ImportReport {
  imported: number;
  skipped: number;
  passwords: Record<string, string>;
  errors: { row: number; email: string; reason: string }[];
}

export default function ClassWizard({ token, onClose, onCreated }: Props) {
  const t = useTranslations('wizard');
  const [step, setStep] = useState(1);
  const [name, setName] = useState('');
  const [grade, setGrade] = useState('');
  const [section, setSection] = useState('');
  const [creating, setCreating] = useState(false);
  const [classId, setClassId] = useState('');
  const [inviteCode, setInviteCode] = useState('');
  const [copied, setCopied] = useState(false);
  const [importing, setImporting] = useState(false);
  const [report, setReport] = useState<ImportReport | null>(null);
  const [fileName, setFileName] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const createClass = async () => {
    if (!name.trim() || !grade.trim() || !section.trim()) {
      toast.error(t('fillAll'));
      return;
    }
    setCreating(true);
    try {
      const cls = await fetchWithCache('/moderator/classes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ name: name.trim(), grade: grade.trim(), section: section.trim() }),
      });
      setClassId(cls.id);
      setInviteCode(cls.invite_code);
      setStep(2);
      onCreated();
    } catch {
      toast.error(t('createFailed'));
    } finally {
      setCreating(false);
    }
  };

  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(inviteCode);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error(t('copyFailed'));
    }
  };

  const importCsv = async (file: File) => {
    setImporting(true);
    setReport(null);
    const form = new FormData();
    form.append('file', file);
    try {
      const res = await fetchWithCache(`/moderator/classes/${classId}/roster/import`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        body: form,
      });
      setReport(res);
      setStep(3);
    } catch {
      toast.error(t('importFailed'));
    } finally {
      setImporting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/70 backdrop-blur-sm p-4 overflow-y-auto" onClick={onClose}>
      <div className="card-glow bg-[#0b0f17] border border-white/10 rounded-3xl p-8 w-full max-w-lg my-8 relative" onClick={(e) => e.stopPropagation()}>
        <button onClick={onClose} className="absolute top-4 right-4 text-white/50 hover:text-white transition-colors" aria-label="Close wizard">
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-2 mb-6">
          {[1, 2, 3].map((n) => (
            <div key={n} className={`h-1.5 flex-1 rounded-full transition-colors ${step >= n ? 'bg-brand-neon' : 'bg-white/10'}`} />
          ))}
        </div>

        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <GraduationCap className="w-6 h-6 text-brand-neon" /> {t('createTitle')}
            </h2>
            <p className="text-sm text-white/50">{t('createHint')}</p>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('classNamePh')}
              className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white placeholder-white/30 focus:border-brand-neon focus:outline-none"
            />
            <div className="grid grid-cols-2 gap-3">
              <input
                value={grade}
                onChange={(e) => setGrade(e.target.value)}
                placeholder={t('gradePh')}
                className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white placeholder-white/30 focus:border-brand-neon focus:outline-none"
              />
              <input
                value={section}
                onChange={(e) => setSection(e.target.value)}
                placeholder={t('sectionPh')}
                className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white placeholder-white/30 focus:border-brand-neon focus:outline-none"
              />
            </div>
            <button onClick={createClass} disabled={creating} className="btn-primary w-full py-3 font-bold inline-flex items-center justify-center gap-2 disabled:opacity-50">
              {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <GraduationCap className="w-4 h-4" />} {t('create')}
            </button>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-5">
            <h2 className="text-2xl font-bold text-white tracking-tight">{t('shareTitle')}</h2>
            <p className="text-sm text-white/50">{t('shareHint')}</p>
            <div className="flex items-center gap-3 bg-white/5 border border-dashed border-brand-neon/40 rounded-2xl px-5 py-6">
              <span className="text-4xl font-black tracking-[0.3em] text-brand-neon font-mono">{inviteCode}</span>
              <button onClick={copyCode} className="ml-auto p-2.5 rounded-xl bg-white/5 hover:bg-white/10 text-white/70 hover:text-white transition-colors" aria-label="Copy invite code">
                {copied ? <Check className="w-5 h-5 text-brand-neon" /> : <Copy className="w-5 h-5" />}
              </button>
            </div>
            <button onClick={() => fileRef.current?.click()} disabled={importing} className="w-full py-3 rounded-2xl bg-white/5 hover:bg-white/10 border border-white/10 text-white font-semibold inline-flex items-center justify-center gap-2 disabled:opacity-50">
              {importing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Upload className="w-4 h-4" />} {t('importCsv')}
            </button>
            <p className="text-xs text-white/40">{t('csvFormat')}</p>
            <input
              ref={fileRef}
              type="file"
              accept=".csv,text/csv"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) {
                  setFileName(f.name);
                  importCsv(f);
                }
              }}
            />
            {fileName && <p className="text-xs text-white/50 flex items-center gap-1"><Users className="w-3.5 h-3.5" /> {fileName}</p>}
            <button onClick={() => setStep(3)} className="w-full py-2.5 text-sm text-white/50 hover:text-white transition-colors">
              {t('skip')}
            </button>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4">
            <h2 className="text-2xl font-bold text-white tracking-tight">{t('doneTitle')}</h2>
            {report ? (
              <div className="space-y-3">
                <div className="flex gap-3">
                  <div className="flex-1 bg-white/5 rounded-2xl p-4 text-center">
                    <p className="text-2xl font-bold text-brand-neon">{report.imported}</p>
                    <p className="text-xs text-white/50 uppercase tracking-wider mt-1">{t('imported')}</p>
                  </div>
                  {report.skipped > 0 && (
                    <div className="flex-1 bg-white/5 rounded-2xl p-4 text-center">
                      <p className="text-2xl font-bold text-brand-amber">{report.skipped}</p>
                      <p className="text-xs text-white/50 uppercase tracking-wider mt-1">{t('skipped')}</p>
                    </div>
                  )}
                </div>

                {Object.keys(report.passwords).length > 0 && (
                  <div className="bg-brand-amber/10 border border-brand-amber/30 rounded-2xl p-4">
                    <p className="text-xs font-bold text-brand-amber uppercase tracking-wider mb-2">{t('tempPasswords')}</p>
                    <ul className="space-y-1 text-xs text-white/80 font-mono">
                      {Object.entries(report.passwords).map(([email, pwd]) => (
                        <li key={email}>{email} → <span className="text-brand-neon">{pwd}</span></li>
                      ))}
                    </ul>
                    <p className="text-[11px] text-white/40 mt-2">{t('passwordsOnce')}</p>
                  </div>
                )}

                {report.errors.length > 0 && (
                  <div className="bg-red-500/10 border border-red-500/30 rounded-2xl p-4">
                    <p className="text-xs font-bold text-red-400 uppercase tracking-wider mb-2">{t('rowErrors')}</p>
                    <ul className="space-y-1 text-xs text-white/70">
                      {report.errors.map((e, i) => (
                        <li key={i}>Row {e.row} · {e.email || '—'} · {e.reason}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ) : (
              <p className="text-white/60">{t('noImport')}</p>
            )}
            <button onClick={onClose} className="btn-primary w-full py-3 font-bold">{t('finish')}</button>
          </div>
        )}
      </div>
    </div>
  );
}