import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        // Semantic palette driven by CSS variables so light/dark themes swap
        // globally without touching component classes. The `ink` scale keeps its
        // dark-theme *roles* (950 = app background … 100 = primary text); the
        // light theme simply inverts the ramp in globals.css.
        ink: {
          950: "rgb(var(--ink-950) / <alpha-value>)",
          900: "rgb(var(--ink-900) / <alpha-value>)",
          800: "rgb(var(--ink-800) / <alpha-value>)",
          700: "rgb(var(--ink-700) / <alpha-value>)",
          500: "rgb(var(--ink-500) / <alpha-value>)",
          400: "rgb(var(--ink-400) / <alpha-value>)",
          300: "rgb(var(--ink-300) / <alpha-value>)",
          100: "rgb(var(--ink-100) / <alpha-value>)",
        },
        brand: {
          DEFAULT: "rgb(var(--brand) / <alpha-value>)",
          strong: "rgb(var(--brand-strong) / <alpha-value>)",
          soft: "rgb(var(--brand-soft) / <alpha-value>)",
        },
        good: "rgb(var(--good) / <alpha-value>)",
        warn: "rgb(var(--warn) / <alpha-value>)",
        bad: "rgb(var(--bad) / <alpha-value>)",
      },
      borderRadius: {
        xl: "0.9rem",
        "2xl": "1.25rem",
      },
      keyframes: {
        rise: {
          "0%": { transform: "translateY(8px)", opacity: "0" },
          "100%": { transform: "translateY(0)", opacity: "1" },
        },
      },
      animation: {
        rise: "rise 0.3s ease-out",
      },
    },
  },
  plugins: [],
};

export default config;
