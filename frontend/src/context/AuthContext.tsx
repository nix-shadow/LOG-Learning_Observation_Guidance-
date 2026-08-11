"use client";

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { logout as serverLogout } from '@/lib/api';
import toast from 'react-hot-toast';

interface User {
  id: string;
  name: string;
  email: string;
  phone: string;
  role: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (userData: User, token: string) => void;
  logout: () => void;
  isAdmin: boolean;
  isModerator: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

/**
 * Decodes a JWT payload without external libraries.
 * Returns null if decoding fails.
 */
function decodeJWTPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = parts[1];
    // Base64url decode
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const jsonStr = atob(base64);
    return JSON.parse(jsonStr);
  } catch {
    return null;
  }
}

/**
 * Checks if a JWT token has expired.
 * Returns true if the token is expired or invalid.
 */
function isTokenExpired(token: string): boolean {
  const payload = decodeJWTPayload(token);
  if (!payload || !payload.exp) return true;
  // Add 30-second buffer to account for clock skew
  return Date.now() >= ((payload.exp as number) * 1000) - 30000;
}

/**
 * Sets a cookie accessible to Next.js middleware (httpOnly cannot be set from JS,
 * but this is sufficient for edge middleware to check token presence).
 */
function setTokenCookie(token: string | null) {
  if (typeof document === 'undefined') return;
  if (token) {
    // SameSite=Lax for CSRF protection, Secure in production
    const secure = window.location.protocol === 'https:' ? '; Secure' : '';
    document.cookie = `log_token=${token}; path=/; SameSite=Lax; max-age=${72 * 60 * 60}${secure}`;
  } else {
    document.cookie = 'log_token=; path=/; max-age=0';
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);

  const logout = useCallback(() => {
    setUser(null);
    setToken(null);
    // Clear the auth cookie so the middleware doesn't bounce /login -> /dashboard.
    setTokenCookie(null);
    // Full lifecycle: revoke JWT server-side, clear localStorage + cookie,
    // wipe the IndexedDB cache (so the next user sees no stale data), redirect.
    serverLogout().catch(() => {
      // serverLogout already handles cleanup; ignore edge failures.
    });
  }, []);

  useEffect(() => {
    // Check for stored session on mount
    const storedToken = localStorage.getItem('log_token');
    const storedUser = localStorage.getItem('log_user');

    if (storedToken && storedUser) {
      // Validate token hasn't expired before restoring session
      if (isTokenExpired(storedToken)) {
        toast('Your session has expired. Please log in again.', { icon: '🔒' });
        logout();
        return;
      }

      setToken(storedToken);
      setUser(JSON.parse(storedUser));
      setTokenCookie(storedToken);
    }
  }, [logout]);

  // Periodic expiry check — runs every 60 seconds
  useEffect(() => {
    if (!token) return;

    const interval = setInterval(() => {
      if (isTokenExpired(token)) {
        toast('Your session has expired. Please log in again.', { icon: '🔒' });
        logout();
      }
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [token, logout]);

  const login = (userData: User, jwt: string) => {
    setUser(userData);
    setToken(jwt);
    localStorage.setItem('log_token', jwt);
    localStorage.setItem('log_user', JSON.stringify(userData));
    setTokenCookie(jwt);
    window.location.href = '/dashboard';
  };

  const isAdmin = user?.role === 'ADMIN';
  const isModerator = user?.role === 'MODERATOR' || isAdmin;

  return (
    <AuthContext.Provider value={{ user, token, login, logout, isAdmin, isModerator }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
