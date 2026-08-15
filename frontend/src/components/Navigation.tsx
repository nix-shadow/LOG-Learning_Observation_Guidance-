"use client";

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { BookOpen, Home, LineChart, Compass, LogIn, LogOut, ShieldAlert, Library, Users, Wifi, WifiOff } from 'lucide-react';
import { motion } from 'framer-motion';
import { useAuth } from '@/context/AuthContext';
import { useSyncQueue } from '@/hooks/useSyncQueue';

export default function Navigation() {
  const pathname = usePathname();
  const { user, logout, isAdmin } = useAuth();
  const isModerator = user?.role === "MODERATOR" || user?.role === "ADMIN";
  const [isOnline, setIsOnline] = useState(true);
  const { pendingCount } = useSyncQueue();

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setIsOnline(navigator.onLine);
      const handleOnline = () => setIsOnline(true);
      const handleOffline = () => setIsOnline(false);
      window.addEventListener('online', handleOnline);
      window.addEventListener('offline', handleOffline);
      return () => {
        window.removeEventListener('online', handleOnline);
        window.removeEventListener('offline', handleOffline);
      };
    }
  }, []);

  const navItems = [
    { name: 'Dashboard', path: '/dashboard', icon: Home, show: !!user },
    { name: 'Journey', path: '/learning', icon: BookOpen, show: !!user },
    { name: 'Catalog', path: '/courses', icon: Library, show: !!user }, // NEW CATALOG PAGE
    { name: 'Observation', path: '/observation', icon: LineChart, show: !!user },
    { name: 'Guidance', path: '/guidance', icon: Compass, show: !!user },
    { name: 'Teacher', path: '/moderator', icon: Users, show: isModerator },
    { name: 'Admin', path: '/admin', icon: ShieldAlert, show: isAdmin },
  ].filter(item => item.show);

  return (
    <nav className="bg-white/80 backdrop-blur-md shadow-sm border-b border-brand-gray sticky top-0 z-50 @container">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center space-x-4">
            <Link href="/" className="flex-shrink-0 flex items-center space-x-2">
              <span className="text-2xl font-bold text-brand-blue tracking-tight">L<span className="text-brand-teal">O</span>G</span>
            </Link>

            {/* Offline/Online Status Pill */}
            <div className={`hidden md:flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${
              isOnline ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-amber-50 text-amber-800 border border-amber-300'
            }`}>
              {isOnline ? (
                <>
                  <Wifi className="w-3.5 h-3.5 mr-1 text-emerald-600" />
                  <span>Online</span>
                </>
              ) : (
                <>
                  <WifiOff className="w-3.5 h-3.5 mr-1 text-amber-600" />
                  <span>Offline {pendingCount > 0 && `(${pendingCount} pending)`}</span>
                </>
              )}
            </div>
          </div>

          <div className="hidden sm:flex items-center space-x-8">
            {navItems.map((item) => {
              const isActive = pathname.startsWith(item.path);
              const Icon = item.icon;
              return (
                <Link key={item.name} href={item.path} passHref>
                  <motion.div
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                    transition={{ type: "spring", stiffness: 400, damping: 17 }}
                    className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition-colors h-full ${
                      isActive
                        ? 'border-brand-teal text-brand-teal'
                        : 'border-transparent text-gray-500 hover:text-gray-900'
                    }`}
                  >
                    <Icon className="w-4 h-4 mr-2" />
                    {item.name}
                  </motion.div>
                </Link>
              );
            })}

            {user ? (
               <motion.button 
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={logout} 
                  className="inline-flex items-center text-sm font-medium text-gray-500 hover:text-red-500 transition-colors"
               >
                  <LogOut className="w-4 h-4 mr-2" /> Logout
               </motion.button>
            ) : (
               <Link href="/login" passHref>
                 <motion.div 
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                    className="inline-flex items-center text-sm font-medium text-brand-blue hover:text-brand-teal transition-colors"
                 >
                    <LogIn className="w-4 h-4 mr-2" /> Login
                 </motion.div>
               </Link>
            )}
          </div>
        </div>
      </div>

      {/* Mobile nav - only show if logged in */}
      {user && (
        <div className="sm:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-brand-gray pb-safe z-50 overflow-x-auto no-scrollbar">
          <div className="flex justify-start sm:justify-around py-3 px-2 min-w-max">
            {navItems.filter(i => i.name !== 'Admin').map((item) => {
              const isActive = pathname.startsWith(item.path);
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  href={item.path}
                  className={`flex flex-col items-center p-2 text-xs font-medium w-16 ${
                    isActive ? 'text-brand-teal' : 'text-gray-500 hover:text-gray-900'
                  }`}
                >
                  <Icon className={`w-6 h-6 mb-1 ${isActive ? 'text-brand-teal' : 'text-gray-400'}`} />
                  <span className="truncate w-full text-center">{item.name}</span>
                </Link>
              );
            })}
          </div>
        </div>
      )}
    </nav>
  );
}
