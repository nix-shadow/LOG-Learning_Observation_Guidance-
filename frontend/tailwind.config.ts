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
        sans: ['var(--font-inter)', 'sans-serif'],
      },
      boxShadow: {
        'bento': '0 0 0 1px rgba(255, 255, 255, 0.05), 0 4px 6px -1px rgba(0, 0, 0, 0.2)',
        'glow': '0 0 20px rgba(0, 240, 255, 0.15)',
        'glow-strong': '0 0 40px rgba(0, 240, 255, 0.4)',
      },
      backgroundImage: {
        'glass-gradient': 'linear-gradient(135deg, rgba(255, 255, 255, 0.03) 0%, rgba(255, 255, 255, 0.01) 100%)',
        'neon-gradient': 'linear-gradient(90deg, #00B4D8, #7000FF, #FF0070)',
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
          '0%, 100%': { opacity: '1', boxShadow: '0 0 20px rgba(0, 240, 255, 0.15)' },
          '50%': { opacity: '.5', boxShadow: '0 0 10px rgba(0, 240, 255, 0.05)' },
        }
      }
    },
  },
  plugins: [
    require('@tailwindcss/container-queries'),
  ],
};
export default config;
