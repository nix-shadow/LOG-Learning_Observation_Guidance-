import withPWAInit from 'next-pwa';

import defaultCache from 'next-pwa/cache.js';

// Research round (WP-0.3 bundle): prune dead service-worker runtime rules —
// every rule kept is one less match the SW evaluates on each request:
//   - google-fonts-webfonts / google-fonts-stylesheets: Inter is self-hosted
//     by next/font at build time — no cross-origin fonts are ever requested
//   - next-image: images.unoptimized:true — there is no /_next/image optimizer
//   - the cross-origin catch-all (NetworkFirst on `.*`) that cached API
//     responses across sessions (existing filter, kept)
const DEAD_CACHE_NAMES = ['google-fonts-webfonts', 'google-fonts-stylesheets', 'next-image'];

const runtimeCaching = defaultCache.filter((cache) => {
  const name = cache.options?.cacheName;
  if (name && DEAD_CACHE_NAMES.includes(name)) return false;
  if (typeof cache.urlPattern === 'string' && cache.urlPattern === '.*' && cache.handler === 'NetworkFirst') return false;
  return true;
});

const withPWA = withPWAInit({
  dest: 'public',
  disable: process.env.NODE_ENV === 'development',
  register: true,
  skipWaiting: true,
  runtimeCaching,
  // Research round: offline navigation fell to the browser's "no internet"
  // error page. navigateFallback serves our honest ~offline page instead.
  navigateFallback: '/~offline',
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  images: {
    unoptimized: true,
  }
};

export default withPWA(nextConfig);