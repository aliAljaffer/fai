import { format, parse, startOfWeek, getDay } from 'date-fns'
import { enUS } from 'date-fns/locale/en-US'
import { useMemo, useState } from 'react'
import { Calendar as BigCalendar, dateFnsLocalizer, type DateCellWrapperProps, type View } from 'react-big-calendar'
import withDragAndDrop, { type EventInteractionArgs } from 'react-big-calendar/lib/addons/dragAndDrop'
import 'react-big-calendar/lib/css/react-big-calendar.css'
import 'react-big-calendar/lib/addons/dragAndDrop/styles.css'
import { Button } from '@/components/ui/button'
import { LogMeetingOutcomeDialog } from '../components/calendar/LogMeetingOutcomeDialog'
import { useBoards, usePendingMeetingOutcomes, useTasks, useUpdateTask } from '../api/hooks'
import { useUIStore } from '../store/ui'
import type { MeetingOutcome, Task } from '../api/types'

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
  const { data: pendingOutcomes } = usePendingMeetingOutcomes()

  const [dialogDate, setDialogDate] = useState<string | null>(null)
  const [dialogOutcome, setDialogOutcome] = useState<MeetingOutcome | undefined>(undefined)

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

  function openNewOutcome(date: Date) {
    setDialogOutcome(undefined)
    setDialogDate(format(date, 'yyyy-MM-dd'))
  }

  function openPendingOutcome(outcome: MeetingOutcome) {
    setDialogOutcome(outcome)
    setDialogDate(outcome.occurred_on)
  }

  function DateCellWrapper({ value, children }: DateCellWrapperProps) {
    return (
      <div className="group relative h-full w-full">
        {children}
        <button
          className="absolute right-1 bottom-1 hidden h-5 w-5 items-center justify-center rounded bg-primary text-xs text-primary-foreground group-hover:flex"
          onClick={(e) => {
            e.stopPropagation()
            openNewOutcome(value)
          }}
        >
          +
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {pendingOutcomes && pendingOutcomes.length > 0 && (
        <div className="flex flex-col gap-2">
          {pendingOutcomes.map((o) => (
            <div
              key={o.id}
              className="flex items-center justify-between rounded-md border bg-accent px-3 py-2 text-sm"
            >
              <span>Meeting "{o.title || 'Untitled'}" just ended — log the outcome?</span>
              <Button size="sm" onClick={() => openPendingOutcome(o)}>
                Log outcome
              </Button>
            </div>
          ))}
        </div>
      )}

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
          components={{ dateCellWrapper: DateCellWrapper }}
        />
      </div>

      {dialogDate && boardId && (
        <LogMeetingOutcomeDialog
          open={!!dialogDate}
          onOpenChange={(open) => !open && setDialogDate(null)}
          outcome={dialogOutcome}
          occurredOn={dialogDate}
          boardId={boardId}
        />
      )}
    </div>
  )
}
