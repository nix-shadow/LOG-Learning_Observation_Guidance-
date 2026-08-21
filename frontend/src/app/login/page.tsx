"use client";

import { useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { fetchWithCache, flushSyncQueue } from '@/lib/api';
import { disclosureHash } from '@/lib/crypto';
import toast from 'react-hot-toast';
import { m as motion } from 'framer-motion';
import Image from 'next/image';
import { GoogleLogin, GoogleOAuthProvider, CredentialResponse } from '@react-oauth/google';
import { Eye, EyeOff } from 'lucide-react';
import ThemeToggle from '@/components/ThemeToggle';

export default function Login() {
  const [activeTab, setActiveTab] = useState<'login' | 'register' | 'parent'>('login');
  
  // Form states
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  
  // WP-2.1: parent claim form (teacher-issued invite code)
  const [parentName, setParentName] = useState('');
  const [parentEmail, setParentEmail] = useState('');
  const [parentPassword, setParentPassword] = useState('');
  const [inviteCode, setInviteCode] = useState('');
  const [parentConsentChecked, setParentConsentChecked] = useState(false);
  
  const [loading, setLoading] = useState(false);
  const [consentChecked, setConsentChecked] = useState(false);
  const [guardianName, setGuardianName] = useState('');
  const { login } = useAuth();

  // WP-0.1: the EXACT bilingual notice text presented at consent time. The
  // rendered checkbox text below MUST stay identical to these constants — the
  // server stores their sha256 as disclosure_hash, the evidence that the
  // guardian saw precisely this text (COPPA 16 CFR §312.5 practice).
  // Research round: the notice now states the retention schedule and the
  // explicit no-disclosure commitment so consent is about the real policy,
  // not a vague "secure storage" promise.
  const CONSENT_NOTICE_EN =
    'I confirm that my guardian understands and agrees to the collection, use, and secure storage of my learning progress for educational purposes, per LOG\'s privacy policy (version 2026-08-v1). LOG never discloses learner data to third parties. Data is kept at most 2 years after last activity; audit records at most 3 years; I can export or delete my data at any time.';
  const CONSENT_NOTICE_NP =
    'मेरा अभिभावकले मेरो सिकाइ प्रगति डेटा शैक्षिक उद्देश्यका लागि सङ्कलन, प्रयोग र सुरक्षित भण्डारण गरिने कुरा बुझेर सहमति दिनुभएको छ भनी पुष्टि गर्दछु। LOG ले विद्यार्थी डेटा तेस्रो पक्षसँग कहिल्यै साझा गर्दैन। डेटा अन्तिम गतिविधि भएको २ वर्षसम्म मात्र राखिनेछ; अडिट रेकर्ड बढीमा ३ वर्ष; तपाईं कुनै पनि समय डेटा निर्यात वा खाता मेटाउन सक्नुहुन्छ।';
  const CONSENT_NOTICE = `Guardian Consent · अभिभावकको सहमति\n${CONSENT_NOTICE_EN}\n${CONSENT_NOTICE_NP}`;

  // WP-2.1: the EXACT parent-portal notice presented at claim time. Its
  // sha256 travels as disclosure_hash so the school can prove the guardian
  // saw precisely this text before accessing their child's digest.
  const PARENT_NOTICE_EN =
    'I confirm that I am the guardian of a learner at this school and agree to view my child\'s learning progress through the LOG parent portal, per LOG\'s privacy policy (version 2026-08-v1). The portal is read-only: progress and guidance only, never contact details or teacher observations. LOG never discloses learner data to third parties. Data is kept at most 2 years after last activity; I can export or delete my child\'s data at any time.';
  const PARENT_NOTICE_NP =
    'म यस विद्यालयका विद्यार्थीको अभिभावक हुँ भनी पुष्टि गर्दछु र LOG अभिभावक पोर्टलमार्फत आफ्नो छोराछोरीको सिकाइ प्रगति हेर्न सहमत छु। पोर्टल केवल पढ्नका लागि हो: प्रगति र मार्गदर्शन मात्र — सम्पर्क विवरण वा शिक्षकका टिप्पणीहरू होइनन्। LOG ले विद्यार्थी डेटा तेस्रो पक्षसँग कहिल्यै साझा गर्दैन। डेटा अन्तिम गतिविधि भएको २ वर्षसम्म मात्र राखिनेछ; तपाईं कुनै पनि समय डेटा निर्यात वा मेटाउन सक्नुहुन्छ।';
  const PARENT_NOTICE = `Parent Portal Consent · अभिभावक पोर्टल सहमति\n${PARENT_NOTICE_EN}\n${PARENT_NOTICE_NP}`;

  // WP-0.1: guardian consent is recorded as evidence after every successful
  // registration path (register + Google). Offline, the POST is queued like
  // any other mutation and syncs later — consent is never silently dropped.
  const submitConsent = async (source: string) => {
    const body = {
      consent_type: 'guardian',
      version: '2026-08-v1',
      granted_by: 'guardian',
      guardian_name: guardianName.trim() || 'Account holder',
      guardian_contact: '',
      language: 'ne',
      source,
      disclosure_hash: await disclosureHash(CONSENT_NOTICE),
    };
    fetchWithCache('/me/consent', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
      .then((res) => {
        if (res && (res as { consent_type?: string }).consent_type) {
          toast.success('Guardian consent recorded.', { icon: '✅' });
        }
      })
      .catch(() => {}); // Non-blocking; consent retries with the sync queue.
  };

  // F1: a fresh token can replay the offline queue that an expired session
  // could not (replays attach the current token at flush time).
  const afterAuth = () => {
    flushSyncQueue().then(({ synced }) => {
      if (synced > 0) toast.success(`${synced} offline action(s) synced.`, { icon: '🔄' });
    }).catch(() => {});
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const res = await fetchWithCache('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });
      if (res.token) {
        toast.success('Logged in successfully!');
        login(res.user, res.token);
        afterAuth();
      } else {
        toast.error(res.error || 'Login failed. Please try again.');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Login failed. Please check your credentials.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!consentChecked) {
      toast.error('Please accept the guardian consent to continue.');
      return;
    }
    if (password !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    setLoading(true);
    try {
      const res = await fetchWithCache('/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email, password })
      });
      if (res.token) {
        toast.success('Account created! Welcome aboard 🎉');
        login(res.user, res.token);
        submitConsent('register');
        afterAuth();
      } else if (res.message) {
        toast.success('Account created! Please sign in.');
        setActiveTab('login');
      } else {
        toast.error(res.error || 'Registration failed. Please try again.');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Registration failed. Please try again.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  // WP-2.1: parent claim — one atomic flow (create PARENT account + claim the
  // teacher-issued invite + record the parent_access consent with the notice's
  // disclosure hash). The returned token carries the PARENT role, so
  // AuthContext lands on /parent.
  const handleParentClaim = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!parentConsentChecked) {
      toast.error('Please accept the parent portal consent to continue.');
      return;
    }
    setLoading(true);
    try {
      const res = await fetchWithCache('/auth/parent-signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: parentName,
          email: parentEmail,
          password: parentPassword,
          invite_code: inviteCode.trim(),
          disclosure_hash: await disclosureHash(PARENT_NOTICE),
          language: 'ne',
        })
      });
      if (res.token) {
        toast.success('Parent account created — your child is linked!');
        login(res.user, res.token);
        afterAuth();
      } else {
        toast.error(res.error || res.detail || 'Could not claim the invite. Ask the teacher to check the code.');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Could not claim the invite. Ask the teacher to check the code.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleGoogleSuccess = async (credentialResponse: CredentialResponse) => {
    if (!consentChecked) {
      toast.error('Please accept the guardian consent to continue.');
      return;
    }
    setLoading(true);
    try {
      const res = await fetchWithCache('/auth/google', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: credentialResponse.credential })
      });
      if (res.token) {
        toast.success('Google Login successful!');
        login(res.user, res.token);
        submitConsent('google');
        afterAuth();
      } else {
        toast.error('Google Login failed.');
      }
    } catch {
      toast.error('Google Login failed.');
    }
    setLoading(false);
  };

  const handleGoogleError = () => {
    toast.error('Google Login was unsuccessful.');
  };

  const inputClasses = "w-full px-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-white/30 focus:ring-2 focus:ring-brand-teal focus:border-transparent outline-none transition-all shadow-inner";

  return (
    <div className="min-h-[80vh] flex items-center justify-center relative">
      <div className="absolute top-4 right-4 z-50">
        <ThemeToggle />
      </div>
      
      {/* Background glow behind card */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[400px] h-[400px] bg-brand-teal/20 rounded-full blur-[100px] pointer-events-none -z-10"></div>
      
      <motion.div
        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, ease: "easeOut" }}
        className="card-glow bg-black/40 backdrop-blur-2xl border border-white/10 w-full max-w-md p-8 shadow-bento"
      >
        <div className="text-center mb-8">
          <Image src="/assets/log-logo.png" alt="LOG Logo" width={150} height={60} className="mx-auto mb-6 dark:invert" />
          <h2 className="text-3xl font-bold text-white tracking-tight">Welcome</h2>
          <p className="text-white/60 mt-2">Sign in or create an account to continue.</p>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-white/10 mb-8">
          <button
            className={`flex-1 pb-3 text-center text-sm font-medium transition-colors ${
              activeTab === 'login' 
                ? 'text-brand-neon border-b-2 border-brand-neon' 
                : 'text-brand-muted hover:text-white/80'
            }`}
            onClick={() => setActiveTab('login')}
          >
            Login
          </button>
          <button
            className={`flex-1 pb-3 text-center text-sm font-medium transition-colors ${
              activeTab === 'register' 
                ? 'text-brand-neon border-b-2 border-brand-neon' 
                : 'text-brand-muted hover:text-white/80'
            }`}
            onClick={() => setActiveTab('register')}
          >
            Register
          </button>
          <button
            className={`flex-1 pb-3 text-center text-sm font-medium transition-colors ${
              activeTab === 'parent' 
                ? 'text-brand-neon border-b-2 border-brand-neon' 
                : 'text-brand-muted hover:text-white/80'
            }`}
            onClick={() => setActiveTab('parent')}
          >
            Parent · अभिभावक
          </button>
        </div>

        {activeTab === 'login' ? (
          <form onSubmit={handleLogin} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Email</label>
              <input
                type="email" required
                value={email} onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className={inputClasses}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Password</label>
              <div className="relative">
                <input
                  type={showPassword ? "text" : "password"} required
                  value={password} onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className={`${inputClasses} pr-12`}
                />
                <button 
                  type="button" 
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-brand-muted hover:text-white/80 focus:outline-none transition-colors"
                >
                  {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                </button>
              </div>
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full py-3.5 text-lg mt-4">
              {loading ? 'Logging in...' : 'Login'}
            </button>
          </form>
        ) : activeTab === 'register' ? (
          <form onSubmit={handleRegister} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Full Name</label>
              <input
                type="text" required
                value={name} onChange={(e) => setName(e.target.value)}
                placeholder="Jane Doe"
                className={inputClasses}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Email</label>
              <input
                type="email" required
                value={email} onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className={inputClasses}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Password</label>
              <div className="relative">
                <input
                  type={showPassword ? "text" : "password"} required minLength={6}
                  value={password} onChange={(e) => setPassword(e.target.value)}
                  placeholder="•••••••• (Min 6 chars)"
                  className={`${inputClasses} pr-12`}
                />
                <button 
                  type="button" 
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-brand-muted hover:text-white/80 focus:outline-none transition-colors"
                >
                  {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                </button>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Confirm Password</label>
              <div className="relative">
                <input
                  type={showConfirmPassword ? "text" : "password"} required minLength={6}
                  value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="••••••••"
                  className={`${inputClasses} pr-12`}
                />
                <button 
                  type="button" 
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-brand-muted hover:text-white/80 focus:outline-none transition-colors"
                >
                  {showConfirmPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                </button>
              </div>
            </div>
            {/* WP-0.1: guardian consent — required for learners under 18.
                Bilingual notice; the evidence record is stored server-side. */}
            <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
              <label className="flex items-start gap-3 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={consentChecked}
                  onChange={(e) => setConsentChecked(e.target.checked)}
                  className="mt-1 w-4 h-4 accent-brand-teal"
                />
                <span className="text-xs text-white/70 leading-relaxed">
                  <strong className="text-white/90">Guardian Consent · अभिभावकको सहमति</strong>
                  <br />
                  {CONSENT_NOTICE_EN}
                  <br />
                  {/* WP-0.3 a11y research round: Devanagari text must be
                      announced with the right language profile. */}
                  <span className="text-white/50" lang="ne">{CONSENT_NOTICE_NP}</span>
                </span>
              </label>
              <input
                type="text"
                value={guardianName}
                onChange={(e) => setGuardianName(e.target.value)}
                placeholder="Guardian's name (optional) · अभिभावकको नाम (वैकल्पिक)"
                lang="ne"
                className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white placeholder-white/30 focus:ring-2 focus:ring-brand-teal focus:border-transparent outline-none transition-all text-sm"
              />
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full py-3.5 text-lg mt-4">
              {loading ? 'Registering...' : 'Register'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleParentClaim} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Parent Name</label>
              <input
                type="text" required
                value={parentName} onChange={(e) => setParentName(e.target.value)}
                placeholder="Jane Doe"
                className={inputClasses}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Email</label>
              <input
                type="email" required
                value={parentEmail} onChange={(e) => setParentEmail(e.target.value)}
                placeholder="you@example.com"
                className={inputClasses}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">Password</label>
              <input
                type="password" required minLength={8}
                value={parentPassword} onChange={(e) => setParentPassword(e.target.value)}
                placeholder="•••••••• (Min 8 chars)"
                className={inputClasses}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-white/80 mb-2">
                Invite code <span className="text-brand-muted">(from your child&apos;s teacher)</span>
              </label>
              <input
                type="text" required
                value={inviteCode} onChange={(e) => setInviteCode(e.target.value)}
                placeholder="e.g. A1B2C3"
                className={`${inputClasses} font-mono tracking-widest uppercase`}
              />
            </div>
            {/* WP-2.1: parent portal consent — read-only digest access,
                evidence recorded server-side as disclosure_hash. */}
            <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
              <label className="flex items-start gap-3 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={parentConsentChecked}
                  onChange={(e) => setParentConsentChecked(e.target.checked)}
                  className="mt-1 w-4 h-4 accent-brand-teal"
                />
                <span className="text-xs text-white/70 leading-relaxed">
                  <strong className="text-white/90">Parent Portal Consent · अभिभावक पोर्टल सहमति</strong>
                  <br />
                  {PARENT_NOTICE_EN}
                  <br />
                  <span className="text-white/50" lang="ne">{PARENT_NOTICE_NP}</span>
                </span>
              </label>
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full py-3.5 text-lg mt-4">
              {loading ? 'Creating account...' : 'Claim my child'}
            </button>
          </form>
        )}

        <div className="mt-8 flex items-center justify-center space-x-4">
          <div className="h-px bg-white/10 w-full"></div>
          <span className="text-brand-muted text-sm font-medium">OR</span>
          <div className="h-px bg-white/10 w-full"></div>
        </div>

        <div className="mt-6 flex justify-center">
          {/* WP-0.3 bundle research round: GoogleOAuthProvider was mounted in
              the root layout, pulling @react-oauth/google into every page.
              It only serves this one button — scoped here, the script only
              loads on the login page. */}
          <GoogleOAuthProvider clientId={process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || 'YOUR_GOOGLE_CLIENT_ID_HERE'}>
            <GoogleLogin
              onSuccess={handleGoogleSuccess}
              onError={handleGoogleError}
              theme="filled_black"
              shape="pill"
            />
          </GoogleOAuthProvider>
        </div>
        <p className="mt-3 text-center text-xs text-brand-muted">
          {activeTab === 'register' && !consentChecked
            ? 'Guardian consent is required before creating an account or signing in with Google.'
            : 'Continue with Google — guardian consent required for new accounts.'}
        </p>
      </motion.div>
    </div>
  );
}
