import { create } from 'zustand'

interface UIState {
  selectedBoardId: number | undefined
  setSelectedBoardId: (id: number) => void
}

export const useUIStore = create<UIState>((set) => ({
  selectedBoardId: undefined,
  setSelectedBoardId: (id) => set({ selectedBoardId: id }),
}))
