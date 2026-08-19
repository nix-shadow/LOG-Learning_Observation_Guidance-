"use client";

import Link from 'next/link';
import { ArrowRight, BookOpen, LineChart, Target, Compass, Zap } from 'lucide-react';
import Image from 'next/image';
import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { prefersReducedMotion } from '@/lib/motion';

export default function Home() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (prefersReducedMotion()) return;
    if (containerRef.current) {
      gsap.fromTo(
        gsap.utils.toArray('.gsap-stagger'),
        { y: 50, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.8, stagger: 0.15, ease: 'power3.out' }
      );
    }
  }, []);

  const steps = [
    { title: "Learn", icon: BookOpen, desc: "Engage with bite-sized educational modules." },
    { title: "Observe", icon: LineChart, desc: "See clear reflections of your progress and habits." },
    { title: "Understand", icon: Target, desc: "Identify your strengths and areas needing attention." },
    { title: "Guide", icon: Compass, desc: "Receive actionable, targeted recommendations." },
    { title: "Improve", icon: Zap, desc: "Apply guidance to build lasting knowledge." },
  ];

  return (
    <div ref={containerRef} className="flex flex-col items-center">
      {/* Hero Section */}
      <section className="w-full py-12 md:py-24 lg:py-32 flex flex-col items-center text-center relative">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-brand-neon/20 rounded-full blur-[120px] pointer-events-none -z-10"></div>
        
        <div className="gsap-stagger card-glow w-full max-w-4xl p-12 md:p-16 flex flex-col items-center border border-white/10 shadow-[0_0_80px_rgba(0,180,216,0.15)]">
          <Image src="/assets/log-logo.png" alt="LOG Logo" width={250} height={100} className="mb-10 drop-shadow-[0_0_15px_rgba(255,255,255,0.3)] dark:invert" />
          
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight mb-8 text-transparent bg-clip-text bg-gradient-to-r from-white via-brand-neon to-brand-amber">
            A smart learning companion.
          </h1>
          
          <p className="text-xl md:text-2xl text-white/70 max-w-3xl mb-12 leading-relaxed">
            Designed for low-connectivity. Optimized for your growth. LOG helps you understand your learning journey and guides your next steps.
          </p>
          
          <div className="flex flex-col sm:flex-row gap-6">
            <Link href="/dashboard" className="btn-primary group flex items-center justify-center text-lg px-10 py-4 shadow-[0_0_20px_rgba(0,240,255,0.4)]">
              Start Learning <ArrowRight className="ml-3 w-6 h-6 group-hover:translate-x-1 transition-transform" />
            </Link>
          </div>
        </div>
      </section>

      {/* How it Works Section */}
      <section className="gsap-stagger w-full py-16 md:py-24 bg-black/40 backdrop-blur-3xl border border-white/10 rounded-[40px] px-8 md:px-16 my-16 shadow-bento relative overflow-hidden">
        <div className="absolute top-0 right-0 -mt-20 -mr-20 w-96 h-96 bg-brand-amber/10 rounded-full blur-[100px] pointer-events-none"></div>
        <div className="absolute bottom-0 left-0 -mb-20 -ml-20 w-96 h-96 bg-purple-500/10 rounded-full blur-[100px] pointer-events-none"></div>

        <div className="relative z-10 text-center mb-20">
          <h2 className="text-4xl font-bold text-white mb-6 tracking-tight">The LOG Cycle</h2>
          <p className="text-white/60 text-lg max-w-2xl mx-auto">
            A continuous loop of purposeful learning, designed to keep you moving forward.
          </p>
        </div>

        <div className="relative z-10 grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-8">
          {steps.map((step, idx) => {
            const Icon = step.icon;
            return (
              <div key={idx} className="gsap-stagger group flex flex-col items-center text-center p-8 bg-white/5 backdrop-blur-xl rounded-[24px] shadow-bento border border-white/10 hover:border-white/30 hover:shadow-glow hover:-translate-y-2 transition-all duration-300">
                <div className="w-20 h-20 rounded-full bg-brand-teal/10 flex items-center justify-center text-brand-neon mb-6 border border-brand-teal/20 group-hover:scale-110 group-hover:shadow-[0_0_15px_rgba(0,240,255,0.3)] transition-all duration-300">
                  <Icon className="w-10 h-10" />
                </div>
                <h3 className="font-bold text-xl mb-3 text-white tracking-wide">{step.title}</h3>
                <p className="text-sm text-white/60 leading-relaxed">{step.desc}</p>
              </div>
            )
          })}
        </div>
      </section>
    </div>
  );
}
