import type { Config } from 'tailwindcss'
import slatePreset from '../shared/tailwind.preset'

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
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
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
        slate_muted: '#6a6e62',
        warm: '#d97757',
        slate_amber: '#ffb648',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}

export default config
