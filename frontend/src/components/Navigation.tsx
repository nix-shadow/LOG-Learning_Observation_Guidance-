"use client";

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { BookOpen, Home, LineChart, Compass } from 'lucide-react';

export default function Navigation() {
  const pathname = usePathname();

  const navItems = [
    { name: 'Dashboard', path: '/dashboard', icon: Home },
    { name: 'Learning', path: '/learning', icon: BookOpen },
    { name: 'Observation', path: '/observation', icon: LineChart },
    { name: 'Guidance', path: '/guidance', icon: Compass },
  ];

  return (
    <nav className="bg-white shadow-sm border-b border-brand-gray sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link href="/" className="flex-shrink-0 flex items-center space-x-2">
              <span className="text-2xl font-bold text-brand-blue tracking-tight">L<span className="text-brand-teal">O</span>G</span>
            </Link>
          </div>

          <div className="hidden sm:flex sm:space-x-8">
            {navItems.map((item) => {
              const isActive = pathname.startsWith(item.path);
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  href={item.path}
                  className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition-colors ${
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
          </div>
        </div>
      </div>

      {/* Mobile nav */}
      <div className="sm:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-brand-gray pb-safe">
        <div className="flex justify-around py-3">
          {navItems.map((item) => {
            const isActive = pathname.startsWith(item.path);
            const Icon = item.icon;
            return (
              <Link
                key={item.name}
                href={item.path}
                className={`flex flex-col items-center p-2 text-xs font-medium ${
                  isActive ? 'text-brand-teal' : 'text-gray-500 hover:text-gray-900'
                }`}
              >
                <Icon className={`w-6 h-6 mb-1 ${isActive ? 'text-brand-teal' : 'text-gray-400'}`} />
                {item.name}
              </Link>
            );
          })}
        </div>
      </div>
    </nav>
  );
}
