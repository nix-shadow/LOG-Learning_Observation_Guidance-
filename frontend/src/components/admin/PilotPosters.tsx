"use client";

// WP-3.3 (RC-10): QR pilot posters + honest pilot measurement. Each activity
// gets a printable QR pointing at /qr/<activityId> — scanning warms the
// learner's offline cache and records a scan. The stats panel renders only
// real stored numbers (zeros are real zeros, never invented).
import { useEffect, useState, useCallback } from "react";
import QRCode from "qrcode";
import { QrCode, RefreshCcw, Loader2 } from "lucide-react";
import { Activity, PilotStats } from "@/lib/types";

type Poster = { id: string; title: string; topic: string };

export default function PilotPosters({ token }: { token: string }) {
  const [activities, setActivities] = useState<Poster[] | null>(null);
  const [qrUrls, setQrUrls] = useState<Record<string, string>>({});
  const [stats, setStats] = useState<PilotStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [error, setError] = useState(false);

  const loadStats = useCallback(() => {
    setStatsLoading(true);
    fetch(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:6101/api/v1"}/admin/pilot/stats`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((body) => setStats(body?.stats ?? null))
      .catch(() => setStats(null))
      .finally(() => setStatsLoading(false));
  }, [token]);

  useEffect(() => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:6101/api/v1";
    fetch(`${apiBase}/learning-journey`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((body) => {
        const acts: Poster[] = (body?.activities ?? []).map((a: Activity) => ({
          id: a.id,
          title: a.title,
          topic: a.topic,
        }));
        setActivities(acts);
        setError(false);
        const base = typeof window !== "undefined" ? window.location.origin : "";
        Promise.all(
          acts.map(async (a) => {
            const url = await QRCode.toDataURL(`${base}/qr/${a.id}`, {
              width: 220,
              margin: 1,
              color: { dark: "#000000", light: "#ffffff" },
            }).catch(() => "");
            return [a.id, url] as const;
          })
        ).then((pairs) => setQrUrls(Object.fromEntries(pairs)));
      })
      .catch(() => {
        setActivities(null);
        setError(true);
      });
    loadStats();
  }, [token, loadStats]);

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-white tracking-tight flex items-center gap-2">
            <QrCode className="w-5 h-5 text-brand-neon" />
            QR Pilot Posters
          </h2>
          <p className="text-sm text-white/60 mt-1">
            Print these and stick them on classroom walls. Scanning warms the
            learner&apos;s offline cache and records the poster in the pilot.
          </p>
        </div>
        <button
          onClick={loadStats}
          className="p-2.5 rounded-xl bg-white/5 border border-white/10 text-white/60 hover:text-white hover:bg-white/10 transition-all"
          aria-label="Refresh pilot stats"
        >
          <RefreshCcw className="w-4 h-4" />
        </button>
      </div>

      {error && (
        <p className="text-sm text-brand-muted">
          Poster list could not be loaded — check the connection and retry.
        </p>
      )}

      {activities && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {activities.map((a) => (
            <div key={a.id} className="bg-white rounded-2xl p-4 flex flex-col items-center text-center">
              {qrUrls[a.id] ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={qrUrls[a.id]} alt={`QR code for ${a.title}`} className="w-28 h-28" />
              ) : (
                <div className="w-28 h-28 flex items-center justify-center text-neutral-400">
                  <Loader2 className="w-6 h-6 animate-spin" />
                </div>
              )}
              <p className="mt-3 text-[11px] font-bold text-neutral-800 leading-tight line-clamp-2">{a.title}</p>
              <p className="text-[10px] text-neutral-500 mt-1 uppercase tracking-wider">{a.topic}</p>
              <p className="text-[9px] text-neutral-400 mt-1 font-mono">/qr/{a.id}</p>
            </div>
          ))}
        </div>
      )}

      {/* Honest pilot measurement: every number below is a real stored scan
          row. An empty pilot shows zeros — never invented engagement. */}
      <div className="border-t border-white/10 pt-6">
        <h3 className="text-sm font-bold text-white/70 uppercase tracking-wider mb-4">Pilot Measurement</h3>
        {statsLoading && !stats ? (
          <p className="text-sm text-brand-muted flex items-center gap-2">
            <Loader2 className="w-4 h-4 animate-spin" /> Loading real scan data…
          </p>
        ) : stats ? (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
              <p className="text-2xl font-extrabold text-white">{stats.total_scans}</p>
              <p className="text-[10px] font-bold text-white/50 uppercase tracking-wider">Total scans</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
              <p className="text-2xl font-extrabold text-white">{stats.scans_today}</p>
              <p className="text-[10px] font-bold text-white/50 uppercase tracking-wider">Last 24h</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
              <p className="text-2xl font-extrabold text-brand-teal">{stats.starts}</p>
              <p className="text-[10px] font-bold text-white/50 uppercase tracking-wider">Clicked through</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
              <p className="text-2xl font-extrabold text-brand-amber">
                {stats.total_scans > 0 ? Math.round(stats.start_rate * 100) : 0}%
              </p>
              <p className="text-[10px] font-bold text-white/50 uppercase tracking-wider">Start rate</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
              <p className="text-2xl font-extrabold text-white">{stats.distinct_posters}</p>
              <p className="text-[10px] font-bold text-white/50 uppercase tracking-wider">Posters seen</p>
            </div>
          </div>
        ) : (
          <p className="text-sm text-brand-muted">
            No pilot data available. Scans will appear here as real posters are scanned.
          </p>
        )}
      </div>
    </div>
  );
}