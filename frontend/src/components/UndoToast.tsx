import { useCallback, useEffect, useRef } from 'react'
import { toast } from 'sonner'

const UNDO_DURATION_MS = 8000

export function UndoToast({
  toastId,
  message,
  onUndo,
}: {
  toastId: string | number
  message: string
  onUndo: () => void
}) {
  const remaining = useRef(UNDO_DURATION_MS)
  const startedAt = useRef(0)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const paused = useRef(false)
  const container = useRef<HTMLDivElement>(null)

  const clearTimer = useCallback(() => {
    if (timer.current !== null) {
      clearTimeout(timer.current)
      timer.current = null
    }
  }, [])

  const armTimer = useCallback(() => {
    startedAt.current = Date.now()
    timer.current = setTimeout(() => toast.dismiss(toastId), remaining.current)
  }, [toastId])

  function pause() {
    if (paused.current) return
    paused.current = true
    clearTimer()
    remaining.current = Math.max(0, remaining.current - (Date.now() - startedAt.current))
  }

  function resume() {
    if (!paused.current) return
    paused.current = false
    armTimer()
  }

  useEffect(() => {
    armTimer()
    return clearTimer
  }, [armTimer, clearTimer])

  useEffect(() => {
    function keepClear(element: Element | null) {
      const toast = container.current
      if (!toast || !element || element === document.body || toast.contains(element)) return
      const rect = element.getBoundingClientRect()
      if (rect.bottom > toast.getBoundingClientRect().top - 8 && rect.top < window.innerHeight) {
        element.scrollIntoView({ block: 'center', behavior: 'smooth' })
      }
    }
    keepClear(document.activeElement)
    function onFocusIn(event: FocusEvent) {
      keepClear(event.target as Element | null)
    }
    document.addEventListener('focusin', onFocusIn)
    return () => document.removeEventListener('focusin', onFocusIn)
  }, [])

  return (
    <div
      ref={container}
      role="status"
      className="flex w-[22rem] max-w-[calc(100vw-2rem)] items-center gap-3 rounded-md border bg-popover px-3 py-2 text-sm text-popover-foreground shadow-md"
      onMouseEnter={pause}
      onMouseLeave={resume}
      onFocus={pause}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) resume()
      }}
    >
      <span className="min-w-0 flex-1">{message}</span>
      <button
        type="button"
        className="rounded px-1 font-medium underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring/50"
        onClick={onUndo}
      >
        Undo
      </button>
    </div>
  )
}
