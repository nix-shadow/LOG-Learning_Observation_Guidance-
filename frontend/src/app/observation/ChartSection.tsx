"use client";
import { ChartDataPoint } from "@/lib/types";
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
  if (!asOf) return null;
  const d = new Date(asOf);
  if (Number.isNaN(d.getTime())) return null;
  return (
    <p className="text-[11px] text-brand-faint mt-4 text-center" aria-live="polite">
      Updated {d.toLocaleDateString()} at {d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
    </p>
  );
}

export default function ChartSection({ chartData, asOf }: { chartData: ChartDataPoint[]; asOf?: string | null }) {
  const reduceMotion = useReducedMotion();
  const animation = reduceMotion ? false : true;

  return (
    <div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
           <h3 className="text-xl font-bold text-white mb-8 flex items-center tracking-tight">
             <TrendingUp className="w-6 h-6 mr-3 text-brand-neon"/> Performance Trend
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
                     <stop offset="5%" stopColor="#00F0FF" stopOpacity={0.3}/>
                     <stop offset="95%" stopColor="#00F0FF" stopOpacity={0}/>
                   </linearGradient>
                 </defs>
                 <XAxis dataKey="name" tick={tickStyle} axisLine={false} tickLine={false} />
                 <YAxis tick={tickStyle} axisLine={false} tickLine={false} />
                 <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.05)" />
                 <Tooltip 
                   contentStyle={{ borderRadius: '16px', border: '1px solid rgba(255,255,255,0.1)', backgroundColor: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(16px)', color: '#fff' }} 
                   itemStyle={{ color: '#00F0FF', fontWeight: 'bold' }}
                 />
                 <Area type="monotone" dataKey="score" stroke="#00F0FF" strokeWidth={3} fillOpacity={1} fill="url(#colorScore)" isAnimationActive={animation} />
               </AreaChart>
            </ResponsiveContainer>
            )}
          </div>
        </div>

        <div className="card-glow bg-black/40 backdrop-blur-2xl border-white/10 p-8">
           <h3 className="text-xl font-bold text-white mb-8 flex items-center tracking-tight">
             <BarChart2 className="w-6 h-6 mr-3 text-brand-amber"/> Daily Engagement
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
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.05)" />
                <Tooltip 
                  cursor={{fill: 'rgba(255,255,255,0.05)'}} 
                  contentStyle={{ borderRadius: '16px', border: '1px solid rgba(255,255,255,0.1)', backgroundColor: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(16px)', color: '#fff' }} 
                  itemStyle={{ color: '#FFB703', fontWeight: 'bold' }}
                />
                <Bar dataKey="duration" fill="#FFB703" radius={[6, 6, 0, 0]} isAnimationActive={animation} />
              </BarChart>
           </ResponsiveContainer>
           )}
         </div>
        </div>
      </div>
      <AsOfLabel asOf={asOf} />
    </div>
  );
}