"use client";

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, CheckCircle2, ChevronRight } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import toast from 'react-hot-toast';
import confetti from 'canvas-confetti';
import { fetchWithCache } from '@/lib/api';
import MicroModuleViewer, { MicroModuleData, AttemptReport } from '@/components/MicroModuleViewer';
import SkeletonLoader from '@/components/SkeletonLoader';

// Mock content for the interactive learning module
const lessonContent = [
  {
    id: 1,
    type: 'concept',
    title: 'Understanding Boolean AND',
    body: 'The AND operator returns true only if BOTH operands are true. Think of it like a strict bouncer at a club: you need both an ID AND a ticket to get in.',
  },
  {
    id: 2,
    type: 'interactive',
    title: 'Knowledge Check',
    question: 'If A is True and B is False, what is (A AND B)?',
    options: ['True', 'False', 'Depends on the language'],
    correctAnswer: 'False',
    explanation: 'Because B is False, the entire AND statement becomes False. Both must be True!'
  },
  {
    id: 3,
    type: 'concept',
    title: 'Understanding Boolean OR',
    body: 'The OR operator returns true if AT LEAST ONE operand is true. Think of it like ordering at a restaurant: you can pay with Cash OR Credit Card.',
  },
  {
    id: 4,
    type: 'completion',
    title: 'Lesson Complete!',
    body: 'Great job! You\'ve mastered the basic concepts of AND and OR logic gates.',
  }
];

