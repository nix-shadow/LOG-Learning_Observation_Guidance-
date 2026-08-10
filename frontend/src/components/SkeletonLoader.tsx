import { motion } from 'framer-motion';

interface SkeletonProps {
  type: 'card' | 'text' | 'chart' | 'stats';
  count?: number;
}

export default function SkeletonLoader({ type, count = 1 }: SkeletonProps) {
  const elements = Array.from({ length: count }, (_, i) => i);

  if (type === 'stats') {
    return (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
        {elements.map((i) => (
          <div key={i} className="card p-6 flex flex-col items-center justify-center space-y-4">
            <motion.div
              className="w-24 h-4 bg-brand-gray/50 rounded-full"
              animate={{ opacity: [0.5, 1, 0.5] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
            />
            <motion.div
              className="w-16 h-10 bg-brand-gray/50 rounded-lg"
              animate={{ opacity: [0.5, 1, 0.5] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut", delay: 0.2 }}
            />
          </div>
        ))}
      </div>
    );
  }

  if (type === 'card') {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {elements.map((i) => (
          <div key={i} className="card p-6 flex flex-col space-y-4">
            <motion.div
              className="w-3/4 h-6 bg-brand-gray/50 rounded-md"
              animate={{ opacity: [0.5, 1, 0.5] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
            />
            <motion.div
              className="w-full h-16 bg-brand-gray/50 rounded-md"
              animate={{ opacity: [0.5, 1, 0.5] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut", delay: 0.1 }}
            />
            <motion.div
              className="w-1/2 h-4 bg-brand-gray/50 rounded-full mt-4"
              animate={{ opacity: [0.5, 1, 0.5] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut", delay: 0.2 }}
            />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-4 w-full">
      {elements.map((i) => (
        <motion.div
          key={i}
          className="w-full h-12 bg-brand-gray/50 rounded-md"
          animate={{ opacity: [0.5, 1, 0.5] }}
          transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut", delay: i * 0.1 }}
        />
      ))}
    </div>
  );
}
