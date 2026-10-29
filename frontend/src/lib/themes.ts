export type ThemeMode = 'light' | 'dark'

export interface PaletteData {
  id: string
  name: string
  colors: string[]
}

export interface ThemePalette extends PaletteData {
  custom?: boolean
}

export const DEFAULT_PALETTE_ID = 'light-steel'

const RAW_PALETTES: Array<{ name: string; colors: string[] }> = [
  { name: 'Light Steel', colors: ['F8F9FA', 'E9ECEF', 'DEE2E6', 'CED4DA', 'ADB5BD', '6C757D', '495057', '343A40', '212529'] },
  { name: 'Slate', colors: ['F8FAFC', 'E2E8F0', '94A3B8', '475569', '0F172A'] },
  { name: 'Fiery Ocean', colors: ['780000', 'C1121F', 'FDF0D5', '003049', '669BBC'] },
  { name: 'Olive Garden Feast', colors: ['606C38', '283618', 'FEFAE0', 'DDA15E', 'BC6C25'] },
  { name: 'Ocean Blue Serenity', colors: ['03045E', '023E8A', '0077B6', '0096C7', '00B4D8', '48CAE4', '90E0EF', 'ADE8F4', 'CAF0F8'] },
  { name: 'Pastel Dreamland Adventure', colors: ['CDB4DB', 'FFC8DD', 'FFAFCC', 'BDE0FE', 'A2D2FF'] },
  { name: 'Dark Sunset', colors: ['335C67', 'FFF3B0', 'E09F3E', '9E2A2B', '540B0E'] },
  { name: 'Autumn Harvest', colors: ['6F1D1B', 'BB9457', '432818', '99582A', 'FFE6A7'] },
  { name: 'Fresh Greens', colors: ['386641', '6A994E', 'A7C957', 'F2E8CF', 'BC4749'] },
  { name: 'Leafy Green Garden', colors: ['132A13', '31572C', '4F772D', '90A955', 'ECF39E'] },
  { name: 'Sunny Beach Day', colors: ['264653', '2A9D8F', 'E9C46A', 'F4A261', 'E76F51'] },
  { name: 'Refreshing Summer Fun', colors: ['8ECAE6', '219EBC', '023047', 'FFB703', 'FB8500'] },
  { name: 'Golden Summer Fields', colors: ['CCD5AE', 'E9EDC9', 'FEFAE0', 'FAEDCD', 'D4A373'] },
  { name: 'Black & Gold Elegance', colors: ['000000', '14213D', 'FCA311', 'E5E5E5', 'FFFFFF'] },
  { name: 'Ocean Sunset', colors: ['001219', '005F73', '0A9396', '94D2BD', 'E9D8A6', 'EE9B00', 'CA6702', 'BB3E03', 'AE2012', '9B2226'] },
  { name: 'Deep Sea', colors: ['0D1321', '1D2D44', '3E5C76', '748CAB', 'F0EBD8'] },
  { name: 'Purple Sunset', colors: ['390099', '9E0059', 'FF0054', 'FF5400', 'FFBD00'] },
  { name: 'Summer Ocean Breeze', colors: ['E63946', 'F1FAEE', 'A8DADC', '457B9D', '1D3557'] },
  { name: 'Peachy Delight', colors: ['D8E2DC', 'FFE5D9', 'FFCAD4', 'F4ACB7', '9D8189'] },
  { name: 'Ocean Breeze', colors: ['03045E', '0077B6', '00B4D8', '90E0EF', 'CAF0F8'] },
  { name: 'Warm Autumn Glow', colors: ['003049', 'D62828', 'F77F00', 'FCBF49', 'EAE2B7'] },
  { name: 'Pastel Comfort', colors: ['D4E09B', 'F6F4D2', 'CBDFBD', 'F19C79', 'A44A3F'] },
  { name: 'Rose Petal', colors: ['880D1E', 'DD2D4A', 'F26A8D', 'F49CBB', 'CBEEF3'] },
  { name: 'Earthy Tones', colors: ['F0EAD2', 'DDE5B6', 'ADC178', 'A98467', '6C584C'] },
  { name: 'Fiery Palette', colors: ['5F0F40', '9A031E', 'FB8B24', 'E36414', '0F4C5C'] },
  { name: 'Cherry Blossom Bloom', colors: ['590D22', '800F2F', 'A4133C', 'C9184A', 'FF4D6D', 'FF758F', 'FF8FA3', 'FFB3C1', 'FFCCD5', 'FFF0F3'] },
  { name: 'Golden Twilight', colors: ['000814', '001D3D', '003566', 'FFC300', 'FFD60A'] },
  { name: 'Soft Peachy Delight', colors: ['FEC5BB', 'FCD5CE', 'FAE1DD', 'F8EDEB', 'E8E8E4', 'D8E2DC', 'ECE4DB', 'FFE5D9', 'FFD7BA', 'FEC89A'] },
  { name: 'Midnight Sky', colors: ['00296B', '003F88', '00509D', 'FDC500', 'FFD500'] },
  { name: 'Earthy Forest Hues', colors: ['DAD7CD', 'A3B18A', '588157', '3A5A40', '344E41'] },
  { name: 'Warm Neutral Tones', colors: ['582F0E', '7F4F24', '936639', 'A68A64', 'B6AD90', 'C2C5AA', 'A4AC86', '656D4A', '414833', '333D29'] },
  { name: 'Fiery Red Sunset', colors: ['03071E', '370617', '6A040F', '9D0208', 'D00000', 'DC2F02', 'E85D04', 'F48C06', 'FAA307', 'FFBA08'] },
  { name: 'Summer Dream', colors: ['0081A7', '00AFB9', 'FDFCDC', 'FED9B7', 'F07167'] },
  { name: 'Meadow Green', colors: ['D9ED92', 'B5E48C', '99D98C', '76C893', '52B69A', '34A0A4', '168AAD', '1A759F', '1E6091', '184E77'] },
  { name: 'Vivid Nightfall', colors: ['10002B', '240046', '3C096C', '5A189A', '7B2CBF', '9D4EDD', 'C77DFF', 'E0AAFF'] },
  { name: 'Crimson Hues', colors: ['250902', '38040E', '640D14', '800E13', 'AD2831'] },
  { name: 'Purple Dream', colors: ['231942', '5E548E', '9F86C0', 'BE95C4', 'E0B1CB'] },
  { name: 'Pastel Dreams', colors: ['FF99C8', 'FCF6BD', 'D0F4DE', 'A9DEF9', 'E4C1F9'] },
  { name: 'Warm Earth Tones', colors: ['8C1C13', 'BF4342', 'E7D7C1', 'A78A7F', '735751'] },
  { name: 'Soft Lavender', colors: ['8E9AAF', 'CBC0D3', 'EFD3D7', 'FEEAFA', 'DEE2FF'] },
  { name: 'Neutral Elegance', colors: ['463F3A', '8A817C', 'BCB8B1', 'F4F3EE', 'E0AFA0'] },
  { name: 'Vibrant Sunset', colors: ['FF6D00', 'FF7900', 'FF8500', 'FF9100', 'FF9E00', '240046', '3C096C', '5A189A', '7B2CBF', '9D4EDD'] },
  { name: 'Purple Raindrops', colors: ['F72585', 'B5179E', '7209B7', '560BAD', '480CA8', '3A0CA3', '3F37C9', '4361EE', '4895EF', '4CC9F0'] },
  { name: 'Pastel Harmony', colors: ['D88C9A', 'F2D0A9', 'F1E3D3', '99C1B9', '8E7DBE'] },
  { name: 'Golden Summer Glow', colors: ['0D3B66', 'FAF0CA', 'F4D35E', 'EE964B', 'F95738'] },
  { name: 'Colorful Daybreak', colors: ['3D348B', '7678ED', 'F7B801', 'F18701', 'F35B04'] },
  { name: 'Deep Blue Waters', colors: ['2F6690', '3A7CA5', 'D9DCD6', '16425B', '81C3D7'] }
]

