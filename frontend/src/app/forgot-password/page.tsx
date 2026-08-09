"use client";

import { useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import toast from 'react-hot-toast';
import { motion } from 'framer-motion';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

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
      <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} className="card w-full max-w-md p-8 shadow-2xl">
        <Link href="/login" className="flex items-center text-sm text-gray-500 hover:text-brand-blue mb-6">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Login
        </Link>

        <h2 className="text-2xl font-bold text-brand-blue mb-2">Reset Password</h2>

        {!submitted ? (
          <>
            <p className="text-gray-500 mb-6">Enter your email address and we&apos;ll send you a link to reset your password.</p>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Email Address</label>
                <input
                  type="email" required
                  value={email} onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="w-full px-4 py-3 rounded-xl border border-gray-300 focus:ring-2 focus:ring-brand-teal focus:border-transparent outline-none transition-all"
                />
              </div>
              <button type="submit" disabled={loading} className="btn-primary w-full py-3 text-lg mt-2">
                {loading ? 'Sending...' : 'Send Reset Link'}
              </button>
            </form>
          </>
        ) : (
          <div className="text-center py-4">
            <div className="w-16 h-16 bg-green-100 text-green-500 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
            </div>
            <h3 className="text-xl font-bold text-brand-blue mb-2">Check your email</h3>
            <p className="text-gray-600">If an account exists for {email}, a reset link has been sent.</p>
          </div>
        )}
      </motion.div>
    </div>
  );
}
