export type Theme = 'light' | 'dark'
export const themeStorageKey = 'creata:theme'

// Apply the approved light palette before paint, preserving a reader's choice.
export const initializeTheme = `(() => {
  let theme = 'light';
  try {
    if (localStorage.getItem(${JSON.stringify(themeStorageKey)}) === 'dark') theme = 'dark';
  } catch {}
  document.documentElement.dataset.theme = theme;
})()`
