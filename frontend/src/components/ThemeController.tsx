import { useLayoutEffect, useMemo } from 'react'
import { useTheme } from 'next-themes'
import { useAppearance } from '../api/hooks'
import {
  applyThemeTokens,
  buildThemeTokens,
  DEFAULT_PALETTE_ID,
  findPalette,
  type PaletteData,
  type ThemeMode,
} from '../lib/themes'
import { useWorkspaceStore } from '../store/workspace'

const NO_CUSTOM_PALETTES: PaletteData[] = []

export function ThemeController() {
  const { resolvedTheme } = useTheme()
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: appearance } = useAppearance()
  const customPalettes = appearance?.custom_palettes ?? NO_CUSTOM_PALETTES
  const globalPaletteId = appearance?.global_palette_id ?? DEFAULT_PALETTE_ID
  const paletteId =
    activeWorkspaceId === undefined
      ? globalPaletteId
      : (appearance?.workspace_palettes[String(activeWorkspaceId)] ?? globalPaletteId)
  const palette = useMemo(
    () => findPalette(paletteId, customPalettes),
    [paletteId, customPalettes],
  )
  const mode: ThemeMode = resolvedTheme === 'light' ? 'light' : 'dark'

  useLayoutEffect(() => {
    const tokens = buildThemeTokens(palette, mode)
    applyThemeTokens(tokens)
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', tokens['--background'])
  }, [palette, mode])

  return null
}
