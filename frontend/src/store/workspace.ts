import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface WorkspaceState {
  activeWorkspaceId: number | undefined
  setActiveWorkspaceId: (id: number | undefined) => void
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      activeWorkspaceId: undefined,
      setActiveWorkspaceId: (id) => set({ activeWorkspaceId: id }),
    }),
    { name: 'fai-workspace' },
  ),
)
