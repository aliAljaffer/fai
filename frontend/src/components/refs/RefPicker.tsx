import { Fragment } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '@/lib/utils'
import type { RefPickerState } from './useRefPicker'

export function RefPicker({ picker }: { picker: RefPickerState }) {
  const { token, field } = picker
  if (!token || !field) return null

  const rect = field.getBoundingClientRect()
  const style = {
    position: 'fixed' as const,
    top: rect.bottom + 4,
    left: rect.left,
    width: Math.max(220, Math.min(rect.width, 420)),
  }
  const empty =
    token.kind === '@' ? 'Type a title or a reference like FAI-12' : 'No labels match'

  return createPortal(
    <div
      role="listbox"
      aria-label={token.kind === '@' ? 'Task references' : 'Labels'}
      className="z-[60] flex max-h-56 flex-col overflow-y-auto rounded-md border bg-popover p-1 text-sm shadow-md"
      style={style}
    >
      {!picker.options.length && (
        <span className="px-2 py-1.5 text-xs text-muted-foreground">{empty}</span>
      )}
      {picker.options.map((option, index) => (
        <Fragment key={option.key}>
          {option.group && option.group !== picker.options[index - 1]?.group && (
            <span className="px-2 pt-1.5 pb-0.5 text-[0.65rem] font-medium tracking-wide text-muted-foreground uppercase">
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
