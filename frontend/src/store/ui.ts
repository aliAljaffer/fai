import { create } from 'zustand'

interface UIState {
  selectedBoardId: number | undefined
  setSelectedBoardId: (id: number | undefined) => void
  commandOpen: boolean
  setCommandOpen: (open: boolean) => void
  newTaskRequest: number
  requestNewTask: () => void
}

export const useUIStore = create<UIState>((set) => ({
  selectedBoardId: undefined,
  setSelectedBoardId: (id) => set({ selectedBoardId: id }),
  commandOpen: false,
  setCommandOpen: (open) => set({ commandOpen: open }),
  newTaskRequest: 0,
  requestNewTask: () => set((state) => ({ newTaskRequest: state.newTaskRequest + 1 })),
}))
