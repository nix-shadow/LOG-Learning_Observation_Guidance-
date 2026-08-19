import React, { useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { m as motion, AnimatePresence } from 'framer-motion';
import { ChevronRight, ChevronLeft, CheckCircle2 } from 'lucide-react';

export interface MicroModuleData {
  id: string;
  title: string;
  content_text: string;
  media_url?: string;
  question?: string;
  options?: string[];
  correct_index?: number;
  explanation?: string;
}

export interface AttemptReport {
  elapsed_seconds: number;
  correct_count: number;
  total_count: number;
  // WP-0.2 research round: the completion wall-clock + the learner's IANA
  // timezone, so an offline completion that flushes days later is dated when
  // it happened (server clamps to a sane window) and "today" is the learner's
  // calendar day, not server UTC.
  completed_at_unix_ms?: number;
  timezone_iana?: string;
}

interface Props {
  modules: MicroModuleData[];
  onComplete: (stats: AttemptReport) => void;
}

export default function MicroModuleViewer({ modules, onComplete }: Props) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [isCorrect, setIsCorrect] = useState<boolean | null>(null);
  const [answerLocked, setAnswerLocked] = useState(false);
  const statsRef = useRef({ startedAt: Date.now(), correct: 0, total: 0 });
  const firstTryRef = useRef<boolean | null>(null);

  useEffect(() => {
    // Reset per-module quiz state when advancing.
    setSelectedIndex(null);
    setIsCorrect(null);
    setAnswerLocked(false);
    firstTryRef.current = null;
  }, [currentIndex]);

  if (!modules || modules.length === 0) return null;

  const currentModule = modules[currentIndex];
  const isLast = currentIndex === modules.length - 1;
  const hasQuiz = Boolean(currentModule.question && currentModule.options?.length);
  const correctIndex = currentModule.correct_index ?? -1;

  const finishAttempt = () => {
    const now = Date.now();
    onComplete({
      elapsed_seconds: Math.max(0, Math.round((now - statsRef.current.startedAt) / 1000)),
      correct_count: statsRef.current.correct,
      total_count: statsRef.current.total,
      completed_at_unix_ms: now,
      timezone_iana: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
    });
  };

  const handleNext = () => {
    if (!isLast) {
      setCurrentIndex(prev => prev + 1);
    } else {
      finishAttempt();
    }
  };

  const handlePrev = () => {
    if (currentIndex > 0) {
      setCurrentIndex(prev => prev - 1);
    }
  };

  const handleAnswerSelect = (idx: number) => {
    if (answerLocked) return;
    // First try counts towards accuracy — exactly like the real attempt facts.
    if (firstTryRef.current === null) {
      firstTryRef.current = idx === correctIndex;
      statsRef.current.total += 1;
      if (firstTryRef.current) statsRef.current.correct += 1;
    }
    setSelectedIndex(idx);
    setIsCorrect(idx === correctIndex);
    if (idx === correctIndex) {
      setAnswerLocked(true);
    }
  };

  const answerButtonClass = (opt: string, idx: number) => {
    let cls = "w-full text-left px-6 py-4 rounded-2xl border transition-all font-bold tracking-wide text-lg ";
    if (selectedIndex === idx) {
      cls += isCorrect
        ? "border-green-500 bg-green-500/10 text-green-400 shadow-[0_0_15px_rgba(34,197,94,0.3)]"
        : "border-red-500 bg-red-500/10 text-red-400 shadow-[0_0_15px_rgba(239,68,68,0.3)]";
    } else {
      cls += "border-white/10 bg-white/5 hover:border-brand-neon hover:bg-brand-neon/10 hover:shadow-glow text-white hover:text-brand-neon";
    }
    return cls;
  };

  return (
    <div className="card-glow bg-black/40 backdrop-blur-3xl border border-white/10 max-w-2xl mx-auto overflow-hidden rounded-[2rem] p-8 shadow-glow relative">
      <div className="absolute inset-0 bg-brand-neon/5 pointer-events-none" />
      <div className="mb-6 flex items-center justify-between relative z-10">
        <span className="text-xs font-bold text-white/50 uppercase tracking-widest">
          Micro-Module {currentIndex + 1} / {modules.length}
        </span>
        <div className="flex gap-1.5">
          {modules.map((_, idx) => (
            <div
              key={idx}
              className={`h-1.5 rounded-full ${idx <= currentIndex ? 'bg-brand-neon w-6 shadow-[0_0_8px_rgba(0,240,255,0.8)]' : 'bg-white/10 w-3'} transition-all duration-300`}
            />
          ))}
        </div>
      </div>

      <div className="relative z-10">
        <AnimatePresence mode="wait">
          <motion.div
            key={currentModule.id}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.3 }}
            className="min-h-[250px]"
          >
            <h2 className="text-2xl font-bold text-white tracking-tight mb-4">{currentModule.title}</h2>

            {currentModule.media_url && (
              <div className="mb-6 rounded-2xl overflow-hidden bg-black/50 border border-white/5 flex items-center justify-center min-h-[150px] relative p-4">
                {/* unoptimized allows offline/local media URLs to load without Next.js image CDN */}
                <Image
                  src={currentModule.media_url}
                  alt={currentModule.title}
                  width={400}
                  height={200}
                  unoptimized
                  className="max-h-[200px] object-contain drop-shadow-lg"
                  style={{ width: 'auto', height: 'auto', maxHeight: '200px' }}
                />
              </div>
            )}

            <p className="text-white/80 text-lg leading-relaxed whitespace-pre-wrap">
              {currentModule.content_text}
            </p>

            {hasQuiz && (
              <div className="mt-8 space-y-4">
                <span className="inline-block px-4 py-1.5 bg-brand-amber/20 text-brand-amber border border-brand-amber/30 text-[10px] font-bold uppercase rounded-full tracking-widest">
                  Knowledge Check
                </span>
                <h3 className="text-xl font-bold text-white">{currentModule.question}</h3>

                {currentModule.options?.map((opt, idx) => (
                  <button
                    key={opt}
                    onClick={() => handleAnswerSelect(idx)}
                    disabled={answerLocked}
                    className={answerButtonClass(opt, idx)}
                  >
                    {opt}
                  </button>
                ))}

                {isCorrect !== null && (
                  <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className={`p-5 rounded-2xl border mt-2 backdrop-blur-md ${
                      isCorrect
                        ? 'bg-brand-blue/10 border-brand-blue/20'
                        : 'bg-brand-amber/10 border-brand-amber/20'
                    }`}
                  >
                    <p className={isCorrect ? 'text-brand-blue font-medium' : 'text-brand-amber font-medium'}>
                      {isCorrect ? 'Correct!' : 'Not quite — give it another try!'}
                    </p>
                    {isCorrect && currentModule.explanation && (
                      <p className="text-white/70 text-sm mt-2">{currentModule.explanation}</p>
                    )}
                  </motion.div>
                )}
              </div>
            )}
          </motion.div>
        </AnimatePresence>
      </div>

      <div className="mt-8 pt-6 border-t border-white/10 flex justify-between items-center relative z-10">
        <button
          onClick={handlePrev}
          disabled={currentIndex === 0}
          className={`flex items-center text-sm font-bold tracking-wide transition-all ${currentIndex === 0 ? 'text-white/20 cursor-not-allowed' : 'text-white/60 hover:text-brand-neon'}`}
        >
          <ChevronLeft className="w-4 h-4 mr-2" /> Previous
        </button>

        <button
          onClick={handleNext}
          disabled={hasQuiz && !answerLocked}
          className={`btn-primary flex items-center gap-2 px-6 py-3 font-bold tracking-wide shadow-glow ${
            hasQuiz && !answerLocked ? 'opacity-40 cursor-not-allowed' : ''
          }`}
        >
          {isLast ? (
            <>Complete <CheckCircle2 className="w-5 h-5 ml-1" /></>
          ) : (
            <>Next <ChevronRight className="w-5 h-5 ml-1" /></>
          )}
        </button>
      </div>
    </div>
  );
}