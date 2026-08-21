"use client";

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import {
  BookOpen, Home, LineChart, Compass, LogIn, LogOut, ShieldAlert, Library,
  Users, Settings as SettingsIcon, Languages, LifeBuoy, HeartHandshake, ChevronDown,
} from 'lucide-react';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { useAuth } from '@/context/AuthContext';
import { useTranslations } from 'next-intl';
import { useLocaleCtx } from '@/i18n/LocaleProvider';
import ThemeToggle from '@/components/ThemeToggle';

function initialsOf(name?: string) {
  const parts = (name || '').trim().split(/\s+/).filter(Boolean);
  return (parts.slice(0, 2).map(w => w[0]!.toUpperCase()).join('')) || 'U';
}

interface AccountUser {
  name?: string;
  email?: string;
  role?: string;
}

/**
 * Avatar disclosure menu (APG disclosure pattern — real links, not role=menu).
 * Identity header → personal tools → role-scoped surfaces → Log out LAST.
 * Self-contained state/refs so the desktop and mobile instances stay independent.
 */
function AccountMenu({
  user, logout, pathname, reduceMotion, langToggle, logoutLabel,
}: {
  user: AccountUser;
  logout: () => void;
  pathname: string;
  reduceMotion: boolean;
  langToggle: React.ReactNode;
  logoutLabel: string;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  useEffect(() => { setOpen(false); }, [pathname]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    document.addEventListener('pointerdown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [open]);

  const roleChip =
    user.role === 'ADMIN'
      ? 'bg-brand-amber/15 text-brand-amber'
      : user.role === 'MODERATOR'
        ? 'bg-brand-teal/15 text-brand-teal'
        : 'bg-brand-blue/10 text-brand-blue';

  const staffLinks = [
    ...(user.role === 'PARENT' ? [{ name: 'Parent Portal', path: '/parent', icon: HeartHandshake }] : []),
    ...(user.role === 'MODERATOR' || user.role === 'ADMIN' ? [{ name: 'Moderator', path: '/moderator', icon: Users }] : []),
    ...(user.role === 'ADMIN' ? [{ name: 'Admin', path: '/admin', icon: ShieldAlert }] : []),
  ];

  const itemCls = (active: boolean) =>
    `flex min-h-[44px] items-center gap-3 rounded-xl px-3 text-sm font-medium transition-colors ${
      active ? 'bg-brand-blue/10 text-brand-blue' : 'text-brand-muted hover:bg-brand-blue/10 hover:text-brand-text'
    }`;

  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        onClick={() => setOpen(o => !o)}
        aria-expanded={open}
        aria-haspopup="true"
        aria-label="Account menu"
        data-testid="account-menu-trigger"
        className={`flex items-center gap-1 rounded-full p-0.5 pr-1 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-blue ${
          open ? 'bg-brand-blue/10' : 'hover:bg-brand-blue/10'
        }`}
      >
        <span className="flex h-10 w-10 select-none items-center justify-center rounded-full bg-brand-blue text-sm font-bold text-white">
          {initialsOf(user.name)}
        </span>
        <ChevronDown className={`h-4 w-4 text-brand-muted transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            key="account-menu"
            initial={{ opacity: 0, scale: reduceMotion ? 1 : 0.96, y: reduceMotion ? 0 : -4 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: reduceMotion ? 1 : 0.96, transition: { duration: 0.12 } }}
            transition={{ duration: 0.16, ease: 'easeOut' }}
            style={{ transformOrigin: 'top right' }}
            className="absolute right-0 top-full z-[60] mt-2 w-72 overflow-hidden rounded-2xl border border-brand-blue/15 bg-brand-dark shadow-bento"
            role="group"
            aria-label="Account"
            data-testid="account-menu"
          >
            <div className="flex items-center gap-3 border-b border-brand-blue/10 px-4 py-3.5">
              <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-brand-blue text-base font-bold text-white">
                {initialsOf(user.name)}
              </span>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-white">{user.name}</p>
                <p className="truncate text-xs text-brand-muted">{user.email}</p>
                {user.role && (
                  <span className={`mt-1 inline-block rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide ${roleChip}`}>
                    {user.role}
                  </span>
                )}
              </div>
            </div>

            <div className="p-1.5">
              <Link href="/settings" className={itemCls(pathname.startsWith('/settings'))} aria-current={pathname.startsWith('/settings') ? 'page' : undefined}>
                <SettingsIcon className="h-4 w-4 shrink-0" />
                Settings
              </Link>
              <Link href="/support" className={itemCls(pathname.startsWith('/support'))} aria-current={pathname.startsWith('/support') ? 'page' : undefined}>
                <LifeBuoy className="h-4 w-4 shrink-0" />
                Support
              </Link>
            </div>

            {staffLinks.length > 0 && (
              <>
                <div className="mx-4 border-t border-brand-blue/10" />
                <div className="p-1.5">
                  {staffLinks.map(item => (
                    <Link key={item.path} href={item.path} className={itemCls(pathname.startsWith(item.path))} aria-current={pathname.startsWith(item.path) ? 'page' : undefined}>
                      <item.icon className="h-4 w-4 shrink-0" />
                      {item.name}
                    </Link>
                  ))}
                </div>
              </>
            )}

            {/* Phone-only quick controls — keeps the mobile top bar clean */}
            <div className="flex items-center justify-between gap-2 border-t border-brand-blue/10 px-4 py-3 md:hidden">
              {langToggle}
              <ThemeToggle />
            </div>

            <div className="border-t border-brand-blue/10 p-1.5">
              <button
                onClick={logout}
                className="flex min-h-[44px] w-full items-center gap-3 rounded-xl px-3 text-sm font-medium text-red-500 transition-colors hover:bg-red-500/10"
              >
                <LogOut className="h-4 w-4 shrink-0" />
                {logoutLabel}
              </button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export default function Navigation() {
  const pathname = usePathname();
  const t = useTranslations('nav');
  const { locale, setLocale } = useLocaleCtx();
  const { user, logout } = useAuth();
  const reduceMotion = useReducedMotion();

  // Learner-first primary destinations (LMS convention: ≤5 top-level links;
  // everything personal or staff-scoped lives in the account menu).
  const primaryItems = user
    ? [
        { name: t('dashboard'), path: '/dashboard', icon: Home },
        { name: t('learning'), path: '/learning', icon: BookOpen },
        { name: 'Catalog', path: '/courses', icon: Library },
        { name: t('observation'), path: '/observation', icon: LineChart },
        { name: 'Guidance', path: '/guidance', icon: Compass },
      ]
    : [];

  const langToggle = (
    <button
      onClick={() => setLocale(locale === 'en' ? 'np' : 'en')}
      aria-label="Switch language / भाषा बदल्नुहोस्"
      title="Switch language"
      className="inline-flex items-center gap-1 text-xs font-bold text-brand-muted hover:text-brand-blue border border-brand-blue/20 hover:border-brand-blue/40 rounded-full px-2.5 py-1.5 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-blue"
    >
      <Languages className="w-3.5 h-3.5" />
      <span className="hidden lg:inline">{locale === 'en' ? 'नेपाली' : 'EN'}</span>
    </button>
  );

  // NOTE: the mobile bottom bar renders OUTSIDE <nav> — backdrop-filter on
  // the nav would otherwise become the containing block for position:fixed
  // and pin the bar under the header instead of the screen bottom.
  return (
    <>
    <nav aria-label="Primary" className="sticky top-0 z-50 border-b border-brand-blue/10 bg-brand-dark/90 backdrop-blur-md transition-colors duration-300">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Desktop: logo | truly-centered primary links | utilities.
            Equal 1fr side columns keep the link group dead-center even
            though the utility cluster is wider than the logo. */}
        <div className="hidden h-16 grid-cols-[1fr_auto_1fr] items-center gap-6 md:grid">
          <Link href="/" className="flex shrink-0 items-center justify-self-start" aria-label="LOG home">
            <span className="text-2xl font-bold tracking-tight text-white">
              L<span className="text-brand-neon">O</span>G
            </span>
          </Link>

          <div className="flex items-center justify-center gap-1">
            {primaryItems.map((item) => {
              const isActive = pathname.startsWith(item.path);
              return (
                <Link
                  key={item.path}
                  href={item.path}
                  aria-current={isActive ? 'page' : undefined}
                  title={item.name}
                  className={`inline-flex min-h-[36px] items-center gap-2 rounded-full px-4 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-blue ${
                    isActive
                      ? 'bg-brand-blue/15 text-brand-blue'
                      : 'text-brand-muted hover:bg-brand-blue/10 hover:text-brand-text'
                  }`}
                >
                  <item.icon className="h-4 w-4 shrink-0" />
                  <span className="whitespace-nowrap">{item.name}</span>
                </Link>
              );
            })}
          </div>

          <div className="flex shrink-0 items-center gap-2 justify-self-end">
            {langToggle}
            <ThemeToggle />
            {user ? (
              <AccountMenu
                user={{ name: user.name, email: user.email, role: user.role }}
                logout={logout}
                pathname={pathname}
                reduceMotion={!!reduceMotion}
                langToggle={langToggle}
                logoutLabel={t('logout')}
              />
            ) : (
              <Link
                href="/login"
                className="inline-flex items-center gap-1.5 rounded-full bg-brand-blue px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-brand-blue/90 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-blue"
              >
                <LogIn className="h-4 w-4" />
                {t('login')}
              </Link>
            )}
          </div>
        </div>

        {/* Mobile top row: logo left, session right (utilities live in the menu) */}
        <div className="flex h-14 items-center justify-between md:hidden">
          <Link href="/" className="flex shrink-0 items-center" aria-label="LOG home">
            <span className="text-xl font-bold tracking-tight text-white">
              L<span className="text-brand-neon">O</span>G
            </span>
          </Link>
          <div className="flex items-center gap-2">
            {user ? (
              <AccountMenu
                user={{ name: user.name, email: user.email, role: user.role }}
                logout={logout}
                pathname={pathname}
                reduceMotion={!!reduceMotion}
                langToggle={langToggle}
                logoutLabel={t('logout')}
              />
            ) : (
              <>
                {langToggle}
                <ThemeToggle />
                <Link href="/login" aria-label={t('login')} className="rounded-full bg-brand-blue p-2 text-white transition-colors hover:bg-brand-blue/90">
                  <LogIn className="h-5 w-5" />
                </Link>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>

    {/* Mobile bottom bar — 5 peer destinations, MD3-style active pill.
        Sits outside <nav> so position:fixed tracks the viewport. */}
    {user && (
        <div className="fixed bottom-0 left-0 right-0 z-40 border-t border-brand-blue/10 bg-brand-dark/95 backdrop-blur-md md:hidden">
          <div className="flex items-stretch justify-around px-1 pb-[max(0.25rem,env(safe-area-inset-bottom))] pt-1">
            {primaryItems.map((item) => {
              const isActive = pathname.startsWith(item.path);
              return (
                <Link
                  key={item.path}
                  href={item.path}
                  aria-current={isActive ? 'page' : undefined}
                  aria-label={item.name}
                  className={`flex min-h-[48px] min-w-0 flex-1 flex-col items-center justify-center gap-0.5 rounded-xl px-1 py-1.5 transition-colors ${
                    isActive ? 'bg-brand-blue/10 text-brand-blue' : 'text-brand-muted active:bg-brand-blue/5'
                  }`}
                >
                  <item.icon className="h-5 w-5 shrink-0" />
                  <span className="w-full truncate px-0.5 text-center text-[10px] font-medium">{item.name}</span>
                </Link>
              );
            })}
          </div>
        </div>
      )}
    </>
  );
}
