import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-sans)", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"],
      },
      colors: {
        // Acento neon lima (inspirado nas referências)
        lime: {
          DEFAULT: "#D7FF3A",
          50: "#F8FFE0",
          100: "#EFFFB3",
          200: "#E6FF80",
          300: "#DDFF5C",
          400: "#D7FF3A",
          500: "#C2EB1F",
          600: "#9CC00F",
          700: "#73900A",
          800: "#4A5D06",
          900: "#232C03",
        },
        // Superfícies escuras
        ink: {
          950: "#000000",
          900: "#0A0A0B",
          850: "#111113",
          800: "#161618",
          750: "#1C1C1F",
          700: "#232327",
          600: "#2C2C31",
          500: "#3A3A40",
          400: "#5A5A62",
          300: "#8A8A93",
          200: "#B5B5BC",
          100: "#E4E4E7",
        },
      },
      borderRadius: {
        "4xl": "2rem",
        "5xl": "2.5rem",
      },
      boxShadow: {
        lime: "0 0 0 1px rgba(215,255,58,0.25), 0 10px 40px -10px rgba(215,255,58,0.45)",
        glow: "0 0 24px rgba(215,255,58,0.35)",
        card: "0 1px 0 0 rgba(255,255,255,0.04) inset, 0 20px 40px -20px rgba(0,0,0,0.8)",
      },
      keyframes: {
        "fade-up": {
          "0%": { opacity: "0", transform: "translateY(8px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
      animation: {
        "fade-up": "fade-up .45s cubic-bezier(.2,.8,.2,1) both",
      },
    },
  },
  plugins: [],
};
export default config;
