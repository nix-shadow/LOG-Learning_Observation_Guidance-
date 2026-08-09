"use client";

import { useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { fetchWithCache } from '@/lib/api';
import toast from 'react-hot-toast';
import { motion } from 'framer-motion';
import Link from 'next/link';
import Image from 'next/image';

export default function Login() {
  const [step, setStep] = useState<'phone' | 'otp'>('phone');
  const [phone, setPhone] = useState('');
  const [otp, setOtp] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();

  const handleRequestOTP = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await fetchWithCache('/auth/request-otp', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone })
      });
      toast.success('OTP sent! Check your messages.');
      setStep('otp');
    } catch {
      toast.error('Failed to send OTP. Are you offline?');
    }
    setLoading(false);
  };

  const handleVerifyOTP = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const res = await fetchWithCache('/auth/verify-otp', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone, otp })
      });
      if (res.token) {
        toast.success('Logged in successfully!');
        login(res.user, res.token);
      } else {
        toast.error('Invalid OTP');
      }
    } catch {
      toast.error('Login failed.');
    }
    setLoading(false);
  };

  const handleGoogleLogin = async () => {
    // Mock Google Login flow
    setLoading(true);
    try {
      const res = await fetchWithCache('/auth/google', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'learner@gmail.com', name: 'Google Learner' })
      });
      if (res.token) {
        toast.success('Google Login successful!');
        login(res.user, res.token);
      }
    } catch {
      toast.error('Google Login failed.');
    }
    setLoading(false);
  };

  return (
    <div className="min-h-[80vh] flex items-center justify-center">
      <motion.div
        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }}
        className="card w-full max-w-md p-8 shadow-2xl"
      >
        <div className="text-center mb-8">
          <Image src="/assets/log-logo.png" alt="LOG Logo" width={150} height={60} className="mx-auto mb-4" />
          <h2 className="text-2xl font-bold text-brand-blue">Welcome Back</h2>
          <p className="text-gray-500">Sign in to continue your learning journey.</p>
        </div>

        {step === 'phone' ? (
          <form onSubmit={handleRequestOTP} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Phone Number</label>
              <input
                type="tel" required
                value={phone} onChange={(e) => setPhone(e.target.value)}
                placeholder="+977 9800000000"
                className="w-full px-4 py-3 rounded-xl border border-gray-300 focus:ring-2 focus:ring-brand-teal focus:border-transparent outline-none transition-all"
              />
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full py-3 text-lg mt-2">
              {loading ? 'Sending...' : 'Send OTP'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleVerifyOTP} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Enter 6-digit OTP</label>
              <input
                type="text" required maxLength={6}
                value={otp} onChange={(e) => setOtp(e.target.value)}
                placeholder="123456"
                className="w-full px-4 py-3 rounded-xl border border-gray-300 focus:ring-2 focus:ring-brand-teal focus:border-transparent outline-none transition-all text-center tracking-widest text-lg"
              />
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full py-3 text-lg mt-2">
              {loading ? 'Verifying...' : 'Verify & Login'}
            </button>
            <button type="button" onClick={() => setStep('phone')} className="w-full text-sm text-gray-500 hover:text-brand-blue mt-2">
              Use a different number
            </button>
          </form>
        )}

        <div className="mt-8 flex items-center justify-center space-x-4">
          <div className="h-px bg-gray-200 w-full"></div>
          <span className="text-gray-400 text-sm">OR</span>
          <div className="h-px bg-gray-200 w-full"></div>
        </div>

        <button
          onClick={handleGoogleLogin}
          disabled={loading}
          className="w-full mt-6 flex items-center justify-center gap-2 border-2 border-gray-200 rounded-xl py-3 hover:bg-gray-50 transition-colors font-medium text-gray-700"
        >
          <svg className="w-5 h-5" viewBox="0 0 24 24"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/><path fill="none" d="M1 1h22v22H1z"/></svg>
          Continue with Google
        </button>

        <div className="mt-6 text-center">
          <Link href="/forgot-password" className="text-sm text-brand-teal hover:underline">Forgot password?</Link>
        </div>
      </motion.div>
    </div>
  );
}
