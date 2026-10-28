import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface CalendarPrefsState {
  firstDay: number
  setFirstDay: (firstDay: number) => void
}

export const useCalendarPrefs = create<CalendarPrefsState>()(
  persist(
    (set) => ({
      firstDay: 0,
      setFirstDay: (firstDay) => set({ firstDay }),
    }),
    { name: 'fai-calendar-prefs' },
  ),
)
