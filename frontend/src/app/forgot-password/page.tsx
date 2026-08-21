"use client";

import { useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import toast from 'react-hot-toast';
import { m as motion } from 'framer-motion';
import Link from 'next/link';
import { ArrowLeft, CheckCircle2 } from 'lucide-react';

export default function ForgotPassword() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await fetchWithCache('/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });
      toast.success('Reset link sent!');
      setSubmitted(true);
    } catch {
      toast.error('Failed to send reset link.');
    }
    setLoading(false);
  };

  return (
    <div className="min-h-[70vh] flex flex-col items-center justify-center">
      <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} className="card-glow bg-black/40 backdrop-blur-3xl border border-white/10 w-full max-w-md p-8 md:p-12 rounded-[2rem]">
        <Link href="/login" className="flex items-center text-sm font-bold text-white/50 hover:text-brand-neon mb-8 transition-colors tracking-wide">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Login
        </Link>

        <h2 className="text-3xl font-bold text-white tracking-tight mb-2">Reset Password</h2>

        {!submitted ? (
          <>
            <p className="text-white/60 mb-8 text-lg">Enter your email address and we&apos;ll send you a link to reset your password.</p>
            <form onSubmit={handleSubmit} className="space-y-6">
              <div>
                <label className="block text-sm font-bold text-white/80 mb-2 uppercase tracking-wider">Email Address</label>
                <input
                  type="email" required
                  value={email} onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="w-full px-4 py-4 bg-black/50 border border-white/10 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-brand-neon/50 transition-all placeholder-white/30"
                />
              </div>
              <button type="submit" disabled={loading} className="btn-primary w-full py-4 text-lg font-bold tracking-wide">
                {loading ? 'Sending...' : 'Send Reset Link'}
              </button>
            </form>
          </>
        ) : (
          <div className="text-center py-6">
            <div className="w-20 h-20 bg-brand-neon/20 border border-brand-neon/30 text-brand-neon rounded-full flex items-center justify-center mx-auto mb-6">
              <CheckCircle2 className="w-10 h-10" />
            </div>
            <h3 className="text-2xl font-bold text-white mb-3">Check your email</h3>
            <p className="text-white/60 text-lg">If an account exists for {email}, a reset link has been sent.</p>
          </div>
        )}
      </motion.div>
    </div>
  );
}
