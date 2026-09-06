import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        // Semantic palette echoing the transactional-email design.
        ink: {
          950: "#0b1120",
          900: "#0f172a",
          800: "#1e293b",
          700: "#334155",
          500: "#64748b",
          400: "#94a3b8",
          300: "#cbd5e1",
          100: "#e2e8f0",
        },
        brand: {
          DEFAULT: "#38bdf8",
          strong: "#0ea5e9",
          soft: "#7dd3fc",
        },
        good: "#34d399",
        warn: "#fbbf24",
        bad: "#f87171",
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
