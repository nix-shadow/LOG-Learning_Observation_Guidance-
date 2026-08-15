import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          blue: "hsl(var(--brand-blue) / <alpha-value>)",
          teal: "hsl(var(--brand-teal) / <alpha-value>)",
          amber: "hsl(var(--brand-amber) / <alpha-value>)",
          white: "hsl(var(--brand-white) / <alpha-value>)",
          gray: "hsl(var(--brand-gray) / <alpha-value>)",
          text: "hsl(var(--brand-text) / <alpha-value>)"
        }
      },
      fontFamily: {
        sans: ['var(--font-inter)', 'sans-serif'],
      },
      boxShadow: {
        'bento': '0 0 0 1px rgba(0, 0, 0, 0.05), 0 4px 6px -1px rgba(0, 0, 0, 0.05)',
        'glow': '0 0 20px rgba(0, 180, 216, 0.15)',
      }
    },
  },
  plugins: [
    require('@tailwindcss/container-queries'),
  ],
};
export default config;
