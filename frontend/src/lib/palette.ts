export const LABEL_COLORS = [
  '#588DC3',
  '#239AA6',
  '#4C9C74',
  '#879045',
  '#B07F32',
  '#C26F5B',
  '#BB6C92',
  '#927BBD',
]

export const STATUS_COLORS = {
  todo: '#5B84AE',
  doing: '#0F9293',
  done: '#519160',
}

export const THEME_BACKGROUNDS = {
  light: '#FAFCFE',
  dark: '#0D1012',
}

export function labelColorFor(name: string) {
  let hash = 0
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return LABEL_COLORS[hash % LABEL_COLORS.length]
}

export function labelChipStyle(color?: string | null) {
  if (!color) return undefined
  return {
    backgroundColor: `color-mix(in oklch, ${color} 14%, transparent)`,
    borderColor: `color-mix(in oklch, ${color} 32%, transparent)`,
  }
}
