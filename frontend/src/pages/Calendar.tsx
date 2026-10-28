import dayGridPlugin from '@fullcalendar/daygrid'
import interactionPlugin, { type DateClickArg } from '@fullcalendar/interaction'
import FullCalendar from '@fullcalendar/react'
import type { DayCellContentArg, EventClickArg, EventDropArg, EventInput } from '@fullcalendar/core'
import { format } from 'date-fns'
import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { LogMeetingOutcomeDialog } from '../components/calendar/LogMeetingOutcomeDialog'
import { useBoards, useTasks, useUpdateTask } from '../api/hooks'
import { useUIStore } from '../store/ui'
import type { MeetingOutcome, Task } from '../api/types'

interface CalendarTaskEvent extends EventInput {
  id: string
  title: string
  start: string
  allDay: boolean
  extendedProps: { task: Task }
}

export function Calendar() {
  const { data: boards } = useBoards()
  const { selectedBoardId } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id
  const { data: tasks } = useTasks(boardId)
  const updateTask = useUpdateTask(boardId)
  const location = useLocation()
  const navigate = useNavigate()

  const [dialogDate, setDialogDate] = useState<string | null>(null)
  const [dialogOutcome, setDialogOutcome] = useState<MeetingOutcome | undefined>(undefined)

  useEffect(() => {
    const outcome = (location.state as { outcome?: MeetingOutcome } | null)?.outcome
    if (!outcome) return
    setDialogOutcome(outcome)
    setDialogDate(outcome.occurred_on)
    navigate(location.pathname, { replace: true, state: null })
  }, [location, navigate])

  const events = useMemo<CalendarTaskEvent[]>(
    () =>
      (tasks ?? [])
        .filter((t) => t.due_date)
        .map((t) => ({
          id: String(t.id),
          title: t.title,
          start: t.due_date!,
          allDay: true,
          extendedProps: { task: t },
        })),
    [tasks],
  )

  function handleEventDrop(info: EventDropArg) {
    const task = (info.event.extendedProps as { task: Task }).task
    if (!info.event.start) return
    updateTask.mutate(
      {
        id: task.id,
        title: task.title,
        description_md: task.description_md,
        priority: task.priority,
        due_date: format(info.event.start, 'yyyy-MM-dd'),
      },
      { onError: () => info.revert() },
    )
  }

  function handleEventClick(info: EventClickArg) {
    info.jsEvent.preventDefault()
    const task = (info.event.extendedProps as { task: Task }).task
    const params = new URLSearchParams(location.search)
    params.set('task', String(task.id))
    navigate({ pathname: location.pathname, search: params.toString() })
  }

  function handleDateClick(info: DateClickArg) {
    openNewOutcome(info.date)
  }

  function openNewOutcome(date: Date) {
    setDialogOutcome(undefined)
    setDialogDate(format(date, 'yyyy-MM-dd'))
  }

  function renderDayCell(arg: DayCellContentArg) {
    return (
      <div className="fc-day-top-row">
        <span>{arg.dayNumberText}</span>
        <button
          className="fc-day-add"
          aria-label={`Log meeting outcome on ${format(arg.date, 'd MMM yyyy')}`}
          onClick={(event) => {
            event.stopPropagation()
            openNewOutcome(arg.date)
          }}
        >
          +
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="fai-calendar rounded-lg border p-3">
        <FullCalendar
          plugins={[dayGridPlugin, interactionPlugin]}
          initialView="dayGridMonth"
          headerToolbar={{ left: 'prev,next today', center: 'title', right: '' }}
          firstDay={1}
          height="75vh"
          fixedWeekCount={false}
          dayMaxEventRows={3}
          events={events}
          editable
          eventDrop={handleEventDrop}
          eventClick={handleEventClick}
          dateClick={handleDateClick}
          dayCellContent={renderDayCell}
          eventDisplay="block"
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
