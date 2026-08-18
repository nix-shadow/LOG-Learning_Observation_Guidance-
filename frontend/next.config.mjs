import withPWAInit from 'next-pwa';

import defaultCache from 'next-pwa/cache.js';

// Filter out the cross-origin catch-all that caches API responses across sessions
const runtimeCaching = defaultCache.filter(
  (cache) => !(typeof cache.urlPattern === 'string' && cache.urlPattern === '.*' && cache.handler === 'NetworkFirst')
);

const withPWA = withPWAInit({
  dest: 'public',
  disable: process.env.NODE_ENV === 'development',
  register: true,
  skipWaiting: true,
  runtimeCaching,
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