export default function LessonModule() {
  const router = useRouter();
  const params = useParams();
  const activityId = params?.id as string || 'act-2';
  const [currentStep, setCurrentStep] = useState(0);
  const [selectedAnswer, setSelectedAnswer] = useState<string | null>(null);
  const [isCorrect, setIsCorrect] = useState<boolean | null>(null);
  const [modules, setModules] = useState<MicroModuleData[]>([]);
  const [activityTitle, setActivityTitle] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch the real micro-modules for this activity (cached for offline replay).
    fetchWithCache(`/activities/${activityId}/modules`)
      .then((res) => {
        setModules((res.modules || []).map((m: MicroModuleData) => ({
          id: m.id,
          title: m.title,
          content_text: m.content_text,
          media_url: m.media_url,
          question: m.question,
          options: m.options,
          correct_index: m.correct_index,
          explanation: m.explanation,
        })));
        setActivityTitle(res.activity?.title || '');
      })
      .catch((err) => {
        console.warn('No micro-modules for this activity — using demo lesson', err);
      })
      .finally(() => setLoading(false));
  }, [activityId]);

  // Server modules take priority; the demo lesson is only an offline/catalog fallback.
  if (loading) return (
    <div className="max-w-3xl mx-auto w-full space-y-6">
      <div className="flex items-center justify-between text-sm font-bold text-white/50 tracking-wider uppercase">
        <span><ArrowLeft className="inline w-4 h-4 mr-2" /> Loading module...</span>
      </div>
      <SkeletonLoader type="card" count={2} />
    </div>
  );

  if (modules.length > 0) {
    const handleComplete = async (stats: AttemptReport) => {
      try {
        await fetchWithCache(`/activities/${activityId}/complete`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(stats),
        });
        toast.success('Lesson marked as completed! Progress recorded.', { icon: '🎉' });
      } catch (e) {
        console.warn('Offline completion queued or sync in progress', e);
      }
      router.push('/learning');
    };

    return (
      <div className="max-w-3xl mx-auto w-full">
        <button onClick={() => router.back()} className="text-white/50 hover:text-brand-neon flex items-center mb-6 transition-colors font-bold tracking-wide text-sm uppercase">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Journey
        </button>
        {activityTitle && (
          <h1 className="text-3xl font-bold text-white mb-6 tracking-tight">{activityTitle}</h1>
        )}
        <MicroModuleViewer modules={modules} onComplete={handleComplete} />
      </div>
    );
  }

  const step = lessonContent[currentStep];
  const progress = ((currentStep) / (lessonContent.length - 1)) * 100;

  const handleNext = () => {
    if (currentStep < lessonContent.length - 1) {
      const nextStepIndex = currentStep + 1;
      setCurrentStep(nextStepIndex);
      setSelectedAnswer(null);
      setIsCorrect(null);
      
      if (lessonContent[nextStepIndex].type === 'completion') {
        confetti({
          particleCount: 150,
          spread: 70,
          origin: { y: 0.6 },
          colors: ['#00F0FF', '#00B4D8', '#FFB703', '#FF003C']
        });
      }
    }
  };

  const handleAnswerSelect = (opt: string) => {
    setSelectedAnswer(opt);
    const correct = opt === step.correctAnswer;
    setIsCorrect(correct);
    if (correct) {
      toast.success('Correct!', { style: { background: 'rgba(0,0,0,0.8)', color: '#00F0FF', border: '1px solid rgba(0,240,255,0.3)' } });
    } else {
      toast.error('Not quite, try again.', { style: { background: 'rgba(0,0,0,0.8)', color: '#FF003C', border: '1px solid rgba(255,0,60,0.3)' } });
    }
  };

  const handleFinish = async () => {
    try {
      await fetchWithCache(`/activities/${activityId}/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      toast.success('Lesson marked as completed! Progress recorded.', { icon: '🎉' });
    } catch (e) {
      console.warn('Offline completion queued or sync in progress', e);
    }
    router.push('/learning');
  };

  return (
    <div className="max-w-3xl mx-auto w-full min-h-[70vh] flex flex-col">
      {/* Header & Progress */}
      <div className="mb-8">
        <button onClick={() => router.back()} className="text-white/50 hover:text-brand-neon flex items-center mb-8 transition-colors font-bold tracking-wide text-sm uppercase">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Journey
        </button>

        <div className="flex items-center justify-between text-xs font-bold text-white/50 mb-3 tracking-widest uppercase">
          <span>Module Progress</span>
          <span>{Math.round(progress)}%</span>
        </div>
        <div className="w-full bg-white/10 h-2.5 rounded-full overflow-hidden shadow-inner">
          <motion.div
            className="bg-brand-neon h-2.5 rounded-full shadow-[0_0_10px_rgba(0,240,255,0.8)]"
            initial={{ width: 0 }}
            animate={{ width: `${progress}%` }}
            transition={{ duration: 0.5 }}
          />
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col justify-center relative">
        <AnimatePresence mode="wait">
          <motion.div
            key={currentStep}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.3 }}
            className="card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-8 md:p-12 rounded-[2rem] shadow-glow relative overflow-hidden"
          >
            <div className="absolute inset-0 bg-brand-neon/5 pointer-events-none" />
            
            <div className="relative z-10">
              {step.type === 'concept' && (
                <div className="space-y-6">
                  <span className="inline-block px-4 py-1.5 bg-brand-blue/20 text-brand-blue border border-brand-blue/30 text-[10px] font-bold uppercase rounded-full tracking-widest">Concept</span>
                  <h1 className="text-3xl font-bold text-white tracking-tight">{step.title}</h1>
                  <p className="text-lg text-white/80 leading-relaxed">{step.body}</p>
                </div>
              )}

              {step.type === 'interactive' && (
                <div className="space-y-6">
                  <span className="inline-block px-4 py-1.5 bg-brand-amber/20 text-brand-amber border border-brand-amber/30 text-[10px] font-bold uppercase rounded-full tracking-widest">Knowledge Check</span>
                  <h1 className="text-2xl font-bold text-white tracking-tight">{step.question}</h1>

                  <div className="space-y-4 mt-8">
                    {step.options?.map((opt) => {
                      let btnClass = "w-full text-left px-6 py-4 rounded-2xl border transition-all font-bold tracking-wide text-lg ";
                      if (selectedAnswer === opt) {
                         btnClass += isCorrect
                           ? "border-green-500 bg-green-500/10 text-green-400 shadow-[0_0_15px_rgba(34,197,94,0.3)]"
                           : "border-red-500 bg-red-500/10 text-red-400 shadow-[0_0_15px_rgba(239,68,68,0.3)]";
                      } else {
                         btnClass += "border-white/10 bg-white/5 hover:border-brand-neon hover:bg-brand-neon/10 hover:shadow-glow text-white hover:text-brand-neon";
                      }

                      return (
                        <button
                          key={opt}
                          onClick={() => handleAnswerSelect(opt)}
                          disabled={isCorrect === true}
                          className={btnClass}
                        >
                          {opt}
                        </button>
                      )
                    })}
                  </div>

                  {isCorrect && (
                    <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="p-5 bg-brand-blue/10 rounded-2xl border border-brand-blue/20 mt-6 backdrop-blur-md">
                      <p className="text-brand-blue font-medium flex items-start">
                        <LightbulbIcon className="w-5 h-5 mr-3 flex-shrink-0 mt-0.5 text-brand-amber drop-shadow-[0_0_5px_rgba(255,183,3,0.8)]" />
                        {step.explanation}
                      </p>
                    </motion.div>
                  )}
                </div>
              )}

              {step.type === 'completion' && (
                <div className="text-center space-y-8 py-10">
                  <div className="mx-auto w-24 h-24 bg-green-500/20 text-green-400 border border-green-500/30 rounded-full flex items-center justify-center shadow-[0_0_30px_rgba(34,197,94,0.3)]">
                    <CheckCircle2 className="w-12 h-12" />
                  </div>
                  <h1 className="text-4xl font-extrabold text-white tracking-tight">{step.title}</h1>
                  <p className="text-xl text-white/60">{step.body}</p>
                </div>
              )}
            </div>
          </motion.div>
        </AnimatePresence>
      </div>

      {/* Footer / Controls */}
      <div className="mt-8 flex justify-end">
        {step.type === 'interactive' && !isCorrect ? (
          <button disabled className="btn-primary opacity-50 cursor-not-allowed px-8 py-4 text-lg font-bold">
            Select an answer
          </button>
        ) : step.type === 'completion' ? (
          <button onClick={handleFinish} className="btn-primary flex items-center px-10 py-4 text-lg font-bold tracking-wide shadow-glow">
            Complete Lesson <CheckCircle2 className="ml-3 w-6 h-6" />
          </button>
        ) : (
          <button onClick={handleNext} className="btn-primary flex items-center px-10 py-4 text-lg font-bold tracking-wide shadow-glow">
            Continue <ChevronRight className="ml-3 w-6 h-6" />
          </button>
        )}
      </div>
    </div>
  );
}

// Simple icon for the explanation
function LightbulbIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg {...props} xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="9" y1="18" x2="15" y2="18"></line>
      <line x1="10" y1="22" x2="14" y2="22"></line>
      <path d="M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1.45.62 2.84 1.5 3.5.76.76 1.23 1.52 1.41 2.5"></path>
    </svg>
  );
}
