import { Fragment, useLayoutEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '@/lib/utils'
import type { RefPickerState } from './useRefPicker'

const GAP = 4
const MARGIN = 8

function useAnchoredPanel(anchor: HTMLElement | null | undefined, redrawKey: unknown) {
  const panelRef = useRef<HTMLDivElement>(null)

  useLayoutEffect(() => {
    const panel = anchor ? panelRef.current : null
    if (!anchor || !panel) return
    const place = () => {
      const rect = anchor.getBoundingClientRect()
      const height = panel.offsetHeight
      const width = Math.max(220, Math.min(rect.width, 420))
      const below = window.innerHeight - rect.bottom - GAP - MARGIN
      const above = rect.top - GAP - MARGIN
      const flip = below < height && above > below
      panel.style.width = `${width}px`
      panel.style.left = `${Math.max(MARGIN, Math.min(rect.left, window.innerWidth - width - MARGIN))}px`
      panel.style.top = `${flip ? rect.top - GAP - height : rect.bottom + GAP}px`
    }
    place()
    window.addEventListener('scroll', place, true)
    window.addEventListener('resize', place)
    return () => {
      window.removeEventListener('scroll', place, true)
      window.removeEventListener('resize', place)
    }
  }, [anchor, redrawKey])

  return panelRef
}

export function RefPicker({ picker }: { picker: RefPickerState }) {
  const { token, field } = picker
  const panelRef = useAnchoredPanel(field, picker.options)
  if (!token || !field) return null

  const empty =
    token.kind === '@' ? 'Type a title or a reference like FAI-12' : 'No labels match'

  return createPortal(
    <div
      ref={panelRef}
      role="listbox"
      aria-label={token.kind === '@' ? 'Task references' : 'Labels'}
      className="fixed z-[60] flex max-h-56 flex-col overflow-y-auto rounded-md border bg-popover p-1 text-sm shadow-md"
    >
      {!picker.options.length && (
        <span className="px-2 py-1.5 text-xs text-muted-foreground">{empty}</span>
      )}
      {picker.options.map((option, index) => (
        <Fragment key={option.key}>
          {option.group && option.group !== picker.options[index - 1]?.group && (
            <span className="px-2 pt-1.5 pb-0.5 text-[0.65rem] font-medium text-muted-foreground">
              {option.group}
            </span>
          )}
        <button
          type="button"
          role="option"
          aria-selected={index === picker.active}
          className={cn(
            'flex items-baseline gap-2 rounded-sm px-2 py-1.5 text-left',
            index === picker.active && 'bg-accent text-accent-foreground',
          )}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => picker.select(option)}
        >
          <span className="shrink-0 font-mono text-xs">{option.label}</span>
          {option.detail && (
            <span className="truncate text-xs text-muted-foreground">{option.detail}</span>
          )}
        </button>
        </Fragment>
      ))}
    </div>,
    document.body,
  )
}
