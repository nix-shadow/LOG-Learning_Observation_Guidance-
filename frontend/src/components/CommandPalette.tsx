'use client';

import { useState, useEffect } from 'react';
import { Command } from 'cmdk';
import { Search, LayoutDashboard, Settings, Activity, BookOpen, User } from 'lucide-react';
import { useRouter } from 'next/navigation';

export default function CommandPalette() {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  // Toggle the menu when ⌘K is pressed
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener('keydown', down);
    return () => document.removeEventListener('keydown', down);
  }, []);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[100] bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
      {/* 
        Clicking the overlay closes the palette. 
        Using a simple div overlay here; for a full accessible modal, Dialog is better, 
        but this keeps it lightweight.
      */}
      <div className="absolute inset-0" onClick={() => setOpen(false)} />
      
      <div className="relative w-full max-w-xl bg-brand-dark/90 border border-white/10 rounded-2xl shadow-glow-strong overflow-hidden flex flex-col">
        <Command label="Global Command Menu" className="w-full">
          <div className="flex items-center border-b border-white/10 px-4 py-3">
            <Search className="w-5 h-5 text-brand-neon mr-3" />
            <Command.Input 
              autoFocus
              placeholder="Type a command or search..." 
              className="w-full bg-transparent outline-none text-white placeholder-white/40 text-lg"
            />
            <div className="text-xs text-white/40 bg-white/10 px-2 py-1 rounded">ESC</div>
          </div>
          
          <Command.List className="max-h-[300px] overflow-y-auto p-2 scrollbar-hide">
            <Command.Empty className="py-6 text-center text-white/50 text-sm">
              No results found.
            </Command.Empty>
            
            <Command.Group heading="Navigation" className="px-2 py-2 text-xs font-semibold text-brand-teal uppercase tracking-wider">
              <Command.Item 
                onSelect={() => { router.push('/dashboard'); setOpen(false); }}
                className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
              >
                <LayoutDashboard className="w-4 h-4 mr-3 text-white/70" />
                <span className="text-white">Dashboard</span>
              </Command.Item>
              <Command.Item 
                onSelect={() => { router.push('/learning-journey'); setOpen(false); }}
                className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
              >
                <Activity className="w-4 h-4 mr-3 text-white/70" />
                <span className="text-white">Learning Journey</span>
              </Command.Item>
              <Command.Item 
                onSelect={() => { router.push('/courses'); setOpen(false); }}
                className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
              >
                <BookOpen className="w-4 h-4 mr-3 text-white/70" />
                <span className="text-white">Course Catalog</span>
              </Command.Item>
            </Command.Group>
            
            <Command.Group heading="Settings" className="px-2 py-2 text-xs font-semibold text-brand-teal uppercase tracking-wider">
              <Command.Item 
                onSelect={() => { /* Handle action */ setOpen(false); }}
                className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
              >
                <User className="w-4 h-4 mr-3 text-white/70" />
                <span className="text-white">Profile Settings</span>
              </Command.Item>
              <Command.Item 
                onSelect={() => { /* Force sync action */ setOpen(false); }}
                className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
              >
                <Settings className="w-4 h-4 mr-3 text-white/70" />
                <span className="text-white">Force Data Sync</span>
              </Command.Item>
            </Command.Group>
          </Command.List>
        </Command>
      </div>
    </div>
  );
}
