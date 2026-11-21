export type ColorScheme = {
  level0: string
  level1: string
  level2: string
  level3: string
  level4: string
}

// GitHub contribution graph color scheme supporting light and dark themes
export const GITHUB_COLOR_SCHEME: { light: ColorScheme; dark: ColorScheme } = {
  light: {
    level0: 'hsl(210 40% 93%)',
    level1: 'hsl(143 43% 79%)',
    level2: 'hsl(142 52% 51%)',
    level3: 'hsl(142 62% 43%)',
    level4: 'hsl(142 71% 29%)'
  },
  dark: {
    level0: 'hsl(215 28% 17%)',
    level1: 'hsl(142 85% 15%)',
    level2: 'hsl(142 100% 22%)',
    level3: 'hsl(142 71% 45%)',
    level4: 'hsl(142 71% 58%)'
  }
}
