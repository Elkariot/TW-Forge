export const THEMES = {
  papyrus: {
    label: 'Папирус',
    vars: {
      '--color-primary': '#C8A850', '--color-primary-hover': '#A88030',
      '--color-secondary': '#DCC880', '--color-cell': '#EED898',
      '--color-accent': '#8B6020', '--color-heading': '#5A3808',
      '--color-bg': '#E8D090', '--color-text': '#2A1800',
      '--color-text-dim': '#6A4820', '--color-text-muted': '#9A7840',
      '--color-border': '#C0A050', '--color-border-soft': '#B09040',
      '--color-input-bg': '#F4E8B0', '--color-error': '#8B2010',
    },
  },
  spring: {
    label: 'Весенний',
    vars: {
      '--color-primary': '#faace1', '--color-primary-hover': '#e090c8',
      '--color-secondary': '#f7daee', '--color-cell': '#f0ecc5',
      '--color-accent': '#95e8e1', '--color-heading': '#3aaa9e',
      '--color-bg': '#fdf5f9', '--color-text': '#3a2035',
      '--color-text-dim': '#9a7080', '--color-text-muted': '#b090a0',
      '--color-border': '#e8c8da', '--color-border-soft': '#ddb8cc',
      '--color-input-bg': '#fff8fb', '--color-error': '#c03060',
    },
  },
  ocean: {
    label: 'Морской',
    vars: {
      '--color-primary': '#7ab8f5', '--color-primary-hover': '#5a98d5',
      '--color-secondary': '#d4eaff', '--color-cell': '#e8f4ff',
      '--color-accent': '#40c8e0', '--color-heading': '#2090b0',
      '--color-bg': '#f0f8ff', '--color-text': '#1a3050',
      '--color-text-dim': '#5070a0', '--color-text-muted': '#8090b0',
      '--color-border': '#b8d8f0', '--color-border-soft': '#90b8d8',
      '--color-input-bg': '#f8fcff', '--color-error': '#c03050',
    },
  },
  forest: {
    label: 'Лесной',
    vars: {
      '--color-primary': '#98d878', '--color-primary-hover': '#78b858',
      '--color-secondary': '#d8f0c8', '--color-cell': '#f0f8e8',
      '--color-accent': '#60c840', '--color-heading': '#3a8028',
      '--color-bg': '#f4faf0', '--color-text': '#1e3818',
      '--color-text-dim': '#5a7850', '--color-text-muted': '#8a9880',
      '--color-border': '#b8dca0', '--color-border-soft': '#98c080',
      '--color-input-bg': '#f8fcf4', '--color-error': '#c03040',
    },
  },
  dark: {
    label: 'Темница',
    vars: {
      '--color-primary': '#7A5018', '--color-primary-hover': '#5A3808',
      '--color-secondary': '#221C10', '--color-cell': '#2A2214',
      '--color-accent': '#C49030', '--color-heading': '#D4A840',
      '--color-bg': '#181208', '--color-text': '#E0D09A',
      '--color-text-dim': '#9A8058', '--color-text-muted': '#685838',
      '--color-border': '#382A14', '--color-border-soft': '#2C2010',
      '--color-input-bg': '#1C1608', '--color-error': '#C04820',
    },
  },
}

export function applyTheme(key) {
  const theme = THEMES[key]
  if (!theme) return key
  for (const [k, v] of Object.entries(theme.vars))
    document.documentElement.style.setProperty(k, v)
  localStorage.setItem('tw-theme', key)
  return key
}

export function initTheme() {
  const saved = localStorage.getItem('tw-theme')
  return applyTheme(saved && THEMES[saved] ? saved : 'papyrus')
}
