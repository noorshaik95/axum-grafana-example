import type { Config } from 'tailwindcss';
import slatePreset from '../shared/tailwind.preset';

const config: Config = {
  presets: [slatePreset as Config],
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    '../shared/components/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        serif: ['Instrument Serif', 'Georgia', 'serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      colors: {
        forest: {
          50: '#f2f7f3',
          100: '#dde9df',
          200: '#b8d2bd',
          300: '#8ab694',
          400: '#5d9a6c',
          500: '#3e7d4f',
          600: '#2d6640',
          700: '#234e32',
          800: '#1a3a26',
          900: '#0f2617',
        },
        cream: '#fbfaf5',
        paper: '#f6f3ec',
        ink: '#12170f',
        muted: '#6a6e62',
        warm: '#d97757',
        amber: {
          DEFAULT: '#ffb648',
          50: '#fff8e6',
          100: '#ffefc0',
          200: '#ffe08a',
          300: '#ffcf54',
          400: '#ffbe2e',
          500: '#ffb648',
          600: '#e09000',
          700: '#b87300',
          800: '#8f5900',
          900: '#664000',
        },
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
};

export default config;
