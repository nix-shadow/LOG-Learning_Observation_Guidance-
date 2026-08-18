import React from 'react';

// Minimal stand-in for next/image so components render in jsdom without
// the Next.js image pipeline.
const NextImage = (props: Record<string, unknown>) => {
  return React.createElement('img', props);
};

export default NextImage;