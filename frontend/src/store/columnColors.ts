import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { BoardColumn } from '../api/types'

interface ColumnColorState {
  colors: Record<string, string>
  setColor: (columnId: number, color: string) => void
}

export const useColumnColors = create<ColumnColorState>()(
  persist(
    (set) => ({
      colors: {},
      setColor: (columnId, color) =>
        set((state) => ({ colors: { ...state.colors, [String(columnId)]: color } })),
    }),
    { name: 'fai-column-colors' },
  ),
)

export function columnColor(column: BoardColumn | undefined, overrides: Record<string, string>) {
  if (!column) return undefined
  const saved = overrides[String(column.id)]
  if (saved) return saved
  if (column.is_done) return '#16a34a'
  if (/progress|doing|active|review/i.test(column.name)) return '#d97706'
  return '#2563eb'
}
