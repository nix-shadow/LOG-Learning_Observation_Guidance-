"use client";

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { BookOpen, Home, LineChart, Compass, LogIn, LogOut, ShieldAlert, Library, Users, Settings as SettingsIcon } from 'lucide-react';
import { m as motion } from 'framer-motion';
import { useAuth } from '@/context/AuthContext';

export default function Navigation() {
  const pathname = usePathname();
  const { user, logout, isAdmin } = useAuth();
  const isModerator = user?.role === "MODERATOR" || user?.role === "ADMIN";
	// Removed local online/offline state since SyncIsland handles it globally

  const navItems = [
    { name: 'Dashboard', path: '/dashboard', icon: Home, show: !!user },
    { name: 'Journey', path: '/learning', icon: BookOpen, show: !!user },
    { name: 'Catalog', path: '/courses', icon: Library, show: !!user }, // NEW CATALOG PAGE
    { name: 'Observation', path: '/observation', icon: LineChart, show: !!user },
    { name: 'Guidance', path: '/guidance', icon: Compass, show: !!user },
    { name: 'Teacher', path: '/moderator', icon: Users, show: isModerator },
    { name: 'Admin', path: '/admin', icon: ShieldAlert, show: isAdmin },
    { name: 'Settings', path: '/settings', icon: SettingsIcon, show: !!user },
  ].filter(item => item.show);

  return (
    <nav className="bg-brand-dark/80 backdrop-blur-2xl shadow-glow border-b border-white/10 sticky top-0 z-50 @container transition-colors duration-300">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center space-x-4">
            <Link href="/" className="flex-shrink-0 flex items-center space-x-2 group">
              <span className="text-2xl font-bold text-white tracking-tight group-hover:animate-pulse-glow transition-all">
                L<span className="text-brand-neon">O</span>G
              </span>
            </Link>
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
                    className={`inline-flex items-center px-2 pt-1 border-b-2 text-sm font-medium transition-colors h-full ${
                      isActive
                        ? 'border-brand-neon text-brand-neon'
                        : 'border-transparent text-white/50 hover:text-white/90 hover:border-white/20'
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
                  className="inline-flex items-center text-sm font-medium text-white/50 hover:text-red-400 transition-colors"
               >
                  <LogOut className="w-4 h-4 mr-2" /> Logout
               </motion.button>
            ) : (
               <Link href="/login" passHref>
                 <motion.div 
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                    className="inline-flex items-center text-sm font-medium text-white hover:text-brand-neon transition-colors"
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
        <div className="sm:hidden fixed bottom-0 left-0 right-0 bg-brand-darker/90 backdrop-blur-xl border-t border-white/10 pb-safe z-50 overflow-x-auto no-scrollbar shadow-[0_-4px_20px_rgba(0,240,255,0.1)]">
          <div className="flex justify-start sm:justify-around py-3 px-2 min-w-max">
            {navItems.filter(i => i.name !== 'Admin').map((item) => {
              const isActive = pathname.startsWith(item.path);
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  href={item.path}
                  className={`flex flex-col items-center p-2 text-xs font-medium w-16 transition-colors ${
                    isActive ? 'text-brand-neon drop-shadow-[0_0_8px_rgba(0,240,255,0.8)]' : 'text-white/50 hover:text-white/90'
                  }`}
                >
                  <Icon className={`w-6 h-6 mb-1 ${isActive ? 'text-brand-neon drop-shadow-[0_0_8px_rgba(0,240,255,0.8)]' : 'text-brand-faint'}`} />
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
