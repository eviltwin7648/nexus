import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-sans)"],
        mono: ["var(--font-mono)"],
      },
      colors: {
        ink: "#0d1015",
        steel: "#2a3445",
        mist: "#eef3f7",
        signal: "#0ea5a4",
        ember: "#f97316",
        pine: "#166534",
      },
      boxShadow: {
        panel: "0 18px 60px rgba(11, 16, 24, 0.12)",
      },
      backgroundImage: {
        grid: "linear-gradient(to right, rgba(37, 56, 88, 0.08) 1px, transparent 1px), linear-gradient(to bottom, rgba(37, 56, 88, 0.08) 1px, transparent 1px)",
      },
      backgroundSize: {
        grid: "32px 32px",
      },
    },
  },
  plugins: [],
};

export default config;
