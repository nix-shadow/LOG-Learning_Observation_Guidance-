import { motion, Transition } from 'framer-motion';

interface SkeletonProps {
  type: 'card' | 'text' | 'chart' | 'stats';
  count?: number;
}

export default function SkeletonLoader({ type, count = 1 }: SkeletonProps) {
  const elements = Array.from({ length: count }, (_, i) => i);

  // High-end shimmer effect using background-position instead of basic opacity
  const shimmerAnimation = {
    backgroundPosition: ["200% 0", "-200% 0"],
  };
  const shimmerTransition: Transition = {
    repeat: Infinity,
    duration: 2,
    ease: "linear",
  };
  const shimmerClass = "bg-gradient-to-r from-brand-gray/40 via-brand-white/80 to-brand-gray/40 bg-[length:200%_100%]";

  if (type === 'stats') {
    return (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
        {elements.map((i) => (
          <motion.div 
            key={i} 
            className="card p-6 flex flex-col items-center justify-center space-y-4 border-none shadow-sm"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ type: "spring", stiffness: 300, damping: 20, delay: i * 0.1 }}
          >
            <motion.div
              className={`w-24 h-4 rounded-full ${shimmerClass}`}
              animate={shimmerAnimation}
              transition={shimmerTransition}
            />
            <motion.div
              className={`w-16 h-10 rounded-lg ${shimmerClass}`}
              animate={shimmerAnimation}
              transition={shimmerTransition}
            />
          </motion.div>
        ))}
      </div>
    );
  }

  if (type === 'card') {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {elements.map((i) => (
          <motion.div 
            key={i} 
            className="card p-6 flex flex-col space-y-4 border-none shadow-sm"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ type: "spring", stiffness: 300, damping: 20, delay: i * 0.1 }}
          >
            <motion.div
              className={`w-3/4 h-6 rounded-md ${shimmerClass}`}
              animate={shimmerAnimation}
              transition={shimmerTransition}
            />
            <motion.div
              className={`w-full h-16 rounded-md ${shimmerClass}`}
              animate={shimmerAnimation}
              transition={shimmerTransition}
            />
            <motion.div
              className={`w-1/2 h-4 rounded-full mt-4 ${shimmerClass}`}
              animate={shimmerAnimation}
              transition={shimmerTransition}
            />
          </motion.div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-4 w-full">
      {elements.map((i) => (
        <motion.div
          key={i}
          className={`w-full h-12 rounded-md ${shimmerClass}`}
          initial={{ opacity: 0, x: -10 }}
          animate={{ opacity: 1, x: 0, ...shimmerAnimation }}
          transition={{ 
            opacity: { duration: 0.3, delay: i * 0.05 },
            x: { type: "spring", stiffness: 300, damping: 20, delay: i * 0.05 },
            backgroundPosition: shimmerTransition
          }}
        />
      ))}
    </div>
  );
}
