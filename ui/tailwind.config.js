/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Deep space backgrounds
        'void': {
          950: '#050508',
          900: '#080810',
          850: '#0a0a14',
          800: '#0c0c18',
          750: '#10101f',
          700: '#141426',
          600: '#1a1a30',
        },
        // Bioluminescent cyan accent
        'glow': {
          50: '#eafffd',
          100: '#cbfff9',
          200: '#9dfffa',
          300: '#5cfffc',
          400: '#00ffd5',
          500: '#00e4c4',
          600: '#00baa3',
          700: '#009485',
          800: '#00756b',
          900: '#006058',
        },
        // Warm accent for warnings/active states
        'ember': {
          400: '#ff9f43',
          500: '#ff7f11',
          600: '#e66a00',
        },
        // Status colors
        'status': {
          success: '#00ffa3',
          warning: '#ffd000',
          error: '#ff4757',
          info: '#00d4ff',
        }
      },
      fontFamily: {
        'display': ['Sora', 'system-ui', 'sans-serif'],
        'mono': ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      backgroundImage: {
        'noise': "url(\"data:image/svg+xml,%3Csvg viewBox='0 0 400 400' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)'/%3E%3C/svg%3E\")",
        'grid-pattern': 'linear-gradient(to right, rgba(0, 255, 213, 0.03) 1px, transparent 1px), linear-gradient(to bottom, rgba(0, 255, 213, 0.03) 1px, transparent 1px)',
        'glow-radial': 'radial-gradient(ellipse at center, rgba(0, 255, 213, 0.15) 0%, transparent 70%)',
      },
      backgroundSize: {
        'grid': '24px 24px',
      },
      boxShadow: {
        'glow': '0 0 20px rgba(0, 255, 213, 0.3)',
        'glow-sm': '0 0 10px rgba(0, 255, 213, 0.2)',
        'glow-lg': '0 0 40px rgba(0, 255, 213, 0.4)',
        'inner-glow': 'inset 0 0 20px rgba(0, 255, 213, 0.1)',
      },
      animation: {
        'pulse-glow': 'pulse-glow 2s ease-in-out infinite',
        'fade-in': 'fade-in 0.3s ease-out',
        'slide-in': 'slide-in 0.3s ease-out',
        'slide-up': 'slide-up 0.4s ease-out',
        'scan': 'scan 3s linear infinite',
      },
      keyframes: {
        'pulse-glow': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.5' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-in': {
          '0%': { transform: 'translateX(20px)', opacity: '0' },
          '100%': { transform: 'translateX(0)', opacity: '1' },
        },
        'slide-up': {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        'scan': {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100%)' },
        },
      },
    },
  },
  plugins: [],
}
