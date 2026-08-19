'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { Command } from 'cmdk';
import { Search, LayoutDashboard, Settings, Activity, BookOpen } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { flushSyncQueue } from '@/lib/api';
import toast from 'react-hot-toast';

// WP-0.3 a11y research round: the palette used to be ⌘K-only (keyboard-only
// trigger — mouse users could never open it) and rendered a bare overlay with
// no dialog semantics, no focus trap, no ESC, and no focus return. Now:
//   - a visible pointer trigger (Search button) opens it on any device
//   - role="dialog" aria-modal="true" + aria-label
//   - Tab is trapped inside, ESC closes, focus returns to the trigger
export default function CommandPalette() {
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const dialogRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const lastFocused = useRef<HTMLElement | null>(null);
  // WP-0.3 a11y research round: cmdk filters as the user types; the result
  // count is mirrored into a polite live region so AT users hear whether the
  // search matched, instead of the dialog silently narrowing.
  const [query, setQuery] = useState('');
  const [resultsCount, setResultsCount] = useState<number | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const closePalette = useCallback(() => {
    setOpen(false);
    // Restore focus to wherever the user was (WCAG 2.4.3 focus order).
    if (lastFocused.current && document.contains(lastFocused.current)) {
      lastFocused.current.focus();
    }
  }, []);

  const openPalette = useCallback(() => {
    lastFocused.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setOpen(true);
  }, []);

  // Toggle the menu when ⌘K is pressed
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        if (open) closePalette();
        else openPalette();
      }
    };
    document.addEventListener('keydown', down);
    return () => document.removeEventListener('keydown', down);
  }, [open, closePalette, openPalette]);

  // Focus the input when the dialog opens, and trap Tab inside it.
  useEffect(() => {
    if (!open) return;
    // Delay so the DOM is committed before focusing.
    const t = setTimeout(() => inputRef.current?.focus(), 0);
    const trap = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        closePalette();
        return;
      }
      if (e.key !== 'Tab' || !dialogRef.current) return;
      const focusables = Array.from(
        dialogRef.current.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
      ).filter((el) => !el.hasAttribute('disabled'));
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && (active === first || !dialogRef.current.contains(active))) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener('keydown', trap);
    return () => {
      clearTimeout(t);
      document.removeEventListener('keydown', trap);
    };
  }, [open, closePalette]);

  // Count the cmdk-visible items and mirror the number into the live region.
  useEffect(() => {
    if (!open) return;
    const list = listRef.current;
    if (!list) return;
    const count = () => {
      const visible = Array.from(list.querySelectorAll('[cmdk-item]')).filter(
        (el) => !el.hasAttribute('aria-hidden') && !el.hasAttribute('hidden')
      ).length;
      setResultsCount(visible);
    };
    count();
    const mo = new MutationObserver(count);
    mo.observe(list, { childList: true, subtree: true, attributes: true, attributeFilter: ['aria-hidden', 'hidden'] });
    return () => mo.disconnect();
  }, [open]);

  // F9: "Force Data Sync" was a dead entry (no-op) and the journey route was
  // wrong (/learning-journey does not exist — the real route is /learning).
  const handleForceSync = async () => {
    closePalette();
    toast('Checking for saved offline changes...', { icon: '📡' });
    const result = await flushSyncQueue();
    if (result.synced > 0) {
      toast.success(`${result.synced} offline change${result.synced > 1 ? 's' : ''} synced!`);
    } else if (result.failed > 0) {
      toast.error(`${result.failed} change${result.failed > 1 ? 's' : ''} could not be synced.`);
    } else {
      toast('No pending changes — everything is up to date.', { icon: '✓' });
    }
  };

  const go = (path: string) => {
    router.push(path);
    closePalette();
  };

  return (
    <>
      {/* Pointer trigger (research round): previously the palette was only
          reachable via ⌘K, which mouse-only users cannot produce. */}
      <button
        onClick={openPalette}
        aria-label="Open command menu (⌘K)"
        aria-haspopup="dialog"
        aria-expanded={open}
        title="Command menu (⌘K)"
        className="fixed bottom-24 sm:bottom-6 right-4 sm:right-6 z-[90] flex items-center gap-2 px-4 py-3 bg-brand-dark/80 backdrop-blur-2xl border border-white/10 rounded-full shadow-glow-strong text-brand-muted hover:text-white hover:border-brand-teal/50 transition-colors"
      >
        <Search className="w-4 h-4" />
        <span className="hidden sm:inline text-sm font-medium">Menu</span>
        <span className="hidden md:inline text-xs text-brand-faint border border-white/10 rounded px-1.5 py-0.5">⌘K</span>
      </button>

      {open && (
        <div
          className="fixed inset-0 z-[100] bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
          role="dialog"
          aria-modal="true"
          aria-label="Global command menu"
        >
          {/* Clicking the overlay closes the palette. */}
          <div className="absolute inset-0" onClick={closePalette} />

          <div
            ref={dialogRef}
            className="relative w-full max-w-xl bg-brand-dark/90 border border-white/10 rounded-2xl shadow-glow-strong overflow-hidden flex flex-col"
          >
            <Command label="Global Command Menu" className="w-full">
              <div className="flex items-center border-b border-white/10 px-4 py-3">
                <Search className="w-5 h-5 text-brand-neon mr-3" />
                <Command.Input
                  ref={inputRef}
                  value={query}
                  onValueChange={setQuery}
                  placeholder="Type a command or search..."
                  className="w-full bg-transparent outline-none text-white placeholder-white/40 text-lg"
                />
                <div className="text-xs text-brand-muted bg-white/10 px-2 py-1 rounded">ESC</div>
              </div>

              <Command.List ref={listRef} className="max-h-[300px] overflow-y-auto p-2 scrollbar-hide">
                <Command.Empty className="py-6 text-center text-white/50 text-sm">
                  No results found.
                </Command.Empty>

                <Command.Group heading="Navigation" className="px-2 py-2 text-xs font-semibold text-brand-teal uppercase tracking-wider">
                  <Command.Item
                    onSelect={() => go('/dashboard')}
                    className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
                  >
                    <LayoutDashboard className="w-4 h-4 mr-3 text-white/70" />
                    <span className="text-white">Dashboard</span>
                  </Command.Item>
                  <Command.Item
                    onSelect={() => go('/learning')}
                    className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
                  >
                    <Activity className="w-4 h-4 mr-3 text-white/70" />
                    <span className="text-white">Learning Journey</span>
                  </Command.Item>
                  <Command.Item
                    onSelect={() => go('/courses')}
                    className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
                  >
                    <BookOpen className="w-4 h-4 mr-3 text-white/70" />
                    <span className="text-white">Course Catalog</span>
                  </Command.Item>
                </Command.Group>

                <Command.Group heading="Settings" className="px-2 py-2 text-xs font-semibold text-brand-teal uppercase tracking-wider">
                  <Command.Item
                    onSelect={handleForceSync}
                    className="flex items-center px-3 py-3 mt-1 rounded-xl cursor-pointer hover:bg-white/10 aria-selected:bg-white/10 transition-colors"
                  >
                    <Settings className="w-4 h-4 mr-3 text-white/70" />
                    <span className="text-white">Force Data Sync</span>
                  </Command.Item>
                </Command.Group>
              </Command.List>

              {/* Polite live region: announces filter results to screen readers
                  (the visible list narrowing is not otherwise announced). */}
              <span className="sr-only" role="status">
                {query.trim().length > 0
                  ? resultsCount === 0
                    ? 'No results found.'
                    : `${resultsCount} result${resultsCount === 1 ? '' : 's'}.`
                  : ''}
              </span>
            </Command>
          </div>
        </div>
      )}
    </>
  );
}