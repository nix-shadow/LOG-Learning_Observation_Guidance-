"use client";

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { BookOpen, Home, LineChart, Compass, LogIn, LogOut, ShieldAlert, Library } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';

export default function Navigation() {
  const pathname = usePathname();
  const { user, logout, isAdmin } = useAuth();

  const navItems = [
    { name: 'Dashboard', path: '/dashboard', icon: Home, show: !!user },
    { name: 'Journey', path: '/learning', icon: BookOpen, show: !!user },
    { name: 'Catalog', path: '/courses', icon: Library, show: !!user }, // NEW CATALOG PAGE
    { name: 'Observation', path: '/observation', icon: LineChart, show: !!user },
    { name: 'Guidance', path: '/guidance', icon: Compass, show: !!user },
    { name: 'Admin', path: '/admin', icon: ShieldAlert, show: isAdmin },
  ].filter(item => item.show);

  return (
    <nav className="bg-white shadow-sm border-b border-brand-gray sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link href="/" className="flex-shrink-0 flex items-center space-x-2">
              <span className="text-2xl font-bold text-brand-blue tracking-tight">L<span className="text-brand-teal">O</span>G</span>
            </Link>
          </div>

          <div className="hidden sm:flex items-center space-x-8">
            {navItems.map((item) => {
              const isActive = pathname.startsWith(item.path);
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  href={item.path}
                  className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition-colors h-full ${
                    isActive
                      ? 'border-brand-teal text-brand-teal'
                      : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                  }`}
                >
                  <Icon className="w-4 h-4 mr-2" />
                  {item.name}
                </Link>
              );
            })}

            {user ? (
               <button onClick={logout} className="inline-flex items-center text-sm font-medium text-gray-500 hover:text-red-500 transition-colors">
                  <LogOut className="w-4 h-4 mr-2" /> Logout
               </button>
            ) : (
               <Link href="/login" className="inline-flex items-center text-sm font-medium text-brand-blue hover:text-brand-teal transition-colors">
                  <LogIn className="w-4 h-4 mr-2" /> Login
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
