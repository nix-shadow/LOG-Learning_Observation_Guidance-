"use client";

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, CheckCircle2, ChevronRight } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import toast from 'react-hot-toast';
import confetti from 'canvas-confetti';
import { fetchWithCache } from '@/lib/api';
import MicroModuleViewer, { MicroModuleData } from '@/components/MicroModuleViewer';
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
      <div className="flex items-center justify-between text-sm font-medium text-gray-500">
        <span><ArrowLeft className="inline w-4 h-4 mr-2" /> Loading module...</span>
      </div>
      <SkeletonLoader type="card" count={2} />
    </div>
  );

  if (modules.length > 0) {
    const handleComplete = async () => {
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
      <div className="max-w-3xl mx-auto w-full">
        <button onClick={() => router.back()} className="text-gray-500 hover:text-brand-blue flex items-center mb-6 transition-colors">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Journey
        </button>
        {activityTitle && (
          <h1 className="text-2xl font-bold text-brand-blue mb-4">{activityTitle}</h1>
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
          colors: ['#00B4D8', '#4285F4', '#34A853', '#FBBC05', '#EA4335']
        });
      }
    }
  };

  const handleAnswerSelect = (opt: string) => {
    setSelectedAnswer(opt);
    const correct = opt === step.correctAnswer;
    setIsCorrect(correct);
    if (correct) {
      toast.success('Correct!');
    } else {
      toast.error('Not quite, try again.');
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
        <button onClick={() => router.back()} className="text-gray-500 hover:text-brand-blue flex items-center mb-6 transition-colors">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Journey
        </button>

        <div className="flex items-center justify-between text-sm font-medium text-gray-500 mb-2">
          <span>Module Progress</span>
          <span>{Math.round(progress)}%</span>
        </div>
        <div className="w-full bg-gray-200 h-2.5 rounded-full overflow-hidden">
          <motion.div
            className="bg-brand-teal h-2.5 rounded-full"
            initial={{ width: 0 }}
            animate={{ width: `${progress}%` }}
            transition={{ duration: 0.5 }}
          />
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col justify-center">
        <AnimatePresence mode="wait">
          <motion.div
            key={currentStep}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.3 }}
            className="card bg-white shadow-xl border-none p-8 md:p-12"
          >
            {step.type === 'concept' && (
              <div className="space-y-6">
                <span className="inline-block px-3 py-1 bg-brand-blue/10 text-brand-blue text-xs font-bold uppercase rounded-full tracking-wider">Concept</span>
                <h1 className="text-3xl font-bold text-brand-blue">{step.title}</h1>
                <p className="text-lg text-gray-700 leading-relaxed">{step.body}</p>
              </div>
            )}

            {step.type === 'interactive' && (
              <div className="space-y-6">
                <span className="inline-block px-3 py-1 bg-brand-amber/20 text-brand-amber text-xs font-bold uppercase rounded-full tracking-wider">Knowledge Check</span>
                <h1 className="text-2xl font-bold text-brand-blue">{step.question}</h1>

                <div className="space-y-3 mt-8">
                  {step.options?.map((opt) => {
                    let btnClass = "w-full text-left p-4 rounded-xl border-2 transition-all font-medium ";
                    if (selectedAnswer === opt) {
                       btnClass += isCorrect
                         ? "border-green-500 bg-green-50 text-green-700"
                         : "border-red-400 bg-red-50 text-red-700";
                    } else {
                       btnClass += "border-gray-200 hover:border-brand-teal hover:bg-brand-teal/5 text-gray-700";
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
                  <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="p-4 bg-brand-blue/5 rounded-lg border border-brand-blue/10">
                    <p className="text-brand-blue font-medium flex items-start">
                      <LightbulbIcon className="w-5 h-5 mr-2 flex-shrink-0 mt-0.5 text-brand-amber" />
                      {step.explanation}
                    </p>
                  </motion.div>
                )}
              </div>
            )}

            {step.type === 'completion' && (
              <div className="text-center space-y-6 py-8">
                <div className="mx-auto w-24 h-24 bg-green-100 text-green-500 rounded-full flex items-center justify-center">
                  <CheckCircle2 className="w-12 h-12" />
                </div>
                <h1 className="text-3xl font-extrabold text-brand-blue">{step.title}</h1>
                <p className="text-lg text-gray-600">{step.body}</p>
              </div>
            )}
          </motion.div>
        </AnimatePresence>
      </div>

      {/* Footer / Controls */}
      <div className="mt-8 flex justify-end">
        {step.type === 'interactive' && !isCorrect ? (
          <button disabled className="btn-primary opacity-50 cursor-not-allowed">
            Select an answer to continue
          </button>
        ) : step.type === 'completion' ? (
          <button onClick={handleFinish} className="btn-primary flex items-center px-8 py-3 text-lg">
            Complete Lesson <CheckCircle2 className="ml-2 w-5 h-5" />
          </button>
        ) : (
          <button onClick={handleNext} className="btn-primary flex items-center px-8 py-3 text-lg">
            Continue <ChevronRight className="ml-2 w-5 h-5" />
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
