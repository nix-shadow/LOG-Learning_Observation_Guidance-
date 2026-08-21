"use client";
import { ChartDataPoint } from "@/lib/types";
import { useTranslations } from 'next-intl';
import { useLocaleCtx } from '@/i18n/LocaleProvider';
import { formatBs } from '@/lib/bikramSambat';
import { BarChart2, TrendingUp } from 'lucide-react';
import { useReducedMotion } from 'framer-motion';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';

// WP-0.3 a11y research round:
//  - tick fill rgba(255,255,255,0.4) ≈ 3.8:1 failed AA — now --brand-muted
//    (9.2:1 against the card surface)
//  - charts carry aria-label/title so the trend is readable to AT
//  - isAnimationActive={false} under prefers-reduced-motion (WCAG 2.3.3)
// WP-0.2 as_of (research round): when the server timestamp is present it is
// rendered verbatim under the chart grid, so a cached payload (24h TTL) never
// masquerades as live data — the staleness is visible, not hidden.
const tickStyle = { fontSize: 12, fill: 'var(--brand-muted)' };

function AsOfLabel({ asOf }: { asOf?: string | null }) {
  const t = useTranslations('common');
  const { locale } = useLocaleCtx();
  if (!asOf) return null;
  const d = new Date(asOf);
  if (Number.isNaN(d.getTime())) return null;
  const bs = formatBs(d, locale);
  return (
    <p className="text-[11px] text-brand-faint mt-4 text-center" aria-live="polite">
      {t('updated')} {d.toLocaleDateString()} {t('at')} {d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
      {bs ? ` (${bs})` : ''}
    </p>
  );
}

export default function ChartSection({ chartData, asOf }: { chartData: ChartDataPoint[]; asOf?: string | null }) {
  const t = useTranslations('chart');
  const reduceMotion = useReducedMotion();
  const animation = reduceMotion ? false : true;

  return (
    <div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
           <h3 className="text-xl font-bold text-white mb-8 flex items-center tracking-tight">
             <TrendingUp className="w-6 h-6 mr-3 text-brand-neon"/> {t('performance')}
           </h3>
           <div className="h-72 w-full">
             {chartData.length === 0 ? (
               <div className="h-full flex flex-col items-center justify-center text-center">
                 <BarChart2 className="w-8 h-8 text-brand-faint mb-3" />
                 <p className="text-brand-muted text-sm">No activity data yet — your progress chart appears here after your first completed activity.</p>
               </div>
             ) : (
<ResponsiveContainer width="100%" height="100%">
               <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                 <title>Performance trend: daily score</title>
                 <defs>
                   <linearGradient id="colorScore" x1="0" y1="0" x2="0" y2="1">
                     <stop offset="5%" stopColor="#2563EB" stopOpacity={0.3}/>
                     <stop offset="95%" stopColor="#2563EB" stopOpacity={0}/>
                   </linearGradient>
                 </defs>
                 <XAxis dataKey="name" tick={tickStyle} axisLine={false} tickLine={false} />
                 <YAxis tick={tickStyle} axisLine={false} tickLine={false} />
                 <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(148,163,184,0.3)" />
                 <Tooltip 
                   contentStyle={{ borderRadius: '16px', border: '1px solid rgba(255,255,255,0.1)', backgroundColor: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(16px)', color: '#fff' }} 
                   itemStyle={{ color: '#2563EB', fontWeight: 'bold' }}
                 />
<Area type="monotone" dataKey="score" stroke="#2563EB" strokeWidth={3} fillOpacity={1} fill="url(#colorScore)" isAnimationActive={animation} />
                  {/* WP-1.2 RC-02: real practice accuracy (0-100), derived
                      from actual completion rows — never fabricated. */}
                  <Area type="monotone" dataKey="accuracy" name="accuracy" stroke="#0D9488" strokeWidth={2} strokeDasharray="6 4" fillOpacity={0} isAnimationActive={animation} />
                </AreaChart>
            </ResponsiveContainer>
            )}
          </div>
        </div>

        <div className="card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
           <h3 className="text-xl font-bold text-white mb-8 flex items-center tracking-tight">
             <BarChart2 className="w-6 h-6 mr-3 text-brand-amber"/> {t('engagement')}
           </h3>
<div className="h-72 w-full">
           {chartData.length === 0 ? (
             <div className="h-full flex flex-col items-center justify-center text-center">
               <BarChart2 className="w-8 h-8 text-brand-faint mb-3" />
               <p className="text-brand-muted text-sm">No activity data yet — your engagement chart appears here after your first completed activity.</p>
             </div>
           ) : (
           <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <title>Daily engagement: minutes practiced</title>
                <XAxis dataKey="name" tick={tickStyle} axisLine={false} tickLine={false} />
                <YAxis tick={tickStyle} axisLine={false} tickLine={false} />
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(148,163,184,0.3)" />
                <Tooltip 
                  cursor={{fill: 'rgba(148,163,184,0.15)'}} 
                  contentStyle={{ borderRadius: '16px', border: '1px solid rgba(255,255,255,0.1)', backgroundColor: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(16px)', color: '#fff' }} 
                  itemStyle={{ color: '#F59E0B', fontWeight: 'bold' }}
                />
                <Bar dataKey="duration" fill="#F59E0B" radius={[6, 6, 0, 0]} isAnimationActive={animation} />
              </BarChart>
           </ResponsiveContainer>
           )}
         </div>
        </div>
      </div>
      {/* WP-1.2 RC-02: honest practice totals derived from the same rows the
          charts render — a period with no practice shows a real zero. */}
      <PracticeTotals chartData={chartData} />
      <AsOfLabel asOf={asOf} />
    </div>
  );
}

function PracticeTotals({ chartData }: { chartData: ChartDataPoint[] }) {
  const t = useTranslations('chart');
  const attempts = chartData.reduce((sum, d) => sum + (d.attempts ?? 0), 0);
  const totalWeight = chartData.reduce((sum, d) => sum + (d.attempts ?? 0), 0);
  const accuracy = totalWeight > 0
    ? chartData.reduce((sum, d) => sum + (d.accuracy ?? 0) * (d.attempts ?? 0), 0) / totalWeight
    : 0;
  return (
    <div className="mt-6 grid grid-cols-2 sm:grid-cols-3 gap-4" aria-label="Practice totals">
      <div className="card-glow p-4 text-center">
        <p className="text-2xl font-bold text-brand-neon">{attempts}</p>
        <p className="text-[11px] font-bold uppercase tracking-widest text-brand-muted">{t('practiceAttempts')}</p>
      </div>
      <div className="card-glow p-4 text-center">
        <p className="text-2xl font-bold text-brand-amber">{accuracy > 0 ? `${Math.round(accuracy)}%` : '—'}</p>
        <p className="text-[11px] font-bold uppercase tracking-widest text-brand-muted">{t('meanAccuracy')}</p>
      </div>
      <div className="card-glow p-4 text-center col-span-2 sm:col-span-1">
        <p className="text-2xl font-bold text-white">{chartData.length}</p>
        <p className="text-[11px] font-bold uppercase tracking-widest text-brand-muted">{t('activeDays')}</p>
      </div>
    </div>
  );
}