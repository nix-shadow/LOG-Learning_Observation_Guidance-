"use client";

import { useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { motion, AnimatePresence } from 'framer-motion';
import { BookOpen, Search, Filter, Star, Clock, Users, ArrowRight, PlayCircle } from 'lucide-react';
import Link from 'next/link';

// Expanded catalog data simulating a large application
const CATALOG = Array.from({ length: 24 }).map((_, i) => ({
  id: `course-${i}`,
  title: [
    'Advanced Boolean Logic', 'Data Structures in Go', 'React Performance Patterns',
    'PostgreSQL Indexing', 'System Design Basics', 'UI/UX Fundamentals',
    'Microservices Architecture', 'Network Protocols'
  ][i % 8] + ` ${Math.floor(i / 8) > 0 ? `(Part ${Math.floor(i / 8) + 1})` : ''}`,
  category: ['Computer Science', 'Frontend', 'Backend', 'Design'][i % 4],
  difficulty: ['Beginner', 'Intermediate', 'Advanced'][i % 3],
  rating: (Math.random() * 1 + 4).toFixed(1), // 4.0 - 5.0
  duration: `${Math.floor(Math.random() * 10) + 2} hours`,
  enrolled: Math.floor(Math.random() * 10000) + 500,
  imageColor: ['bg-brand-blue', 'bg-brand-teal', 'bg-brand-amber', 'bg-purple-600', 'bg-indigo-600'][i % 5],
}));

export default function CoursesCatalog() {
  useAuth();
  const [searchTerm, setSearchTerm] = useState('');
  const [activeCategory, setActiveCategory] = useState('All');

  const categories = ['All', 'Computer Science', 'Frontend', 'Backend', 'Design'];

  const filteredCourses = CATALOG.filter(c => {
    const matchesSearch = c.title.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = activeCategory === 'All' || c.category === activeCategory;
    return matchesSearch && matchesCategory;
  });

  return (
    <div className="w-full space-y-8">
      {/* Hero Section */}
      <section className="bg-brand-blue rounded-3xl p-8 md:p-16 text-white relative overflow-hidden flex flex-col md:flex-row items-center justify-between">
        <div className="relative z-10 max-w-2xl">
          <h1 className="text-4xl md:text-5xl font-bold mb-4">Explore the Catalog</h1>
          <p className="text-brand-gray/80 text-lg mb-8">
            Discover hundreds of expertly crafted modules designed to help you master new skills, even offline.
          </p>
          <div className="relative max-w-md">
            <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
            <input
              type="text"
              placeholder="Search for courses, topics, or skills..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-12 pr-4 py-4 rounded-full text-brand-text outline-none focus:ring-4 focus:ring-brand-teal/50 transition-all"
            />
          </div>
        </div>
        <div className="absolute top-0 right-0 -mt-32 -mr-32 w-[500px] h-[500px] bg-brand-teal/20 rounded-full blur-3xl pointer-events-none"></div>
        <div className="hidden md:block relative z-10 bg-white/10 p-8 rounded-full backdrop-blur-md border border-white/20">
          <BookOpen className="w-24 h-24 text-brand-teal" />
        </div>
      </section>

      {/* Filters */}
      <section className="flex items-center justify-between overflow-x-auto pb-4 gap-4 no-scrollbar">
        <div className="flex gap-2 min-w-max">
          <button className="flex items-center px-4 py-2 bg-gray-100 rounded-full text-sm font-medium text-gray-600 hover:bg-gray-200 transition-colors">
            <Filter className="w-4 h-4 mr-2" /> Filters
          </button>
          <div className="h-8 w-px bg-gray-300 mx-2 self-center"></div>
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat)}
              className={`px-6 py-2 rounded-full text-sm font-medium transition-all ${
                activeCategory === cat
                  ? 'bg-brand-blue text-white shadow-md'
                  : 'bg-white border border-gray-200 text-gray-600 hover:border-brand-teal hover:text-brand-teal'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
      </section>

      {/* Grid */}
      <section>
        <AnimatePresence mode="popLayout">
          <motion.div layout className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {filteredCourses.map((course) => (
              <motion.div
                layout
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
                transition={{ duration: 0.2 }}
                key={course.id}
                className="card p-0 overflow-hidden flex flex-col group hover:shadow-xl hover:-translate-y-1 transition-all duration-300"
              >
                <div className={`h-40 w-full ${course.imageColor} relative overflow-hidden flex items-center justify-center`}>
                  <div className="absolute inset-0 bg-black/10 group-hover:bg-transparent transition-colors"></div>
                  <PlayCircle className="w-16 h-16 text-white/50 group-hover:text-white/90 transition-colors transform group-hover:scale-110 duration-300" />
                  <div className="absolute top-4 left-4 bg-white/20 backdrop-blur-md px-3 py-1 rounded-full text-xs font-bold text-white uppercase tracking-wider">
                    {course.category}
                  </div>
                </div>

                <div className="p-5 flex-1 flex flex-col">
                  <div className="flex items-center justify-between mb-3">
                    <span className={`text-xs font-bold uppercase tracking-wider ${
                      course.difficulty === 'Beginner' ? 'text-green-600' :
                      course.difficulty === 'Intermediate' ? 'text-brand-amber' : 'text-red-600'
                    }`}>
                      {course.difficulty}
                    </span>
                    <span className="flex items-center text-sm font-medium text-gray-600">
                      <Star className="w-4 h-4 text-brand-amber mr-1 fill-current" /> {course.rating}
                    </span>
                  </div>

                  <h3 className="text-lg font-bold text-brand-blue mb-2 line-clamp-2 group-hover:text-brand-teal transition-colors">
                    {course.title}
                  </h3>

                  <div className="mt-auto pt-4 border-t border-gray-100 flex items-center justify-between text-sm text-gray-500">
                    <span className="flex items-center"><Clock className="w-4 h-4 mr-1" /> {course.duration}</span>
                    <span className="flex items-center"><Users className="w-4 h-4 mr-1" /> {course.enrolled.toLocaleString()}</span>
                  </div>
                </div>

                <div className="p-4 pt-0">
                  <Link href={`/learning/${course.id}`} className="w-full py-3 bg-gray-50 hover:bg-brand-teal hover:text-white text-brand-blue rounded-xl font-semibold flex items-center justify-center transition-colors">
                    Start Learning <ArrowRight className="w-4 h-4 ml-2" />
                  </Link>
                </div>
              </motion.div>
            ))}
          </motion.div>
        </AnimatePresence>

        {filteredCourses.length === 0 && (
          <div className="text-center py-24">
            <div className="w-24 h-24 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-6">
              <Search className="w-10 h-10 text-gray-400" />
            </div>
            <h3 className="text-2xl font-bold text-brand-blue mb-2">No courses found</h3>
            <p className="text-gray-500">Try adjusting your filters or search terms.</p>
          </div>
        )}
      </section>
    </div>
  );
}
