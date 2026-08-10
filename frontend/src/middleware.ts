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
];

// Routes that should redirect authenticated users away (login/register pages)
const AUTH_ROUTES = ['/login', '/forgot-password'];

/**
 * Next.js Edge Middleware — runs before every request.
 * Checks for the presence of a JWT token and redirects unauthenticated
 * users away from protected routes at the edge (no content flash).
 *
 * Note: We only check token *presence* here, not validity.
 * Full JWT validation happens server-side in the Go backend.
 * The AuthContext handles client-side expiry checking.
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
