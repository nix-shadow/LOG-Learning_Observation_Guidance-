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

// WP-1.3: next-intl ships untranspiled ESM that jest's CJS pipeline cannot
// load. Mock it against the REAL en.json messages so component tests keep
// asserting the actual English copy users see — the mock is the dictionary,
// not a fake. Dotted keys (e.g. "cat.connectivity") resolve through nested
// namespaces, matching next-intl's lookup semantics.
jest.mock('next-intl', () => {
  const en = require('@/messages/en.json');
  const resolve = (ns: string, key: string): string => {
    const base = en[ns];
    if (!base) return key;
    let node: unknown = base;
    for (const part of key.split('.')) {
      if (node && typeof node === 'object' && part in (node as Record<string, unknown>)) {
        node = (node as Record<string, unknown>)[part];
      } else {
        return key;
      }
    }
    return typeof node === 'string' ? node : key;
  };
  return {
    useTranslations: (ns: string) => (key: string) => resolve(ns, key),
    NextIntlClientProvider: ({ children }: { children: React.ReactNode }) => children,
  };
});