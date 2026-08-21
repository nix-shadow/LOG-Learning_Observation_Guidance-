"use client";

import { useEffect, useMemo, useState, useRef } from 'react';
import { useAuth } from '@/context/AuthContext';
import { BookOpen, Search, Filter, Star, Clock, ArrowRight, PlayCircle, WifiOff } from 'lucide-react';
import Link from 'next/link';
import { fetchWithCache } from '@/lib/api';
import toast from 'react-hot-toast';
import SkeletonLoader from '@/components/SkeletonLoader';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { prefersReducedMotion } from '@/lib/motion';

interface Course {
  id: string;
  title: string;
  category: string;
  difficulty: string;
  duration: string;
  rating: number;
  enrolled?: number;
  is_enrolled?: boolean;
}

const CATEGORY_COLORS = ['bg-brand-blue', 'bg-brand-teal', 'bg-brand-amber', 'bg-purple-600', 'bg-[#FF003C]'];

export default function CoursesCatalog() {
  useAuth();
  const [courses, setCourses] = useState<Course[]>([]);
  const [total, setTotal] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeCategory, setActiveCategory] = useState('All');
  const [loading, setLoading] = useState(true);
  const [offlineFallback, setOfflineFallback] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchWithCache('/courses?page=1&limit=100')
      .then((res) => {
        setCourses(res.courses || []);
        setTotal(res.pagination?.total ?? 0);
      })
      .catch((err) => {
        console.warn('Failed to load courses — showing cached/default data', err);
        setOfflineFallback(true);
      })
      .finally(() => setLoading(false));
  }, []);

  const categories = useMemo(() => {
    const set = new Set<string>(courses.map((c) => c.category));
    return ['All', ...Array.from(set)];
  }, [courses]);

  const filteredCourses = courses.filter(c => {
    const matchesSearch = c.title.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = activeCategory === 'All' || c.category === activeCategory;
    return matchesSearch && matchesCategory;
  });

  useGSAP(() => {
    if (prefersReducedMotion()) return;
    if (!loading && filteredCourses.length > 0) {
      gsap.fromTo(
        gsap.utils.toArray('.course-card'),
        { y: 50, opacity: 0 },
        { y: 0, opacity: 1, duration: 0.6, stagger: 0.05, ease: 'power3.out' }
      );
    }
  }, { dependencies: [loading, activeCategory, searchTerm, filteredCourses], scope: containerRef });

  // WP-0.2 C5: enrollment is persisted server-side; offline, the mutation is
  // queued like any other and the UI reflects the optimistic intent honestly
  // ("will sync"). Never a client-side-only state.
  // WP-0.2 research round: a 4xx (real server rejection) must NOT flip the
  // toggle — the previous state is restored and the error surfaces. Only
  // success or an honest queued intent updates the UI.
  const toggleEnrollment = async (course: Course) => {
    const target = !course.is_enrolled;
    try {
      const res = await fetchWithCache(`/courses/${course.id}/enroll`, {
        method: target ? 'POST' : 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('log_token')}`,
          'Content-Type': 'application/json',
        },
      });
      setCourses((prev) => prev.map((c) => (c.id === course.id ? { ...c, is_enrolled: target } : c)));
      if (res && res.queued) {
        toast.success(target ? 'Enrollment saved offline — will sync.' : 'Unenrollment saved offline — will sync.', { icon: '💾' });
      } else {
        toast.success(target ? 'Enrolled!' : 'Unenrolled.');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Could not update enrollment.';
      toast.error(target ? `Enrollment not changed: ${msg}` : `Unenrollment not changed: ${msg}`);
    }
  };

  if (loading) return (
    <div className="w-full space-y-8">
      <SkeletonLoader type="card" count={3} />
    </div>
  );

  return (
    <div ref={containerRef} className="w-full space-y-8">
      {/* Hero Section */}
      <section className="card-glow bg-black/40 backdrop-blur-3xl border border-white/10 rounded-3xl p-8 md:p-16 text-white relative overflow-hidden flex flex-col md:flex-row items-center justify-between">
        <div className="absolute inset-0 bg-brand-neon/5 pointer-events-none" />
        <div className="relative z-10 max-w-2xl">
          <h1 className="text-4xl md:text-5xl font-bold mb-4 tracking-tight">Explore the Catalog</h1>
          <p className="text-white/60 text-lg mb-8">
            Discover expertly crafted modules designed to help you master new skills, even offline.
          </p>
          {total > 0 && (
            <p className="text-brand-muted text-sm mb-4 font-medium tracking-wide uppercase">{total} courses available · Refresh to see the latest</p>
          )}
          <div className="relative max-w-md group">
            <div className="absolute -inset-1 bg-gradient-to-r from-brand-neon to-brand-teal rounded-full blur opacity-25 group-hover:opacity-50 transition duration-500"></div>
            <div className="relative flex items-center bg-black/50 backdrop-blur-xl border border-white/10 rounded-full px-4 py-3">
              <Search className="text-white/50 w-5 h-5 mr-3" />
              <input
                type="text"
                placeholder="Search for courses, topics, or skills..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full bg-transparent text-white placeholder-white/40 outline-none"
              />
            </div>
          </div>
        </div>
        <div className="absolute top-0 right-0 -mt-32 -mr-32 w-[500px] h-[500px] bg-brand-neon/10 rounded-full blur-[100px] pointer-events-none"></div>
        <div className="hidden md:block relative z-10 bg-white/5 p-8 rounded-[2rem] backdrop-blur-md border border-white/10 rotate-12 hover:rotate-0 transition-transform duration-700">
          <BookOpen className="w-24 h-24 text-brand-neon" />
        </div>
      </section>

      {/* Filters */}
      <section className="flex items-center justify-between overflow-x-auto pb-4 gap-4 no-scrollbar">
        <div className="flex gap-2 min-w-max">
          <button className="flex items-center px-4 py-2 bg-white/5 border border-white/10 rounded-full text-sm font-bold text-white/60 hover:text-white hover:bg-white/10 transition-colors tracking-wide">
            <Filter className="w-4 h-4 mr-2" /> Filters
          </button>
          <div className="h-8 w-px bg-white/10 mx-2 self-center"></div>
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat)}
              className={`px-6 py-2 rounded-full text-sm font-bold tracking-wide transition-all ${
                activeCategory === cat
                  ? 'bg-brand-neon/20 border-brand-neon text-brand-neon'
                  : 'bg-white/5 border border-white/10 text-white/60 hover:border-brand-neon/50 hover:text-white'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
      </section>

      {offlineFallback && (
        <div className="flex items-center gap-2 text-sm text-brand-amber bg-brand-amber/10 border border-brand-amber/20 backdrop-blur-md rounded-xl px-4 py-3">
          <WifiOff className="w-4 h-4" />
          Couldn&apos;t reach the network and no cached catalog is available yet. Reconnect to browse the latest courses.
        </div>
      )}

      {/* Grid */}
      <section>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {filteredCourses.map((course) => (
            <div
              key={course.id}
              className="course-card card-glow bg-black/40 backdrop-blur-3xl border border-white/10 p-0 overflow-hidden flex flex-col group hover:-translate-y-1 transition-all duration-300"
            >
              <div className={`h-40 w-full ${CATEGORY_COLORS[stringToIndex(course.id)]} bg-opacity-30 relative overflow-hidden flex items-center justify-center border-b border-white/10`}>
                <div className="absolute inset-0 bg-black/40 group-hover:bg-black/20 transition-colors duration-500"></div>
                <PlayCircle className="w-16 h-16 text-white/70 group-hover:text-brand-neon transition-all transform group-hover:scale-110 duration-500 z-10" />
                <div className="absolute top-4 left-4 bg-black/60 backdrop-blur-md px-3 py-1 rounded-full text-[10px] font-bold text-white uppercase tracking-widest border border-white/10 z-10">
                  {course.category}
                </div>
              </div>

              <div className="p-5 flex-1 flex flex-col relative">
                <div className="flex items-center justify-between mb-3">
                  <span className={`text-[10px] font-bold uppercase tracking-widest px-2 py-1 rounded-full border ${
                    course.difficulty === 'Beginner' ? 'bg-green-500/20 text-green-400 border-green-500/30' :
                    course.difficulty === 'Intermediate' ? 'bg-brand-amber/20 text-brand-amber border-brand-amber/30' : 'bg-red-500/20 text-red-400 border-red-500/30'
                  }`}>
                    {course.difficulty}
                  </span>
                  <span className="flex items-center text-sm font-medium text-white/80">
                    <Star className="w-4 h-4 text-brand-amber mr-1 fill-current" /> {Number(course.rating).toFixed(1)}
                  </span>
                </div>

                <h3 className="text-xl font-bold text-white mb-2 tracking-tight group-hover:text-brand-neon transition-colors duration-300 line-clamp-2">
                  {course.title}
                </h3>

                <div className="mt-auto pt-4 border-t border-white/10 flex items-center justify-between text-sm text-white/50 font-medium">
                  <span className="flex items-center"><Clock className="w-4 h-4 mr-1.5" /> {course.duration}</span>
                  <span className="flex items-center text-brand-muted">
                    <BookOpen className="w-4 h-4 mr-1.5" /> {course.enrolled ?? 0} enrolled
                  </span>
                </div>
              </div>

              <div className="p-4 pt-0 flex gap-3">
                {course.is_enrolled && (
                  <button
                    onClick={() => toggleEnrollment(course)}
                    className="px-4 py-3 bg-brand-teal/20 border border-brand-teal/40 text-brand-teal hover:bg-brand-teal/30 rounded-xl font-bold text-sm transition-all duration-300"
                  >
                    Enrolled ✓
                  </button>
                )}
                <Link href={`/learning/${course.id}`} className="flex-1 py-3 bg-white/5 border border-white/10 hover:border-brand-neon hover:bg-brand-neon/10 text-white hover:text-brand-neon rounded-xl font-bold tracking-wide flex items-center justify-center transition-all duration-300">
                  Start Learning <ArrowRight className="w-4 h-4 ml-2" />
                </Link>
              </div>
            </div>
          ))}
        </div>

        {filteredCourses.length === 0 && (
          <div className="text-center py-24 card-glow bg-black/40 backdrop-blur-3xl border border-white/10 rounded-3xl">
            <div className="w-24 h-24 bg-white/5 border border-white/10 rounded-full flex items-center justify-center mx-auto mb-6">
              <Search className="w-10 h-10 text-brand-faint" />
            </div>
            <h3 className="text-2xl font-bold text-white mb-2 tracking-tight">No courses found</h3>
            <p className="text-white/50">Try adjusting your filters or search terms.</p>
          </div>
        )}
      </section>
    </div>
  );
}

// Deterministic color assignment from a course ID (stable across renders/offline)
function stringToIndex(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash * 31 + str.charCodeAt(i)) >>> 0;
  }
  return hash % CATEGORY_COLORS.length;
}
