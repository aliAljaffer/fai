import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface RemindersPanelState {
  open: boolean
  setOpen: (open: boolean) => void
}

export const useRemindersPanel = create<RemindersPanelState>()(
  persist(
    (set) => ({
      open: false,
      setOpen: (open) => set({ open }),
    }),
    { name: 'fai-reminders-panel' },
  ),
)
