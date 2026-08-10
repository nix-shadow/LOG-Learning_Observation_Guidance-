import React, { useState } from 'react';
import Image from 'next/image';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronRight, ChevronLeft, CheckCircle2 } from 'lucide-react';

export interface MicroModuleData {
  id: string;
  title: string;
  content_text: string;
  media_url?: string;
}

interface Props {
  modules: MicroModuleData[];
  onComplete: () => void;
}

export default function MicroModuleViewer({ modules, onComplete }: Props) {
  const [currentIndex, setCurrentIndex] = useState(0);

  if (!modules || modules.length === 0) return null;

  const currentModule = modules[currentIndex];
  const isLast = currentIndex === modules.length - 1;

  const handleNext = () => {
    if (!isLast) {
      setCurrentIndex(prev => prev + 1);
    } else {
      onComplete();
    }
  };

  const handlePrev = () => {
    if (currentIndex > 0) {
      setCurrentIndex(prev => prev - 1);
    }
  };

  return (
    <div className="card max-w-2xl mx-auto overflow-hidden">
      <div className="mb-4 flex items-center justify-between">
        <span className="text-sm font-bold text-gray-500 uppercase tracking-wider">
          Micro-Module {currentIndex + 1} / {modules.length}
        </span>
        <div className="flex gap-1">
          {modules.map((_, idx) => (
            <div 
              key={idx} 
              className={`h-1.5 rounded-full ${idx <= currentIndex ? 'bg-brand-teal w-6' : 'bg-gray-200 w-3'} transition-all duration-300`} 
            />
          ))}
        </div>
      </div>

      <AnimatePresence mode="wait">
        <motion.div
          key={currentModule.id}
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: -20 }}
          transition={{ duration: 0.3 }}
          className="min-h-[200px]"
        >
          <h2 className="text-xl font-bold text-brand-blue mb-3">{currentModule.title}</h2>
          
          {currentModule.media_url && (
            <div className="mb-4 rounded-xl overflow-hidden bg-gray-100 flex items-center justify-center min-h-[120px] relative">
              {/* unoptimized allows offline/local media URLs to load without Next.js image CDN */}
              <Image
                src={currentModule.media_url}
                alt={currentModule.title}
                width={400}
                height={200}
                unoptimized
                className="max-h-[200px] object-contain"
                style={{ width: 'auto', height: 'auto', maxHeight: '200px' }}
              />
            </div>
          )}
          
          <p className="text-brand-text leading-relaxed whitespace-pre-wrap">
            {currentModule.content_text}
          </p>
        </motion.div>
      </AnimatePresence>

      <div className="mt-8 pt-4 border-t border-gray-100 flex justify-between items-center">
        <button 
          onClick={handlePrev}
          disabled={currentIndex === 0}
          className={`flex items-center text-sm font-semibold ${currentIndex === 0 ? 'text-gray-300 cursor-not-allowed' : 'text-brand-blue hover:text-brand-teal'}`}
        >
          <ChevronLeft className="w-4 h-4 mr-1" /> Previous
        </button>

        <button 
          onClick={handleNext}
          className="btn-primary flex items-center gap-2"
        >
          {isLast ? (
            <>Complete <CheckCircle2 className="w-4 h-4" /></>
          ) : (
            <>Next <ChevronRight className="w-4 h-4" /></>
          )}
        </button>
      </div>
    </div>
  );
}
