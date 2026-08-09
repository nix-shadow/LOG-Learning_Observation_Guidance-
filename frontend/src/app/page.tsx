import Link from 'next/link';
import { ArrowRight, BookOpen, LineChart, Target, Compass, Zap } from 'lucide-react';
import Image from 'next/image';

export default function Home() {
  const steps = [
    { title: "Learn", icon: BookOpen, desc: "Engage with bite-sized educational modules." },
    { title: "Observe", icon: LineChart, desc: "See clear reflections of your progress and habits." },
    { title: "Understand", icon: Target, desc: "Identify your strengths and areas needing attention." },
    { title: "Guide", icon: Compass, desc: "Receive actionable, targeted recommendations." },
    { title: "Improve", icon: Zap, desc: "Apply guidance to build lasting knowledge." },
  ];

  return (
    <div className="flex flex-col items-center">
      {/* Hero Section */}
      <section className="w-full py-12 md:py-24 lg:py-32 flex flex-col items-center text-center">
        <Image src="/assets/log-logo.png" alt="LOG Logo" width={250} height={100} className="mb-8" />
        <h1 className="text-4xl md:text-6xl font-extrabold text-brand-blue tracking-tight mb-6">
          A smart learning companion.
        </h1>
        <p className="text-xl md:text-2xl text-gray-600 max-w-3xl mb-10">
          Designed for low-connectivity. Optimized for your growth. LOG helps you understand your learning journey and guides your next steps.
        </p>
        <div className="flex flex-col sm:flex-row gap-4">
          <Link href="/dashboard" className="btn-primary flex items-center justify-center text-lg px-8 py-3">
            Start Learning <ArrowRight className="ml-2 w-5 h-5" />
          </Link>
        </div>
      </section>

      {/* How it Works Section */}
      <section className="w-full py-12 md:py-24 bg-brand-gray/20 rounded-3xl px-6 md:px-12 my-12">
        <div className="text-center mb-16">
          <h2 className="text-3xl font-bold text-brand-blue mb-4">The LOG Cycle</h2>
          <p className="text-gray-600 max-w-2xl mx-auto">
            A continuous loop of purposeful learning, designed to keep you moving forward.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-8">
          {steps.map((step, idx) => {
            const Icon = step.icon;
            return (
              <div key={idx} className="flex flex-col items-center text-center p-6 bg-white rounded-2xl shadow-sm border border-brand-gray">
                <div className="w-16 h-16 rounded-full bg-brand-teal/10 flex items-center justify-center text-brand-teal mb-4">
                  <Icon className="w-8 h-8" />
                </div>
                <h3 className="font-bold text-lg mb-2 text-brand-blue">{step.title}</h3>
                <p className="text-sm text-gray-600">{step.desc}</p>
              </div>
            )
          })}
        </div>
      </section>
    </div>
  );
}
