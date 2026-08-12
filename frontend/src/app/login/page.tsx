"use client";

import { useState, useEffect } from 'react';
import { useAuth } from '@/context/AuthContext';
import { fetchWithCache } from '@/lib/api';
import toast from 'react-hot-toast';
import { motion } from 'framer-motion';
import Link from 'next/link';
import Image from 'next/image';
import { GoogleLogin, CredentialResponse } from '@react-oauth/google';

export default function Login() {
  const [step, setStep] = useState<'phone' | 'otp'>('phone');
  const [phone, setPhone] = useState('');
  const [otp, setOtp] = useState('');
  const [loading, setLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const { login } = useAuth();

  useEffect(() => {
    let timer: NodeJS.Timeout;
    if (countdown > 0) {
      timer = setTimeout(() => setCountdown(countdown - 1), 1000);
    }
    return () => clearTimeout(timer);
  }, [countdown]);

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
      setCountdown(60); // start 60s cooldown
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

  const handleGoogleSuccess = async (credentialResponse: CredentialResponse) => {
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
            <div className="flex flex-col items-center gap-2 mt-4">
              <button 
                type="button" 
                onClick={handleRequestOTP} 
                disabled={countdown > 0 || loading}
                className="text-sm font-medium text-brand-teal hover:underline disabled:text-gray-400 disabled:no-underline transition-colors"
              >
                {countdown > 0 ? `Resend OTP in ${countdown}s` : 'Resend OTP'}
              </button>
              <button type="button" onClick={() => { setStep('phone'); setCountdown(0); }} className="text-sm text-gray-500 hover:text-brand-blue">
                Use a different number
              </button>
            </div>
          </form>
        )}

        <div className="mt-8 flex items-center justify-center space-x-4">
          <div className="h-px bg-gray-200 w-full"></div>
          <span className="text-gray-400 text-sm">OR</span>
          <div className="h-px bg-gray-200 w-full"></div>
        </div>

        <div className="mt-6 flex justify-center">
          <GoogleLogin
            onSuccess={handleGoogleSuccess}
            onError={handleGoogleError}
          />
        </div>

        <div className="mt-6 text-center">
          <Link href="/forgot-password" className="text-sm text-brand-teal hover:underline">Forgot password?</Link>
        </div>
      </motion.div>
    </div>
  );
}
