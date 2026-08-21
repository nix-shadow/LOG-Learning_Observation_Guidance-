"use client";

// WP-1.5: join a teacher's class with the invite code they share in class.
// An unknown code shows the honest error ("No class found") — never a fake
// success. The mutation flows through the normal offline queue + consent gate.

import { useState } from 'react';
import { KeyRound, Loader2, Check } from 'lucide-react';
import toast from 'react-hot-toast';
import { fetchWithCache } from '@/lib/api';
import { useTranslations } from 'next-intl';

export default function JoinClassCard() {
  const t = useTranslations('joinClass');
  const [code, setCode] = useState('');
  const [joining, setJoining] = useState(false);
  const [joined, setJoined] = useState('');

  const join = async () => {
    const trimmed = code.trim().toUpperCase();
    if (trimmed.length < 4) {
      toast.error(t('enterCode'));
      return;
    }
    setJoining(true);
    try {
      const res = await fetchWithCache('/classes/join', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('log_token')}` },
        body: JSON.stringify({ code: trimmed }),
      });
      setJoined(res.name || trimmed);
      setCode('');
      toast.success(t('joined') + (res.name ? ` ${res.name}` : ''));
    } catch (e: unknown) {
      const detail = (e as { detail?: string })?.detail;
      toast.error(detail || t('notFound'));
    } finally {
      setJoining(false);
    }
  };

  return (
    <div className="card-glow border border-white/5 bg-black/20 rounded-2xl p-6 h-full">
      <div className="flex items-center gap-2 mb-4">
        <KeyRound className="w-4 h-4 text-brand-neon" />
        <h3 className="font-semibold text-white">{t('title')}</h3>
      </div>
      <p className="text-xs text-white/40 mb-3">{t('hint')}</p>
      {joined ? (
        <div className="flex items-center gap-2 text-sm text-brand-neon">
          <Check className="w-4 h-4" /> {t('inClass')} <span className="font-semibold">{joined}</span>
        </div>
      ) : (
        <div className="flex gap-2">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="ABC123"
            maxLength={10}
            className="flex-1 bg-white/5 border border-white/10 rounded-xl px-3 py-2.5 text-white font-mono tracking-[0.2em] uppercase placeholder-white/25 focus:border-brand-neon focus:outline-none"
            aria-label="Class invite code"
          />
          <button
            onClick={join}
            disabled={joining}
            className="bg-brand-neon/15 hover:bg-brand-neon/25 text-brand-neon px-4 py-2.5 rounded-xl font-semibold text-sm transition-colors disabled:opacity-50 inline-flex items-center gap-2"
          >
            {joining ? <Loader2 className="w-4 h-4 animate-spin" /> : t('join')}
          </button>
        </div>
      )}
    </div>
  );
}