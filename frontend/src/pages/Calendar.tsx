import dayGridPlugin from '@fullcalendar/daygrid'
import interactionPlugin, { type DateClickArg } from '@fullcalendar/interaction'
import FullCalendar from '@fullcalendar/react'
import type {
  DayCellContentArg,
  EventApi,
  EventClickArg,
  EventDropArg,
  EventInput,
} from '@fullcalendar/core'
import { CalendarPlus, MessageSquarePlus } from 'lucide-react'
import { format, addMonths, startOfMonth, subMonths } from 'date-fns'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { CalendarSourcesDialog } from '../components/calendar/CalendarSourcesDialog'
import { LogMeetingOutcomeDialog } from '../components/calendar/LogMeetingOutcomeDialog'
import { useBoards, useCalendarSources, useImportedCalendars, useTasks, useUpdateTask } from '../api/hooks'
import { useUIStore } from '../store/ui'
import type { ImportedEvent, MeetingOutcome, Task } from '../api/types'

type CalendarEventProps =
  | { kind: 'task'; task: Task }
  | { kind: 'imported'; event: ImportedEvent; sourceName: string }

function contrastText(hex: string): string {
  if (!/^#[0-9a-f]{6}$/i.test(hex)) return '#ffffff'
  const r = Number.parseInt(hex.slice(1, 3), 16)
  const g = Number.parseInt(hex.slice(3, 5), 16)
  const b = Number.parseInt(hex.slice(5, 7), 16)
  return (r * 299 + g * 587 + b * 114) / 1000 > 150 ? '#111111' : '#ffffff'
}

function initialRange() {
  const now = new Date()
  return {
    from: format(startOfMonth(subMonths(now, 1)), 'yyyy-MM-dd'),
    to: format(addMonths(startOfMonth(now), 2), 'yyyy-MM-dd'),
  }
}

export function Calendar() {
  const { data: boards } = useBoards()
  const { selectedBoardId } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id
  const { data: tasks } = useTasks(boardId)
  const { data: sources } = useCalendarSources()
  const updateTask = useUpdateTask(boardId)
  const location = useLocation()
  const navigate = useNavigate()

  const [dialogDate, setDialogDate] = useState<string | null>(null)
  const [dialogOutcome, setDialogOutcome] = useState<MeetingOutcome | undefined>(undefined)
  const [sourcesOpen, setSourcesOpen] = useState(false)
  const [range, setRange] = useState(initialRange)
  const [draggingTitle, setDraggingTitle] = useState<string | null>(null)
  const calendarRef = useRef<HTMLDivElement>(null)

  const { data: importedCalendars } = useImportedCalendars(range.from, range.to)

  useEffect(() => {
    const outcome = (location.state as { outcome?: MeetingOutcome } | null)?.outcome
    if (!outcome) return
    setDialogOutcome(outcome)
    setDialogDate(outcome.occurred_on)
    navigate(location.pathname, { replace: true, state: null })
  }, [location, navigate])

  const taskEvents = useMemo<EventInput[]>(
    () =>
      (tasks ?? [])
        .filter((task) => task.due_date && !task.completed_at && !task.parent_task_id)
        .map((task) => ({
          id: String(task.id),
          title: task.title,
          start: task.due_date!,
          allDay: true,
          extendedProps: { kind: 'task', task },
        })),
    [tasks],
  )

  const importedEvents = useMemo<EventInput[]>(
    () =>
      (importedCalendars ?? []).flatMap((calendar) =>
        calendar.events.map((event) => ({
          id: event.id,
          title: event.title,
          start: event.start,
          end: event.end,
          allDay: event.all_day,
          editable: false,
          backgroundColor: calendar.source.color,
          borderColor: calendar.source.color,
          textColor: contrastText(calendar.source.color),
          extendedProps: {
            kind: 'imported',
            event,
            sourceName: calendar.source.name,
          } satisfies CalendarEventProps,
        })),
      ),
    [importedCalendars],
  )

  const events = useMemo(() => [...taskEvents, ...importedEvents], [taskEvents, importedEvents])
  const importErrors = (importedCalendars ?? []).filter((calendar) => calendar.error)

  function handleEventDrop(info: EventDropArg) {
    const props = info.event.extendedProps as CalendarEventProps
    if (props.kind !== 'task' || !info.event.start) {
      info.revert()
      return
    }
    updateTask.mutate(
      {
        id: props.task.id,
        title: props.task.title,
        description_md: props.task.description_md,
        priority: props.task.priority,
        due_date: format(info.event.start, 'yyyy-MM-dd'),
      },
      { onError: () => info.revert() },
    )
  }

  function handleEventClick(info: EventClickArg) {
    info.jsEvent.preventDefault()
    const props = info.event.extendedProps as CalendarEventProps
    if (props.kind === 'imported') {
      if (props.event.url) window.open(props.event.url, '_blank', 'noopener,noreferrer')
      return
    }
    const params = new URLSearchParams(location.search)
    params.set('task', String(props.task.id))
    navigate({ pathname: location.pathname, search: params.toString() })
  }

  function handleDragStart(info: { event: EventApi }) {
    setDraggingTitle(info.event.title)
    calendarRef.current?.classList.add('fai-dragging')
  }

  function handleDragStop() {
    setDraggingTitle(null)
    calendarRef.current?.classList.remove('fai-dragging')
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
          aria-label={`Add meeting summary on ${format(arg.date, 'd MMM yyyy')}`}
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
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h1 className="text-lg font-semibold">Calendar</h1>
          <p className="text-xs text-muted-foreground">
            Tasks with due dates plus imported feeds. Drag a task onto another day to reschedule it.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" disabled={!boardId} onClick={() => openNewOutcome(new Date())}>
            <MessageSquarePlus />
            Add meeting summary
          </Button>
          <Button size="sm" variant="outline" onClick={() => setSourcesOpen(true)}>
            <CalendarPlus />
            Calendars
          </Button>
        </div>
      </header>

      {(!!sources?.length || importErrors.length > 0 || draggingTitle) && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
          {sources?.map((source) => (
            <span key={source.id} className="inline-flex items-center gap-1.5">
              <span
                aria-hidden
                className="size-2.5 rounded-full"
                style={{ backgroundColor: source.color }}
              />
              {source.name}
            </span>
          ))}
          {importErrors.map((calendar) => (
            <span key={calendar.source.id} className="text-destructive">
              Could not import {calendar.source.name}.
            </span>
          ))}
          {draggingTitle && (
            <span role="status" className="text-foreground">
              Drop "{draggingTitle}" on a highlighted day.
            </span>
          )}
        </div>
      )}

      <div ref={calendarRef} className="fai-calendar overflow-x-auto rounded-lg border p-3">
        <div className="min-w-[56rem]">
          <FullCalendar
            plugins={[dayGridPlugin, interactionPlugin]}
            initialView="dayGridMonth"
            headerToolbar={{ left: 'prev,next today', center: 'title', right: '' }}
            firstDay={1}
            height="auto"
            fixedWeekCount={false}
            dayMaxEventRows={3}
            events={events}
            editable
            eventDragMinDistance={6}
            dragRevertDuration={150}
            eventDrop={handleEventDrop}
            eventDragStart={handleDragStart}
            eventDragStop={handleDragStop}
            eventClick={handleEventClick}
            dateClick={handleDateClick}
            dayCellContent={renderDayCell}
            datesSet={(info) =>
              setRange((current) => {
                const next = {
                  from: format(info.start, 'yyyy-MM-dd'),
                  to: format(info.end, 'yyyy-MM-dd'),
                }
                return current.from === next.from && current.to === next.to ? current : next
              })
            }
            eventDisplay="block"
          />
        </div>
      </div>

      <CalendarSourcesDialog open={sourcesOpen} onOpenChange={setSourcesOpen} />

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
