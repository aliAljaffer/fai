import { Moon, Sun } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from '@/components/ui/button'
import { THEME_BACKGROUNDS } from '../lib/palette'

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  const dark = resolvedTheme !== 'light'

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      title={dark ? 'Switch to light theme' : 'Switch to dark theme'}
      aria-label={dark ? 'Switch to light theme' : 'Switch to dark theme'}
      onClick={() => {
        const next = dark ? 'light' : 'dark'
        setTheme(next)
        document
          .querySelector('meta[name="theme-color"]')
          ?.setAttribute('content', THEME_BACKGROUNDS[next])
      }}
    >
      {dark ? <Sun /> : <Moon />}
    </Button>
  )
}
