import { create } from 'zustand'

interface UIState {
  selectedBoardId: number | undefined
  setSelectedBoardId: (id: number) => void
  commandOpen: boolean
  setCommandOpen: (open: boolean) => void
}

export const useUIStore = create<UIState>((set) => ({
  selectedBoardId: undefined,
  setSelectedBoardId: (id) => set({ selectedBoardId: id }),
  commandOpen: false,
  setCommandOpen: (open) => set({ commandOpen: open }),
}))
