import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const COLUMN_MIN_WIDTH = 200
export const COLUMN_MAX_WIDTH = 720

export function clampColumnWidth(width: number) {
  return Math.round(Math.min(COLUMN_MAX_WIDTH, Math.max(COLUMN_MIN_WIDTH, width)))
}

interface ColumnWidthState {
  widths: Record<string, number>
  setWidth: (columnId: number, width: number) => void
  resetWidth: (columnId: number) => void
}

export const useColumnWidths = create<ColumnWidthState>()(
  persist(
    (set) => ({
      widths: {},
      setWidth: (columnId, width) =>
        set((state) => ({
          widths: { ...state.widths, [String(columnId)]: clampColumnWidth(width) },
        })),
      resetWidth: (columnId) =>
        set((state) => {
          const { [String(columnId)]: _removed, ...rest } = state.widths
          return { widths: rest }
        }),
    }),
    { name: 'fai-column-widths' },
  ),
)
