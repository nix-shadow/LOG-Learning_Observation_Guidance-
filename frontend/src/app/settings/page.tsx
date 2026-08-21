"use client";

import { useState, useEffect, useRef } from 'react';
import { setManualOffline, clearApiCache, getStoragePersistence } from '@/lib/api';
import { wipeLocalData } from '@/lib/crypto';
import toast from 'react-hot-toast';
import { Wifi, WifiOff, Lock, Save, Loader2, Settings as SettingsIcon, LogOut, ShieldCheck, Download, Trash2, Bell, Type, Contrast, Moon, Sun } from 'lucide-react';
import ReminderToggle from '@/components/ReminderToggle';
import { useA11y } from '@/components/A11yProvider';
import { FONT_SCALE_LABELS } from '@/lib/a11y';
import { useTheme } from 'next-themes';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';

export default function SettingsPage() {
  const [mounted, setMounted] = useState(false);
  const [isOffline, setIsOffline] = useState(false);
  const [passwords, setPasswords] = useState({ old: '', new: '', confirm: '' });
  const [isUpdatingPassword, setIsUpdatingPassword] = useState(false);
  const [isRevokingAll, setIsRevokingAll] = useState(false);
  const [consentStatus, setConsentStatus] = useState<{ granted: boolean; version?: string }>({ granted: false });
  const [analyticsConsent, setAnalyticsConsent] = useState<boolean | null>(null);
  const [storageGrant, setStorageGrant] = useState<boolean | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState('');
  const [isDeleting, setIsDeleting] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const { prefs: a11y, setPrefs: setA11y } = useA11y();
  const { theme, setTheme } = useTheme();

  useEffect(() => {
    setMounted(true);
    setIsOffline(localStorage.getItem('log_offline_mode') === 'true');
    loadConsentStatus();
    // WP-0.1 research round: surface the browser's storage-persistence grant
    // honestly — null (checking) / false (evictable) / true (protected).
    const readGrant = () => setStorageGrant(getStoragePersistence());
    readGrant();
    const t = setInterval(readGrant, 1500);
    return () => clearInterval(t);
  }, []);

  // WP-0.1: reflect the recorded guardian consent state (null when absent —
  // never a fabricated value).
  const loadConsentStatus = async () => {
    try {
      const token = localStorage.getItem('log_token');
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/me/consent`, {
        headers: { 'Authorization': `Bearer ${token}` },
        cache: 'no-store',
      });
      if (!res.ok) return;
      const data = await res.json();
      const guardian = (data.consent || []).find(
        (c: { consent_type: string; status: string }) =>
          c.consent_type === 'guardian' && c.status === 'granted'
      );
      const analytics = (data.consent || []).find(
        (c: { consent_type: string; status: string }) =>
          c.consent_type === 'analytics' && c.status === 'granted'
      );
      setConsentStatus({ granted: !!guardian, version: guardian?.version || data.policy?.version });
      // WP-4.3: null = not loaded; false/true = the real recorded state.
      setAnalyticsConsent(analytics ? true : false);
    } catch {
      // Offline — leave the honest "not loaded" state.
    }
  };

  // WP-4.3: the analytics participation toggle. Granting records a
  // self-granted analytics consent; toggling off withdraws the same row
  // (the backend flips status, it never deletes the evidence).
  const handleAnalyticsToggle = async (enabled: boolean) => {
    const token = localStorage.getItem('log_token');
    const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/me/consent`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
      body: JSON.stringify({
        consent_type: 'analytics',
        version: '2026-08-v1',
        granted_by: 'self',
        language: 'en',
        source: 'settings',
        status: enabled ? 'granted' : 'withdrawn',
      }),
    });
    if (!res.ok) {
      toast.error('Could not update analytics preference. Please try again.');
      setAnalyticsConsent(!enabled);
      return;
    }
    setAnalyticsConsent(enabled);
    toast.success(enabled ? 'Analytics participation enabled.' : 'Analytics participation disabled.');
  };

  // WP-0.1: personal-data export (server-side envelope, never cached).
  const handleExport = async () => {
    setIsExporting(true);
    try {
      const token = localStorage.getItem('log_token');
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/me/export`, {
        headers: { 'Authorization': `Bearer ${token}` },
        cache: 'no-store',
      });
      if (!res.ok) {
        const error = await res.json().catch(() => ({}));
        throw new Error(error.error || 'Export failed');
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const disposition = res.headers.get('Content-Disposition') || '';
      const match = disposition.match(/filename="?([^";]+)"?/);
      a.download = match ? match[1] : `log-data-export-${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      toast.success('Your data export has been downloaded.');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsExporting(false);
    }
  };

  // WP-0.1: account erasure — type DELETE to confirm. The queue, cache, and
  // queue key are wiped locally so no plaintext residue survives.
  const handleDeleteAccount = async () => {
    if (deleteConfirm !== 'DELETE') {
      toast.error('Type DELETE to confirm account deletion.');
      return;
    }
    if (!window.confirm('This permanently deletes your account and all your learning data. This cannot be undone. Continue?')) return;
    setIsDeleting(true);
    try {
      const token = localStorage.getItem('log_token');
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/me`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ confirm: 'DELETE' }),
      });
      if (!res.ok) {
        const error = await res.json().catch(() => ({}));
        throw new Error(error.error || 'Failed to delete account');
      }
      await wipeLocalData();
      localStorage.removeItem('log_token');
      localStorage.removeItem('log_user');
      document.cookie = 'log_token=; path=/; max-age=0';
      toast.success('Account deleted. Thank you for learning with LOG.');
      window.location.href = '/login';
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
      setIsDeleting(false);
    }
  };

  useGSAP(() => {
    if (prefersReducedMotion()) return;
    if (mounted) {
      gsap.fromTo(".settings-header", 
        { opacity: 0, y: -20 },
        { opacity: 1, y: 0, duration: 0.6, ease: "power3.out" }
      );
      gsap.fromTo(".settings-card", 
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.6, stagger: 0.1, ease: "power3.out" }
      );
    }
  }, { dependencies: [mounted], scope: containerRef });

  const handleOfflineToggle = () => {
    const newState = !isOffline;
    setIsOffline(newState);
    setManualOffline(newState);
    toast.success(newState ? 'Offline mode simulated' : 'Back online');
  };

  const handlePasswordUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (passwords.new !== passwords.confirm) {
      toast.error("New passwords don't match");
      return;
    }
    if (passwords.new.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }

    setIsUpdatingPassword(true);
    try {
      const token = localStorage.getItem('log_token');
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/auth/password`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          old_password: passwords.old,
          new_password: passwords.new
        })
      });

      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error || 'Failed to update password');
      }

      toast.success("Password updated successfully");
      setPasswords({ old: '', new: '', confirm: '' });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsUpdatingPassword(false);
    }
  };

  const handleLogoutAll = async () => {
    if (!window.confirm('Log out on all devices? You will be signed out of this device too and must log in again.')) return;
    setIsRevokingAll(true);
    try {
      const token = localStorage.getItem('log_token');
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1'}/auth/logout-all`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error || 'Failed to log out all devices');
      }
      toast.success('Logged out on all devices');
      // F5: this device's API cache must go too — a later log-in as a
      // different account would otherwise read the revoked user's cached data.
      await clearApiCache();
      localStorage.removeItem('log_token');
      localStorage.removeItem('log_user');
      document.cookie = 'log_token=; path=/; max-age=0';
      window.location.href = '/login';
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsRevokingAll(false);
    }
  };

  if (!mounted) return null;

  return (
    <div ref={containerRef} className="flex-1 w-full max-w-4xl mx-auto space-y-8 pb-12 pt-4">
      {/* Header */}
      <div className="settings-header flex items-center justify-between">
        <div>
          <h1 className="text-4xl font-bold text-white tracking-tight">
            Settings
          </h1>
          <p className="text-white/60 mt-2 text-lg">
            Manage your preferences and security
          </p>
        </div>
        <div className="p-4 bg-white/5 rounded-2xl backdrop-blur-xl border border-white/10">
          <SettingsIcon className="w-8 h-8 text-brand-neon animate-spin-slow" />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Network Card */}
        <div className="settings-card card-glow bg-black/40 rounded-3xl backdrop-blur-3xl border border-white/10 p-6 relative overflow-hidden group hover:border-brand-teal/50 transition-all duration-300">
          <div className="absolute inset-0 bg-brand-neon/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center gap-2">
            <Wifi className="w-5 h-5 text-brand-teal" />
            Network
          </h2>
          <div className="flex-1 flex items-center justify-between relative z-10">
            <div>
              <p className="font-bold text-white">Offline Simulation</p>
              <p className="text-sm text-white/60">Force app into offline mode</p>
            </div>
            <button
              onClick={handleOfflineToggle}
              className={`p-3 rounded-xl border transition-all duration-300 ${isOffline ? 'bg-brand-neon/20 border-brand-neon text-brand-neon' : 'bg-white/5 border-white/10 hover:bg-white/10 text-white/60 hover:text-white'}`}
            >
              {isOffline ? <WifiOff className="w-5 h-5" /> : <Wifi className="w-5 h-5" />}
            </button>
          </div>
        </div>

        {/* Reminders Card (WP-1.4) */}
        <div className="settings-card card-glow bg-black/40 rounded-3xl backdrop-blur-3xl border border-white/10 p-6 relative overflow-hidden group hover:border-brand-neon/50 transition-all duration-300">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center gap-2">
            <Bell className="w-5 h-5 text-brand-neon" />
            Daily Reminders
          </h2>
          <ReminderToggle />
        </div>

        {/* Appearance Card — dark/light theme, saved on this device */}
        <div className="settings-card card-glow bg-black/40 rounded-3xl backdrop-blur-3xl border border-white/10 p-6 relative overflow-hidden group hover:border-brand-teal/50 transition-all duration-300">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center gap-2">
            <Moon className="w-5 h-5 text-brand-teal" />
            Appearance
          </h2>
          <div className="space-y-6 relative z-10">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-bold text-white flex items-center gap-2">
                  {mounted && theme === 'dark' ? <Moon className="w-4 h-4 text-brand-teal" /> : <Sun className="w-4 h-4 text-brand-amber" />}
                  Dark Mode
                </p>
                <p className="text-sm text-white/60">
                  Dark is the default. Switch to light for bright classrooms — saved on this device, works offline.
                </p>
              </div>
              <button
                role="switch"
                aria-checked={mounted ? theme === 'dark' : true}
                aria-label="Toggle dark mode"
                onClick={() => setTheme(mounted && theme === 'dark' ? 'light' : 'dark')}
                className={`p-3 rounded-xl border transition-all duration-300 ${!mounted || theme === 'dark' ? 'bg-brand-teal/20 border-brand-teal text-brand-teal' : 'bg-white/5 border-white/10 hover:bg-white/10 text-white/60 hover:text-white'}`}
              >
                {!mounted || theme === 'dark' ? <Moon className="w-5 h-5" /> : <Sun className="w-5 h-5" />}
              </button>
            </div>
          </div>
        </div>

        {/* Accessibility Card (WP-3.4 RC-12) */}
        <div className="settings-card card-glow bg-black/40 rounded-3xl backdrop-blur-3xl border border-white/10 p-6 relative overflow-hidden group hover:border-brand-amber/50 transition-all duration-300">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center gap-2">
            <Contrast className="w-5 h-5 text-brand-amber" />
            Accessibility
          </h2>
          <div className="space-y-6 relative z-10">
            <div>
              <p className="font-bold text-white mb-3 flex items-center gap-2">
                <Type className="w-4 h-4 text-brand-amber" />
                Text Size
              </p>
              <div className="flex gap-2">
                {FONT_SCALE_LABELS.map((scale) => (
                  <button
                    key={scale}
                    aria-pressed={a11y.fontScale === scale}
                    onClick={() => setA11y({ ...a11y, fontScale: scale })}
                    className={`px-4 py-2.5 rounded-xl border text-sm font-bold transition-all ${
                      a11y.fontScale === scale
                        ? 'bg-brand-amber/20 border-brand-amber text-brand-amber'
                        : 'bg-white/5 border-white/10 text-white/60 hover:bg-white/10 hover:text-white'
                    }`}
                    style={{ fontSize: scale === 'normal' ? 13 : scale === 'large' ? 15 : 17 }}
                  >
                    {scale === 'normal' ? 'A' : scale === 'large' ? 'A' : 'A'}
                  </button>
                ))}
              </div>
              <p className="text-xs text-brand-muted mt-2">
                Saved on this device and applied everywhere, even offline.
              </p>
            </div>
            <div className="flex items-center justify-between border-t border-white/10 pt-5">
              <div>
                <p className="font-bold text-white">High Contrast</p>
                <p className="text-sm text-white/60">
                  Stronger text and borders for low-light classrooms and low-vision screens.
                </p>
              </div>
              <button
                role="switch"
                aria-checked={a11y.highContrast}
                onClick={() => setA11y({ ...a11y, highContrast: !a11y.highContrast })}
                className={`p-3 rounded-xl border transition-all duration-300 ${a11y.highContrast ? 'bg-brand-amber/20 border-brand-amber text-brand-amber' : 'bg-white/5 border-white/10 hover:bg-white/10 text-white/60 hover:text-white'}`}
              >
                <Contrast className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>

        {/* Security Card */}
        <div className="settings-card md:col-span-2 card-glow bg-black/40 rounded-3xl backdrop-blur-3xl border border-white/10 p-6 relative overflow-hidden group hover:border-brand-blue/50 transition-all duration-300">
          <div className="absolute inset-0 bg-brand-blue/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center gap-2">
            <Lock className="w-5 h-5 text-brand-blue" />
            Security
          </h2>
          
          <form onSubmit={handlePasswordUpdate} className="space-y-5 max-w-md relative z-10">
            <div>
              <label className="block text-sm font-bold text-white/80 mb-2 uppercase tracking-wider">Current Password</label>
              <input
                type="password"
                required
                value={passwords.old}
                onChange={e => setPasswords({...passwords, old: e.target.value})}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-brand-neon/50 transition-all placeholder-white/30"
                placeholder="Enter current password"
              />
            </div>
            <div>
              <label className="block text-sm font-bold text-white/80 mb-2 uppercase tracking-wider">New Password</label>
              <input
                type="password"
                required
                minLength={8}
                value={passwords.new}
                onChange={e => setPasswords({...passwords, new: e.target.value})}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-brand-neon/50 transition-all placeholder-white/30"
                placeholder="Minimum 8 characters"
              />
            </div>
            <div>
              <label className="block text-sm font-bold text-white/80 mb-2 uppercase tracking-wider">Confirm New Password</label>
              <input
                type="password"
                required
                minLength={8}
                value={passwords.confirm}
                onChange={e => setPasswords({...passwords, confirm: e.target.value})}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-brand-neon/50 transition-all placeholder-white/30"
                placeholder="Repeat new password"
              />
            </div>
            <div className="pt-4">
              <button
                type="submit"
                disabled={isUpdatingPassword}
                className="btn-primary w-full md:w-auto px-8 py-3 text-lg flex items-center justify-center font-bold tracking-wide"
              >
                {isUpdatingPassword ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <Save className="w-5 h-5 mr-2" />
                    Update Password
                  </>
                )}
              </button>
            </div>
          </form>

          <div className="border-t border-white/10 pt-6 mt-6 relative z-10">
            <h3 className="text-lg font-bold text-white mb-2 flex items-center gap-2">
              <LogOut className="w-5 h-5 text-red-400" />
              Active Sessions
            </h3>
            <p className="text-sm text-white/60 mb-4">
              If your device was lost or stolen, sign out of every device at once.
            </p>
            <button
              onClick={handleLogoutAll}
              disabled={isRevokingAll}
              className="bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 text-red-400 hover:text-red-300 px-6 py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition-all disabled:opacity-50"
            >
              {isRevokingAll ? <Loader2 className="w-5 h-5 animate-spin" /> : <LogOut className="w-5 h-5" />}
              Log Out on All Devices
            </button>
          </div>
        </div>

        {/* Privacy & Data Card (WP-0.1) */}
        <div className="settings-card md:col-span-2 card-glow bg-black/40 rounded-3xl backdrop-blur-3xl border border-white/10 p-6 relative overflow-hidden group hover:border-brand-teal/50 transition-all duration-300">
          <div className="absolute inset-0 bg-brand-teal/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-brand-teal" />
            Privacy &amp; Data
          </h2>

          <div className="relative z-10 space-y-6">
            {/* Consent status */}
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="font-bold text-white">Guardian Consent</p>
                <p className="text-sm text-white/60">
                  {consentStatus.granted
                    ? `Recorded on your account (policy ${consentStatus.version || '2026-08-v1'}).`
                    : 'Not recorded yet. You can grant it at registration.'}
                </p>
                <p className="text-xs text-brand-muted mt-1 leading-relaxed">
                  अभिभावकको सहमति — LOG ले विद्यार्थी डेटा तेस्रो पक्षसँग साझा
                  गर्दैन। सिकाइ डेटा २ वर्षसम्म राखिन्छ। तपाईं जुनसुकै बेला डेटा
                  निर्यात गर्न वा खाता मेटाउन सक्नुहुन्छ।
                </p>
              </div>
              <span
                className={`px-4 py-1.5 rounded-full text-xs font-bold border ${
                  consentStatus.granted
                    ? 'bg-brand-teal/10 border-brand-teal/40 text-brand-teal'
                    : 'bg-white/5 border-white/10 text-brand-muted'
                }`}
              >
                {consentStatus.granted ? 'Granted' : 'Pending'}
              </span>
            </div>

            {/* Analytics participation (WP-4.3) — aggregate-only, opt-in.
                Null = not loaded yet (honest three-state, never a guess). */}
            <div className="flex items-center justify-between gap-4 border-t border-white/10 pt-5">
              <div>
                <p className="font-bold text-white">Analytics Participation</p>
                <p className="text-sm text-white/60">
                  When on, anonymous usage totals (number of activities completed,
                  average scores) may be included in school-level reports. This is
                  aggregate data only — your individual progress is never shown.
                </p>
              </div>
              <button
                onClick={() => handleAnalyticsToggle(!analyticsConsent)}
                disabled={analyticsConsent === null}
                aria-checked={analyticsConsent === true}
                role="switch"
                className={`shrink-0 w-14 h-8 rounded-full p-1 transition-all disabled:opacity-40 ${
                  analyticsConsent === true
                    ? 'bg-brand-teal'
                    : 'bg-white/10'
                }`}
              >
                <span
                  className={`block w-6 h-6 rounded-full bg-white transition-transform ${
                    analyticsConsent === true ? 'translate-x-6' : ''
                  }`}
                />
              </button>
            </div>

            {/* Offline storage persistence (WP-0.1 research round) — honest
                three-state view of the browser's eviction exemption grant. */}
            <div className="flex items-start justify-between gap-4 border-t border-white/10 pt-5">
              <div>
                <p className="font-bold text-white">Offline Storage Protection</p>
                <p className="text-sm text-white/60">
                  {storageGrant === true
                    ? 'Your browser confirmed offline data is protected from automatic eviction.'
                    : storageGrant === false
                      ? 'Your browser did not grant permanent storage — offline work could be evicted under storage pressure. Reconnect regularly to sync.'
                      : 'Checking with your browser...'}
                </p>
              </div>
              <span
                className={`shrink-0 px-4 py-1.5 rounded-full text-xs font-bold border ${
                  storageGrant === true
                    ? 'bg-brand-teal/10 border-brand-teal/40 text-brand-teal'
                    : storageGrant === false
                      ? 'bg-amber-500/10 border-amber-500/40 text-amber-400'
                      : 'bg-white/5 border-white/10 text-brand-muted'
                }`}
              >
                {storageGrant === true ? 'Protected' : storageGrant === false ? 'Not granted' : 'Checking'}
              </span>
            </div>

            {/* Export */}
            <div className="flex items-center justify-between gap-4 border-t border-white/10 pt-5">
              <div>
                <p className="font-bold text-white">Download My Data</p>
                <p className="text-sm text-white/60">
                  A complete, portable copy of everything LOG stores about you
                  (JSON, self-describing envelope).
                </p>
              </div>
              <button
                onClick={handleExport}
                disabled={isExporting}
                className="bg-brand-teal/10 hover:bg-brand-teal/20 border border-brand-teal/40 text-brand-teal hover:text-brand-teal px-5 py-2.5 rounded-xl font-bold flex items-center gap-2 transition-all disabled:opacity-50 shrink-0"
              >
                {isExporting ? <Loader2 className="w-5 h-5 animate-spin" /> : <Download className="w-5 h-5" />}
                {isExporting ? 'Preparing...' : 'Export'}
              </button>
            </div>

            {/* Erasure */}
            <div className="border-t border-white/10 pt-5">
              <p className="font-bold text-white flex items-center gap-2">
                <Trash2 className="w-5 h-5 text-red-400" />
                Delete Account
              </p>
              <p className="text-sm text-white/60 mt-1">
                Permanently erases your learning data (progress, activities,
                observations, guidance) and closes the account. Anonymized audit
                records may remain up to 3 years. This cannot be undone.
              </p>
              <div className="flex flex-wrap items-center gap-3 mt-3">
                <input
                  type="text"
                  value={deleteConfirm}
                  onChange={(e) => setDeleteConfirm(e.target.value)}
                  placeholder='Type DELETE to confirm'
                  className="w-full sm:w-56 px-4 py-2.5 bg-black/50 border border-white/10 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-red-500/50 transition-all placeholder-white/30 text-sm"
                />
                <button
                  onClick={handleDeleteAccount}
                  disabled={isDeleting}
                  className="bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 text-red-400 hover:text-red-300 px-5 py-2.5 rounded-xl font-bold flex items-center gap-2 transition-all disabled:opacity-50"
                >
                  {isDeleting ? <Loader2 className="w-5 h-5 animate-spin" /> : <Trash2 className="w-5 h-5" />}
                  {isDeleting ? 'Deleting...' : 'Delete my account'}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
