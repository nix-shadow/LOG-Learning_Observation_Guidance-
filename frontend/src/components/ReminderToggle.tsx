"use client";

// WP-1.4: opt-in daily reminder. Web Notifications are ONLY requested after
// explicit user permission (never on load). Honest limitation: without a push
// server (no VAPID keys in this school-LAN deployment) notifications fire
// while this tab is open. The UI says so — no fake "push enabled" claims.

import { useEffect, useState } from 'react';
import { Bell, BellOff, Check } from 'lucide-react';
import { useTranslations } from 'next-intl';

const REMINDER_TIME_KEY = 'log_reminder_time';
const REMINDER_FIRED_KEY = 'log_reminder_fired_on';

export default function ReminderToggle() {
  const t = useTranslations('reminder');
  const [permission, setPermission] = useState<NotificationPermission | 'unsupported'>(
    typeof window !== 'undefined' && 'Notification' in window
      ? Notification.permission
      : 'unsupported'
  );
  const [reminderTime, setReminderTime] = useState(() => {
    if (typeof window === 'undefined') return '18:00';
    return localStorage.getItem(REMINDER_TIME_KEY) ?? '18:00';
  });

  const askPermission = async () => {
    if (typeof window === 'undefined' || !('Notification' in window)) return;
    const result = await Notification.requestPermission();
    setPermission(result);
  };

  const disable = () => {
    setPermission('denied');
    localStorage.removeItem(REMINDER_FIRED_KEY);
  };

  useEffect(() => {
    if (permission !== 'granted') return;
    const timer = window.setInterval(() => {
      const now = new Date();
      const hhmm = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
      if (hhmm !== reminderTime) return;
      const today = now.toISOString().slice(0, 10);
      if (localStorage.getItem(REMINDER_FIRED_KEY) === today) return;
      localStorage.setItem(REMINDER_FIRED_KEY, today);
      try {
        new Notification(t('title'), { body: t('body') });
      } catch {
        // Notification constructor can throw on some engines — a missed
        // reminder is never worth an error toast.
      }
    }, 60_000);
    return () => window.clearInterval(timer);
  }, [permission, reminderTime, t]);

  if (permission === 'unsupported') {
    return (
      <p className="text-sm text-white/40 flex items-center gap-2">
        <BellOff className="w-4 h-4" /> {t('unsupported')}
      </p>
    );
  }

  if (permission !== 'granted') {
    return (
      <div className="space-y-3">
        <button onClick={askPermission} className="btn-primary px-5 py-2.5 text-sm font-bold inline-flex items-center gap-2">
          <Bell className="w-4 h-4" /> {permission === 'denied' ? t('reEnable') : t('enable')}
        </button>
        <p className="text-xs text-white/40">{t('optInNote')}</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <span className="p-2 rounded-full bg-brand-dark text-brand-neon">
          <Check className="w-4 h-4" />
        </span>
        <label className="text-sm text-white/80 flex items-center gap-2">
          {t('dailyAt')}
          <input
            type="time"
            value={reminderTime}
            onChange={(e) => {
              setReminderTime(e.target.value);
              localStorage.setItem(REMINDER_TIME_KEY, e.target.value);
              localStorage.removeItem(REMINDER_FIRED_KEY);
            }}
            className="bg-white/5 border border-white/10 rounded-lg px-2 py-1 text-white text-sm"
          />
        </label>
        <button onClick={disable} className="text-xs text-white/40 hover:text-white/70 underline">
          {t('disable')}
        </button>
      </div>
      <p className="text-xs text-white/40">{t('honestLimit')}</p>
    </div>
  );
}