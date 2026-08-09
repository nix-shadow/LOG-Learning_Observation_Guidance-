"use client";

import { useEffect, useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import { Users, BookOpen, AlertCircle } from 'lucide-react';
import toast from 'react-hot-toast';
import { motion } from 'framer-motion';

export default function ModeratorDashboard() {
  const { user } = useAuth();
  const router = useRouter();
  const [loading, setLoading] = useState(true);

  const isModerator = user?.role === 'MODERATOR' || user?.role === 'ADMIN';

  useEffect(() => {
    if (!user || !isModerator) {
      toast.error('Unauthorized access. Teachers/Moderators only.');
      router.push('/dashboard');
      return;
    }


    // In a full implementation, we'd fetch moderator specific data.
    // For this build, we mock the successful response to show the UI layer.
    setTimeout(() => setLoading(false), 500);

  }, [user, isModerator, router]);

  if (!isModerator) return null;
  if (loading) return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-teal"></div>
    </div>
  );

  return (
    <div className="max-w-6xl mx-auto w-full space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8 border-b border-gray-200 pb-6">
        <div>
           <h1 className="text-3xl font-bold text-brand-blue flex items-center">
             <BookOpen className="w-8 h-8 text-brand-teal mr-3" /> Teacher Portal
           </h1>
           <p className="text-gray-500 mt-2">Manage your classes and review student progress.</p>
        </div>
        <div className="flex gap-3">
           <button className="btn-secondary">View Reports</button>
           <button className="btn-primary">Create Assignment</button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="card border-t-4 border-t-brand-teal">
           <h3 className="text-gray-500 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><Users className="w-4 h-4 mr-2"/> Active Students</h3>
           <p className="text-4xl font-bold text-brand-blue">124</p>
           <p className="text-sm text-green-600 mt-2">↑ 12% from last week</p>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{delay: 0.1}} className="card border-t-4 border-t-brand-amber">
           <h3 className="text-gray-500 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><AlertCircle className="w-4 h-4 mr-2"/> Needs Attention</h3>
           <p className="text-4xl font-bold text-brand-blue">8</p>
           <p className="text-sm text-brand-amber mt-2">Students falling behind</p>
        </motion.div>

        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{delay: 0.2}} className="card border-t-4 border-t-brand-blue">
           <h3 className="text-gray-500 font-medium mb-1 uppercase tracking-wider text-sm flex items-center"><BookOpen className="w-4 h-4 mr-2"/> Assignments Due</h3>
           <p className="text-4xl font-bold text-brand-blue">3</p>
           <p className="text-sm text-gray-500 mt-2">To be graded by Friday</p>
        </motion.div>
      </div>

      <div className="card mt-12">
        <h2 className="text-xl font-bold text-brand-blue mb-6">Class Roster: Logic 101</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-gray-500 bg-gray-50">
                <th className="pb-3 pt-3 px-4 font-medium rounded-tl-lg">Student Name</th>
                <th className="pb-3 pt-3 px-4 font-medium">Completion %</th>
                <th className="pb-3 pt-3 px-4 font-medium">Current Streak</th>
                <th className="pb-3 pt-3 px-4 font-medium rounded-tr-lg">Action</th>
              </tr>
            </thead>
            <tbody>
              {/* Mock Roster */}
              {['Aisha Student', 'Bikash Thapa', 'Chandan Gurung', 'Dawa Sherpa'].map((name, i) => (
                <tr key={i} className="border-b border-gray-100 last:border-0 hover:bg-gray-50">
                  <td className="py-4 px-4 font-medium text-brand-blue flex items-center">
                    <div className="w-8 h-8 rounded-full bg-brand-teal/20 text-brand-teal flex items-center justify-center mr-3 font-bold">
                      {name.charAt(0)}
                    </div>
                    {name}
                  </td>
                  <td className="py-4 px-4">
                    <div className="flex items-center">
                      <div className="w-full bg-gray-200 h-2 rounded-full mr-3 max-w-[100px]">
                        <div className="bg-brand-teal h-2 rounded-full" style={{ width: `${85 - i*10}%` }}></div>
                      </div>
                      <span className="text-gray-600">{85 - i*10}%</span>
                    </div>
                  </td>
                  <td className="py-4 px-4 text-gray-600 font-medium">{4 - i} days</td>
                  <td className="py-4 px-4">
                    <button className="text-sm text-brand-teal font-medium hover:underline">Message</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
