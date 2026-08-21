import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Routes that require authentication
const PROTECTED_ROUTES = [
  '/dashboard',
  '/learning',
  '/courses',
  '/observation',
  '/guidance',
  '/moderator',
  '/admin',
  '/settings',
  // WP-2.1: parent portal — a PARENT token is required (students have no
  // learner identity there and /dashboard 404s for parents).
  '/parent',
  // WP-2.2: support funnel is available to every authenticated role.
  '/support',
];

// Routes that should redirect authenticated users away (login/register pages)
const AUTH_ROUTES = ['/login', '/forgot-password'];

/**
 * Decodes the JWT `role` claim without verifying the signature (the Go backend
 * is the source of truth for validity). Used only for edge routing decisions.
 */
function decodeRole(token: string): string | null {
  try {
    const payload = token.split('.')[1];
    if (!payload) return null;
    const json = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return typeof json.role === 'string' ? json.role : null;
  } catch {
    return null;
  }
}

/**
 * Next.js Edge Middleware — runs before every request.
 * Checks for the presence of a JWT token and redirects unauthenticated
 * users away from protected routes at the edge (no content flash).
 * Role claims are decoded here too so a STUDENT token cannot read
 * /admin or /moderator HTML; the Go backend enforces roles authoritatively.
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Check if the request is for a protected route
  const isProtectedRoute = PROTECTED_ROUTES.some(route => pathname.startsWith(route));
  const isAuthRoute = AUTH_ROUTES.some(route => pathname.startsWith(route));

  // We check for the token in cookies (set by AuthContext)
  const token = request.cookies.get('log_token')?.value;

  if (isProtectedRoute && !token) {
    // Redirect unauthenticated users to login
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  if (token) {
    const role = decodeRole(token) ?? '';
    if (pathname.startsWith('/admin') && role !== 'ADMIN') {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
    if (pathname.startsWith('/moderator') && role !== 'ADMIN' && role !== 'MODERATOR') {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
    if (pathname.startsWith('/parent') && role !== 'PARENT') {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
  }

  if (isAuthRoute && token) {
    // Redirect authenticated users away from login pages
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }

  return NextResponse.next();
}

export const config = {
  // Match all routes except static files, API routes, and Next.js internals
  matcher: [
    '/((?!api|_next/static|_next/image|favicon.ico|manifest.json|assets|sw.js|workbox-*).*)',
  ],
};