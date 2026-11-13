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

export function labelColorFor(name: string) {
  let hash = 0
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return LABEL_COLORS[hash % LABEL_COLORS.length]
}

// The backend stores this grey for a label created without a colour.
const UNSET_LABEL_COLOR = '#888888'

export function tagColor(tag: { name: string; color?: string | null }) {
  const unset = !tag.color || tag.color.toLowerCase() === UNSET_LABEL_COLOR
  return unset ? labelColorFor(tag.name) : tag.color!
}

export function labelChipStyle(color?: string | null) {
  if (!color) return undefined
  return {
    backgroundColor: `color-mix(in oklch, ${color} 14%, transparent)`,
    borderColor: `color-mix(in oklch, ${color} 32%, transparent)`,
  }
}
