import { useRef, type RefObject } from 'react'
import {
  COLUMN_MAX_WIDTH,
  COLUMN_MIN_WIDTH,
  clampColumnWidth,
  useColumnWidths,
} from '@/store/columnWidths'

const KEY_STEP = 16

export function ColumnResizeHandle({
  columnId,
  columnName,
  targetRef,
  onPreview,
}: {
  columnId: number
  columnName: string
  targetRef: RefObject<HTMLElement | null>
  onPreview: (width: number | null) => void
}) {
  const setWidth = useColumnWidths((state) => state.setWidth)
  const resetWidth = useColumnWidths((state) => state.resetWidth)
  const drag = useRef<{ startX: number; startWidth: number; width: number } | null>(null)

  const currentWidth = () => targetRef.current?.getBoundingClientRect().width ?? COLUMN_MIN_WIDTH

  function finish(commit: boolean) {
    const state = drag.current
    drag.current = null
    if (state && commit && state.width !== state.startWidth) setWidth(columnId, state.width)
    onPreview(null)
  }

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={`Resize ${columnName}`}
      aria-valuemin={COLUMN_MIN_WIDTH}
      aria-valuemax={COLUMN_MAX_WIDTH}
      tabIndex={0}
      title="Drag to resize, double-click to reset"
      className="group/resize absolute inset-y-0 -right-4 z-20 flex w-4 cursor-col-resize touch-none justify-center outline-none select-none"
      onPointerDown={(event) => {
        if (event.button !== 0) return
        event.preventDefault()
        event.stopPropagation()
        event.currentTarget.setPointerCapture(event.pointerId)
        const startWidth = currentWidth()
        drag.current = { startX: event.clientX, startWidth, width: startWidth }
      }}
      onPointerMove={(event) => {
        const state = drag.current
        if (!state) return
        state.width = clampColumnWidth(state.startWidth + event.clientX - state.startX)
        onPreview(state.width)
      }}
      onPointerUp={() => finish(true)}
      onPointerCancel={() => finish(false)}
      onDoubleClick={() => resetWidth(columnId)}
      onKeyDown={(event) => {
        if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
        event.preventDefault()
        setWidth(columnId, currentWidth() + (event.key === 'ArrowRight' ? KEY_STEP : -KEY_STEP))
      }}
    >
      <span className="h-full w-0.5 rounded-full bg-transparent transition-colors group-hover/resize:bg-ring/60 group-focus-visible/resize:bg-ring group-active/resize:bg-ring" />
    </div>
  )
}
