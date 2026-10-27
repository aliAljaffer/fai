import { format, parse, startOfWeek, getDay } from 'date-fns'
import { enUS } from 'date-fns/locale/en-US'
import { useMemo } from 'react'
import { Calendar as BigCalendar, dateFnsLocalizer, type View } from 'react-big-calendar'
import withDragAndDrop, { type EventInteractionArgs } from 'react-big-calendar/lib/addons/dragAndDrop'
import 'react-big-calendar/lib/css/react-big-calendar.css'
import 'react-big-calendar/lib/addons/dragAndDrop/styles.css'
import { useBoards, useTasks, useUpdateTask } from '../api/hooks'
import { useUIStore } from '../store/ui'
import type { Task } from '../api/types'

const localizer = dateFnsLocalizer({
  format,
  parse,
  startOfWeek: () => startOfWeek(new Date(), { weekStartsOn: 1 }),
  getDay,
  locales: { 'en-US': enUS },
})

const DnDCalendar = withDragAndDrop<CalendarEvent>(BigCalendar)

interface CalendarEvent {
  id: number
  title: string
  start: Date
  end: Date
  allDay: true
  task: Task
}

export function Calendar() {
  const { data: boards } = useBoards()
  const { selectedBoardId } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id
  const { data: tasks } = useTasks(boardId)
  const updateTask = useUpdateTask(boardId)

  const events = useMemo<CalendarEvent[]>(
    () =>
      (tasks ?? [])
        .filter((t) => t.due_date)
        .map((t) => {
          const date = new Date(`${t.due_date}T00:00:00`)
          return { id: t.id, title: t.title, start: date, end: date, allDay: true, task: t }
        }),
    [tasks],
  )

  function handleEventDrop({ event, start }: EventInteractionArgs<CalendarEvent>) {
    const due = format(start as Date, 'yyyy-MM-dd')
    updateTask.mutate({
      id: event.task.id,
      title: event.task.title,
      description_md: event.task.description_md,
      priority: event.task.priority,
      due_date: due,
    })
  }

  return (
    <div className="h-[75vh] rounded-lg border p-3">
      <DnDCalendar
        localizer={localizer}
        events={events}
        views={['month'] as View[]}
        defaultView="month"
        onEventDrop={handleEventDrop}
        draggableAccessor={() => true}
        resizable={false}
        popup
      />
    </div>
  )
}
