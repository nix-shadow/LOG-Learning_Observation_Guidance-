import '@testing-library/jest-dom';
import React from 'react';

// canvas-confetti is used by the lesson page; jsdom has no canvas.
jest.mock('canvas-confetti', () => jest.fn());

// framer-motion's exit animations never resolve under jsdom, which hangs
// AnimatePresence mode="wait". Render children synchronously instead.
jest.mock('framer-motion', () => ({
  motion: new Proxy(
    {},
    {
      get: (_target, prop: string) =>
        ({ children, ...props }: { children?: React.ReactNode; [key: string]: unknown }) =>
          React.createElement(prop as string, props, children),
    }
  ),
  AnimatePresence: ({ children }: { children: React.ReactNode }) => children,
}));