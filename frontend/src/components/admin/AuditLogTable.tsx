"use client";
import { useEffect, useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { AuditLogEntry } from '@/lib/types';
import { Download, ScrollText } from 'lucide-react';
import toast from 'react-hot-toast';

interface AuditLogTableProps {
  token: string;
  onExport?: () => void;
  exporting?: boolean;
}

// AuditLogTable loads and renders the audit trail. The export button is a
// page-level concern (download flow), so it is injected as a callback — the
// table itself only ever renders real entries or an honest empty state.
export default function AuditLogTable({ token, onExport, exporting }: AuditLogTableProps) {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchWithCache('/admin/audit-log', { headers: { 'Authorization': `Bearer ${token}` } })
      .then((d) => setLogs(d.audit_logs || []))
      .catch(() => {
        toast.error('Failed to load audit log');
        setLogs([]);
      })
      .finally(() => setLoading(false));
  }, [token]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <ScrollText className="w-6 h-6 text-brand-blue" />
          <h2 className="text-xl font-bold text-white tracking-tight">Audit Log</h2>
        </div>
        {onExport && (
          <button onClick={onExport} disabled={exporting}
            className="text-sm text-brand-neon font-medium hover:underline flex items-center gap-1 disabled:opacity-50">
            <Download className="w-4 h-4" /> {exporting ? 'Exporting…' : 'Export Students CSV'}
          </button>
        )}
      </div>
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <div className="w-10 h-10 border-4 border-white/10 border-t-brand-neon rounded-full animate-spin"></div>
        </div>
      ) : (
        <div className="overflow-x-auto max-h-72 overflow-y-auto">
          <table className="w-full text-left text-xs">
            <thead className="sticky top-0 bg-black/80 backdrop-blur">
              <tr className="border-b border-white/10 text-white/40 uppercase tracking-wider text-[10px] font-bold">
                <th className="pb-3 pr-3">Time</th>
                <th className="pb-3 pr-3">Actor</th>
                <th className="pb-3 pr-3">Action</th>
                <th className="pb-3">Detail</th>
              </tr>
            </thead>
            <tbody>
              {logs.length === 0 ? (
                <tr><td colSpan={4} className="py-8 text-center text-white/40">No audit entries yet.</td></tr>
              ) : logs.map(e => (
                <tr key={e.id} className="border-b border-white/5 last:border-0">
                  <td className="py-3 pr-3 text-white/40 whitespace-nowrap">{new Date(e.created_at).toLocaleString()}</td>
                  <td className="py-3 pr-3 text-white/70">{e.user_id.slice(0, 12)}…</td>
                  <td className="py-3 pr-3"><span className="px-2 py-0.5 rounded-full bg-brand-neon/10 text-brand-neon text-[10px] font-bold">{e.action}</span></td>
                  <td className="py-3 text-white/60">{e.detail}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}