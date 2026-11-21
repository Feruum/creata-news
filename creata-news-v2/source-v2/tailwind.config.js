module.exports = {
  content: ["./src/**/*.{js,jsx}"],
  darkMode: ["class", '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        paper: "rgb(var(--paper) / <alpha-value>)", band: "rgb(var(--band) / <alpha-value>)", card: "rgb(var(--card) / <alpha-value>)", ink: "rgb(var(--ink) / <alpha-value>)", muted: "rgb(var(--muted) / <alpha-value>)",
        rule: "rgb(var(--rule) / <alpha-value>)", accent: "rgb(var(--accent) / <alpha-value>)", "accent-ink": "rgb(var(--accent-ink) / <alpha-value>)", soft: "rgb(var(--soft) / <alpha-value>)", onaccent: "rgb(var(--onaccent) / <alpha-value>)"
      },
      fontFamily: {
        head: ['"Golos Text"', "system-ui", "sans-serif"],
        logo: ["Unbounded", '"Golos Text"', "sans-serif"],
        mono: ['"IBM Plex Mono"', "ui-monospace", "monospace"],
        serif: ['"PT Serif"', "Georgia", "serif"]
      },
      maxWidth: { site: "1360px" }
    }
  },
  plugins: []
};
