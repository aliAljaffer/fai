import { useSyncExternalStore } from 'react'

const reducedMotionQuery = '(prefers-reduced-motion: reduce)'

function subscribeReducedMotion(notify: () => void) {
  const media = window.matchMedia(reducedMotionQuery)
  media.addEventListener('change', notify)
  return () => media.removeEventListener('change', notify)
}

export function useViewTransition() {
  return !useSyncExternalStore(
    subscribeReducedMotion,
    () => window.matchMedia(reducedMotionQuery).matches,
    () => false,
  )
}
