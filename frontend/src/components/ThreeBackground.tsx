'use client';

import dynamic from 'next/dynamic';
import { useEffect, useState } from 'react';

// The 3D canvas is heavy (~2000 instanced particles) and conflicts with the
// low-connectivity / low-end-device constraint. Three.js is only downloaded and
// rendered when the device can actually handle it:
//   - no prefers-reduced-motion
//   - not a coarse-pointer (touch) device
//   - at least 4 hardware threads
//   - saveData not requested (Android Data Saver / iOS Low Data Mode)
//   - not on a slow effective connection (2g/3g)
const ThreeBackgroundCanvas = dynamic(() => import('./ThreeBackgroundCanvas'), {
  ssr: false,
  loading: () => null,
});

export default function ThreeBackground() {
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const coarse = window.matchMedia('(pointer: coarse)').matches;
    const lowCpu = (navigator.hardwareConcurrency ?? 8) < 4;
    const saveData = !!(navigator as Navigator & { saveData?: boolean }).saveData;
    const conn = (navigator as Navigator & { connection?: { effectiveType?: string } }).connection;
    const slowNetwork = !!conn && ['slow-2g', '2g', '3g'].includes(conn.effectiveType ?? '');
    setEnabled(!reduced && !coarse && !lowCpu && !saveData && !slowNetwork);
  }, []);

  if (!enabled) {
    return null;
  }

  return <ThreeBackgroundCanvas />;
}