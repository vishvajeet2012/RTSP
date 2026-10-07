import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: { base: 'var(--background)', surface: 'var(--surface)', edge: 'var(--border)' },
    },
  },
  plugins: [],
} satisfies Config;