function slug(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')
}

export const BUILT_IN_PALETTES: ThemePalette[] = RAW_PALETTES.map((palette) => ({
  ...palette,
  colors: palette.colors.map((color) => '#' + color),
  id: slug(palette.name),
}))

export function allPalettes(customPalettes: PaletteData[]): ThemePalette[] {
  return [
    ...BUILT_IN_PALETTES,
    ...customPalettes.map((palette) => ({ ...palette, custom: true })),
  ]
}

// A saved palette can disappear when a custom theme is deleted, so fall back to
// Light Steel and then to the first built-in if Light Steel is unavailable.
export function findPalette(id: string, customPalettes: PaletteData[] = []): ThemePalette {
  const builtIn = BUILT_IN_PALETTES.find((palette) => palette.id === id)
  if (builtIn) return builtIn
  const custom = customPalettes.find((palette) => palette.id === id)
  if (custom) return { ...custom, custom: true }
  return (
    BUILT_IN_PALETTES.find((palette) => palette.id === DEFAULT_PALETTE_ID) ??
    BUILT_IN_PALETTES[0]
  )
}

export function customPaletteId(name: string) {
  return `custom-${slug(name) || 'palette'}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
}

function normalize(hex: string) {
  const value = hex.replace('#', '').trim()
  return '#' + (value.length === 3 ? value.split('').map((char) => char + char).join('') : value)
}

function hexToRgb(hex: string) {
  const value = normalize(hex).slice(1)
  return {
    r: parseInt(value.slice(0, 2), 16),
    g: parseInt(value.slice(2, 4), 16),
    b: parseInt(value.slice(4, 6), 16),
  }
}

function rgbToHex(r: number, g: number, b: number) {
  const part = (value: number) => Math.round(clamp(value, 0, 255)).toString(16).padStart(2, '0')
  return '#' + part(r) + part(g) + part(b)
}

function rgbToHsl(hex: string) {
  const { r, g, b } = hexToRgb(hex)
  const rn = r / 255
  const gn = g / 255
  const bn = b / 255
  const max = Math.max(rn, gn, bn)
  const min = Math.min(rn, gn, bn)
  const l = (max + min) / 2
  let h = 0
  let s = 0
  if (max !== min) {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    if (max === rn) h = ((gn - bn) / d + (gn < bn ? 6 : 0)) * 60
    else if (max === gn) h = ((bn - rn) / d + 2) * 60
    else h = ((rn - gn) / d + 4) * 60
  }
  return { h, s, l }
}

function hslToHex(h: number, s: number, l: number) {
  const hue = ((h % 360) + 360) % 360
  const sat = clamp(s)
  const light = clamp(l)
  const c = (1 - Math.abs(2 * light - 1)) * sat
  const x = c * (1 - Math.abs(((hue / 60) % 2) - 1))
  const m = light - c / 2
  let r = 0
  let g = 0
  let b = 0
  if (hue < 60) [r, g, b] = [c, x, 0]
  else if (hue < 120) [r, g, b] = [x, c, 0]
  else if (hue < 180) [r, g, b] = [0, c, x]
  else if (hue < 240) [r, g, b] = [0, x, c]
  else if (hue < 300) [r, g, b] = [x, 0, c]
  else [r, g, b] = [c, 0, x]
  return rgbToHex((r + m) * 255, (g + m) * 255, (b + m) * 255)
}

function mix(a: string, b: string, amount: number) {
  const ca = hexToRgb(a)
  const cb = hexToRgb(b)
  const t = clamp(amount)
  return rgbToHex(
    ca.r + (cb.r - ca.r) * t,
    ca.g + (cb.g - ca.g) * t,
    ca.b + (cb.b - ca.b) * t,
  )
}

function luminance(hex: string) {
  const { r, g, b } = hexToRgb(hex)
  const channel = (value: number) => {
    const v = value / 255
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

export function contrast(a: string, b: string) {
  const l1 = luminance(a)
  const l2 = luminance(b)
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)
}

function ensureContrast(hex: string, background: string, target: number) {
  const base = normalize(hex)
  if (contrast(base, background) >= target) return base
  const { h, s, l } = rgbToHsl(base)
  const lighten = luminance(background) < 0.42
  let lo = lighten ? l : 0
  let hi = lighten ? 1 : l
  let best = lighten ? '#ffffff' : '#000000'
  for (let i = 0; i < 24; i++) {
    const mid = (lo + hi) / 2
    const candidate = hslToHex(h, s, mid)
    if (contrast(candidate, background) >= target) {
      best = candidate
      if (lighten) hi = mid
      else lo = mid
    } else if (lighten) {
      lo = mid
    } else {
      hi = mid
    }
  }
  return best
}

function bestOn(background: string) {
  return contrast('#ffffff', background) >= contrast('#111315', background)
    ? '#ffffff'
    : '#111315'
}

function hueDistance(a: number, b: number) {
  const d = Math.abs(a - b) % 360
  return d > 180 ? 360 - d : d
}

function pickPrimary(colors: string[]) {
  let best = colors[0]
  let bestScore = -1
  for (const color of colors) {
    const { s, l } = rgbToHsl(color)
    const mid = 1 - Math.min(1, Math.abs(l - 0.48) / 0.58)
    let score = s * 0.55 + mid * 0.45
    if (l < 0.14) score -= 0.35
    if (l > 0.86) score -= 0.15
    if (score > bestScore) {
      bestScore = score
      best = color
    }
  }
  return best
}

function pickAccent(colors: string[], primary: string) {
  const primaryHue = rgbToHsl(primary).h
  let best = primary
  let bestScore = -1
  for (const color of colors) {
    if (normalize(color) === normalize(primary)) continue
    const { h, s, l } = rgbToHsl(color)
    if (l < 0.08) continue
    const mid = 1 - Math.min(1, Math.abs(l - 0.52) / 0.56)
    let score = s * 0.35 + (hueDistance(h, primaryHue) / 180) * 0.2 + mid * 0.45
    if (l < 0.18) score -= 0.2
    if (l > 0.9) score -= 0.1
    if (score > bestScore) {
      bestScore = score
      best = color
    }
  }
  return best
}

const SEMANTIC: Record<ThemeMode, Record<'todo' | 'doing' | 'done' | 'danger' | 'warning', string>> = {
  light: {
    todo: '#5b84ae',
    doing: '#0f9293',
    done: '#519160',
    danger: '#b3261e',
    warning: '#a16207',
  },
  dark: {
    todo: '#8fb3d6',
    doing: '#45b8b8',
    done: '#78b98b',
    danger: '#ff8a80',
    warning: '#ffd166',
  },
}

function clamp(value: number, min = 0, max = 1) {
  return Math.min(max, Math.max(min, value))
}

function semanticColor(
  name: 'todo' | 'doing' | 'done' | 'danger' | 'warning',
  surface: string,
  primary: string,
  mode: ThemeMode,
) {
  return ensureContrast(mix(SEMANTIC[mode][name], primary, 0.12), surface, 4.5)
}

export function buildThemeTokens(palette: ThemePalette, mode: ThemeMode) {
  const colors = palette.colors.map(normalize)
  const sorted = [...colors].sort((a, b) => luminance(a) - luminance(b))
  const darkest = sorted[0]
  const lightest = sorted[sorted.length - 1]
  const primaryRaw = pickPrimary(colors)
  const accentRaw = pickAccent(colors, primaryRaw)
  const primaryHue = rgbToHsl(primaryRaw).h

  let background: string
  let card: string
  let soft: string
  let foreground: string
  let mutedForeground: string
  let border: string
  let primary: string
  let primaryText: string
  let accent: string
  let surface: string

  if (mode === 'light') {
    background = mix('#ffffff', lightest, 0.16)
    card = mix('#ffffff', lightest, 0.045)
    soft = mix(background, darkest, 0.035)
    foreground = ensureContrast(mix(darkest, '#0b0d10', 0.28), background, 13)
    mutedForeground = ensureContrast(mix(background, darkest, 0.55), background, 4.5)
    border = mix(background, darkest, 0.14)
    primary = ensureContrast(primaryRaw, background, 4.5)
    primaryText = ensureContrast(primaryRaw, background, 4.5)
    accent = ensureContrast(accentRaw, background, 4.5)
    surface = background
  } else {
    const seed = rgbToHsl(primaryRaw)
    background = hslToHex(primaryHue, clamp(seed.s * 0.46, 0.025, 0.16), 0.09)
    card = hslToHex(primaryHue, clamp(seed.s * 0.36, 0.02, 0.125), 0.135)
    soft = hslToHex(primaryHue, clamp(seed.s * 0.3, 0.02, 0.105), 0.19)
    foreground = ensureContrast(mix(lightest, '#ffffff', 0.35), background, 13)
    mutedForeground = ensureContrast(mix(lightest, background, 0.58), background, 4.8)
    border = mix(background, '#ffffff', 0.12)
    primary = ensureContrast(primaryRaw, card, 4.5)
    primaryText = ensureContrast(primaryRaw, background, 4.5)
    accent = ensureContrast(accentRaw, card, 4.5)
    surface = card
  }

  const onPrimary = bestOn(primary)
  const danger = semanticColor('danger', surface, primaryRaw, mode)
  const warning = semanticColor('warning', surface, primaryRaw, mode)
  const todo = semanticColor('todo', surface, primaryRaw, mode)
  const doing = semanticColor('doing', surface, primaryRaw, mode)
  const done = semanticColor('done', surface, primaryRaw, mode)

  return {
    '--background': background,
    '--foreground': foreground,
    '--card': card,
    '--card-foreground': foreground,
    '--popover': card,
    '--popover-foreground': foreground,
    '--primary': primary,
    '--primary-foreground': onPrimary,
    '--secondary': soft,
    '--secondary-foreground': foreground,
    '--muted': soft,
    '--muted-foreground': mutedForeground,
    '--accent': soft,
    '--accent-foreground': foreground,
    '--destructive': danger,
    '--destructive-foreground': bestOn(danger),
    '--attention': warning,
    '--attention-foreground': bestOn(warning),
    '--code-string': done,
    '--code-number': warning,
    '--border': border,
    '--input': border,
    '--ring': primaryText,
    '--chart-1': primary,
    '--chart-2': accent,
    '--status-todo': todo,
    '--status-doing': doing,
    '--status-done': done,
    '--brand-tile': primary,
    '--brand-ink': onPrimary,
    '--sidebar': soft,
    '--sidebar-foreground': foreground,
    '--sidebar-primary': primary,
    '--sidebar-primary-foreground': onPrimary,
    '--sidebar-accent': soft,
    '--sidebar-accent-foreground': foreground,
    '--sidebar-border': border,
    '--sidebar-ring': primaryText,
  }
}

export function applyThemeTokens(tokens: Record<string, string>) {
  const root = document.documentElement
  for (const [name, value] of Object.entries(tokens)) {
    root.style.setProperty(name, value)
  }
}
