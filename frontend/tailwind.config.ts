import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        brand: {
          blue: "hsl(var(--brand-blue) / <alpha-value>)",
          teal: "hsl(var(--brand-teal) / <alpha-value>)",
          amber: "hsl(var(--brand-amber) / <alpha-value>)",
          white: "hsl(var(--brand-white) / <alpha-value>)",
          gray: "hsl(var(--brand-gray) / <alpha-value>)",
          text: "hsl(var(--brand-text) / <alpha-value>)",
          dark: "hsl(var(--brand-dark) / <alpha-value>)",
          darker: "hsl(var(--brand-darker) / <alpha-value>)",
          neon: "hsl(var(--brand-neon) / <alpha-value>)",
          muted: "hsl(var(--brand-muted) / <alpha-value>)",
          faint: "hsl(var(--brand-faint) / <alpha-value>)",
        }
      },
      fontFamily: {
        sans: ['var(--font-inter)', 'var(--font-devanagari)', 'sans-serif'],
      },
      boxShadow: {
        'bento': '0 0 0 1px rgba(15, 23, 42, 0.06), 0 4px 6px -1px rgba(15, 23, 42, 0.12)',
        // WP-4.6: glows are subtle elevation halos now — never neon blooms,
        // never hover-triggered (see AGENTS.md §2a).
        'glow': '0 2px 12px -2px rgb(var(--glow-rgb) / 0.15)',
        'glow-strong': '0 4px 20px -4px rgb(var(--glow-rgb) / 0.25)',
      },
      backgroundImage: {
        'glass-gradient': 'linear-gradient(135deg, rgba(255, 255, 255, 0.03) 0%, rgba(255, 255, 255, 0.01) 100%)',
        'neon-gradient': 'linear-gradient(90deg, #2563EB, #0D9488, #F59E0B)',
      },
      animation: {
        'spin-slow': 'spin 8s linear infinite',
        'gradient-x': 'gradient-x 3s ease infinite',
        'shimmer': 'shimmer 2s linear infinite',
        'pulse-glow': 'pulse-glow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        'gradient-x': {
          '0%, 100%': {
            'background-size': '200% 200%',
            'background-position': 'left center'
          },
          '50%': {
            'background-size': '200% 200%',
            'background-position': 'right center'
          },
        },
        'shimmer': {
          'from': {
            'backgroundPosition': '200% 0'
          },
          'to': {
            'backgroundPosition': '-200% 0'
          }
        },
        'pulse-glow': {
          '0%, 100%': { opacity: '1', boxShadow: '0 2px 12px -2px rgb(var(--glow-rgb) / 0.15)' },
          '50%': { opacity: '.75', boxShadow: '0 1px 8px -2px rgb(var(--glow-rgb) / 0.05)' },
        }
      }
    },
  },
  plugins: [
    require('@tailwindcss/container-queries'),
  ],
};
export default config;
