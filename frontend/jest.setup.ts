import '@testing-library/jest-dom';
import React from 'react';

// jsdom provides crypto.getRandomValues but NOT crypto.subtle (WebCrypto
// primitives). Node's webcrypto.subtle is a drop-in for the queue-encryption
// tests (Node v22 ships stable crypto.subtle). jsdom's crypto getter is
// non-configurable, so defineProperty must target the property, not the object.
Object.defineProperty(globalThis.crypto, 'subtle', {
  writable: true,
  configurable: true,
  value: require('crypto').webcrypto.subtle,
});

// jsdom also lacks TextEncoder/TextDecoder; Node's util versions are identical.
const { TextEncoder, TextDecoder } = require('util');
Object.assign(globalThis, { TextEncoder, TextDecoder });

// framer-motion's exit animations never resolve under jsdom, which hangs
// AnimatePresence mode="wait". Render children synchronously instead.
// WP-0.3 LazyMotion sweep: the app imports the `m` proxy (identical API to
// `motion`), so the mock must provide it too.
const motionProxy = new Proxy(
  {},
  {
    get: (_target, prop: string) =>
      ({ children, ...props }: { children?: React.ReactNode; [key: string]: unknown }) =>
        React.createElement(prop as string, props, children),
  }
);
jest.mock('framer-motion', () => ({
  motion: motionProxy,
  m: motionProxy,
  useReducedMotion: () => false,
  AnimatePresence: ({ children }: { children: React.ReactNode }) => children,
}));