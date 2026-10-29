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
import { MessageSquarePlus, X } from 'lucide-react'
import { format, addDays, addMonths, parseISO, startOfMonth, subMonths } from 'date-fns'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LogMeetingOutcomeDialog } from '../components/calendar/LogMeetingOutcomeDialog'
import { useBoards, useCalendarSources, useColumns, useImportedCalendars, useMeetingOutcomeByEvent, useTasks, useUpdateTask } from '../api/hooks'
import { TaskComposer } from '../components/board/TaskComposer'
import { useCalendarPrefs } from '../store/calendarPrefs'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import type { ImportedEvent, MeetingOutcome, Task } from '../api/types'

const WEEK_DAYS = {
  0: 'Sunday',
  1: 'Monday',
  2: 'Tuesday',
  3: 'Wednesday',
  4: 'Thursday',
  5: 'Friday',
  6: 'Saturday',
}

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
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { selectedBoardId } = useUIStore()
  const boardId =
    boards?.find((board) => board.id === selectedBoardId)?.id ?? boards?.[0]?.id
  const { data: tasks } = useTasks(boardId)
  const { data: sources } = useCalendarSources(workspaceId)
  const { data: columns } = useColumns(boardId)
  const updateTask = useUpdateTask(boardId)
  const location = useLocation()
  const navigate = useNavigate()

  const [dialogDate, setDialogDate] = useState<string | null>(null)
  const [dayDate, setDayDate] = useState<string | null>(null)
  const [dialogOutcome, setDialogOutcome] = useState<MeetingOutcome | undefined>(undefined)
  const [dialogEvent, setDialogEvent] = useState<ImportedEvent | undefined>(undefined)
  const [importedEvent, setImportedEvent] = useState<{
    event: ImportedEvent
    sourceName: string
    date: string
  } | null>(null)
  const [range, setRange] = useState(initialRange)
  const [draggingTitle, setDraggingTitle] = useState<string | null>(null)
  const calendarRef = useRef<HTMLDivElement>(null)
  const agendaRef = useRef<HTMLElement>(null)
  const { firstDay, setFirstDay } = useCalendarPrefs()
  const [isMobile, setIsMobile] = useState(() =>
    typeof window !== 'undefined' && window.matchMedia('(max-width: 767px)').matches,
  )

  useEffect(() => {
    const query = window.matchMedia('(max-width: 767px)')
    function onChange() {
      setIsMobile(query.matches)
    }
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  }, [])

  useEffect(() => {
    if (!dayDate) return
    agendaRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [dayDate, importedEvent])

  const { data: importedCalendars } = useImportedCalendars(range.from, range.to, workspaceId)
  const { data: eventOutcome } = useMeetingOutcomeByEvent(
    importedEvent?.event.source_id,
    importedEvent?.event.id,
  )
  const eventArc = eventOutcome
    ? (tasks ?? []).find((task) => task.kind === 'arc' && task.meeting_outcome_id === eventOutcome.id)
    : undefined

  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const eventId = params.get('event')
    if (!eventId) return
    const source = params.get('source')
    const sourceId = source ? Number(source) : undefined
    const match = (importedCalendars ?? [])
      .flatMap((calendar) => calendar.events.map((event) => ({ event, sourceName: calendar.source.name })))
      .find(({ event }) => event.id === eventId && (sourceId === undefined || event.source_id === sourceId))
    if (!match) return
    const date = match.event.all_day
      ? match.event.start.slice(0, 10)
      : format(new Date(match.event.start), 'yyyy-MM-dd')
    setDayDate(date)
    setImportedEvent({ ...match, date })
    params.delete('event')
    params.delete('source')
    navigate({ pathname: location.pathname, search: params.toString() }, { replace: true })
  }, [importedCalendars, location, navigate])

  useEffect(() => {
    const outcome = (location.state as { outcome?: MeetingOutcome } | null)?.outcome
    if (!outcome) return
    setDialogOutcome(outcome)
    setDialogEvent(undefined)
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
  const [today] = useState(() => format(new Date(), 'yyyy-MM-dd'))

  function handleEventDrop(info: EventDropArg) {
    const props = info.event.extendedProps as CalendarEventProps
    if (props.kind !== 'task' || !info.event.start) {
      info.revert()
      return
    }
    updateTask.mutate(
      { id: props.task.id, due_date: format(info.event.start, 'yyyy-MM-dd') },
      { onError: () => info.revert() },
    )
  }

  function handleEventClick(info: EventClickArg) {
    info.jsEvent.preventDefault()
    const props = info.event.extendedProps as CalendarEventProps
    if (props.kind === 'imported') {
      const date = props.event.all_day
        ? props.event.start.slice(0, 10)
        : format(new Date(props.event.start), 'yyyy-MM-dd')
      if (date <= today) {
        setDayDate(date)
        setImportedEvent({ event: props.event, sourceName: props.sourceName, date })
        return
      }
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
    // The "+" button's React handler runs after FullCalendar's native cell click,
    // so stopPropagation in that handler cannot prevent this; ignore it here.
    if ((info.jsEvent.target as HTMLElement).closest('.fc-day-add')) return
    setDayDate(format(info.date, 'yyyy-MM-dd'))
  }

  function openNewOutcome(date: Date, event?: ImportedEvent) {
    if (format(date, 'yyyy-MM-dd') > today) return
    setDialogOutcome(undefined)
    setDialogEvent(event)
    setDialogDate(format(date, 'yyyy-MM-dd'))
  }

  function renderDayCell(arg: DayCellContentArg) {
    return (
      <div className="fc-day-top-row">
        <span>{arg.dayNumberText}</span>
        {format(arg.date, 'yyyy-MM-dd') <= today && (
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
        )}
      </div>
    )
  }

  const openColumn = [...(columns ?? [])]
    .sort((a, b) => a.position - b.position)
    .find((column) => !column.is_done)
  const dayTasks = (tasks ?? []).filter((task) => task.due_date === dayDate)
  const dayImported = (importedCalendars ?? []).flatMap((calendar) =>
    calendar.events
      .filter((event) => (event.all_day ? event.start.slice(0, 10) : format(new Date(event.start), 'yyyy-MM-dd')) === dayDate)
      .map((event) => ({ event, sourceName: calendar.source.name })),
  )

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h1 className="text-lg font-semibold">Calendar</h1>
          <p className="text-xs text-muted-foreground">
            Tasks with due dates plus imported feeds. Drag a task onto another day to reschedule it.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Select
            value={String(firstDay)}
            onValueChange={(value) => value && setFirstDay(Number(value))}
          >
            <SelectTrigger size="sm" aria-label="First day of week">
              <SelectValue>
                {(value) => `Week starts ${WEEK_DAYS[Number(value) as keyof typeof WEEK_DAYS]}`}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {Object.entries(WEEK_DAYS).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  Week starts {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button size="sm" disabled={!boardId} onClick={() => openNewOutcome(new Date())}>
            <MessageSquarePlus />
            Add meeting summary
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
            <span
              key={calendar.source.id}
              className="text-attention"
              title={calendar.error}
            >
              {calendar.source.name}: sync failed
              {calendar.last_synced_at
                ? `, showing events from ${format(new Date(calendar.last_synced_at), 'd MMM, HH:mm')}`
                : ''}
              .
            </span>
          ))}
          {draggingTitle && (
            <span role="status" className="text-foreground">
              Drop "{draggingTitle}" on a highlighted day.
            </span>
          )}
        </div>
      )}

      {isMobile ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 7 }, (_, index) => format(addDays(new Date(), index), 'yyyy-MM-dd')).map(
            (date) => {
              const dateTasks = (tasks ?? []).filter((task) => task.due_date === date)
              const dateEvents = (importedCalendars ?? []).flatMap((calendar) =>
                calendar.events
                  .filter((event) =>
                    (event.all_day ? event.start.slice(0, 10) : format(new Date(event.start), 'yyyy-MM-dd')) === date,
                  )
                  .map((event) => ({ event, sourceName: calendar.source.name })),
              )
              return (
                <section key={date} className="rounded-lg border p-3">
                  <div className="mb-1 flex items-center justify-between">
                    <h2 className="text-sm font-semibold">{format(parseISO(date), 'EEE, d MMM')}</h2>
                    <Button size="xs" variant="ghost" onClick={() => setDayDate(date)}>
                      Add
                    </Button>
                  </div>
                  {!dateTasks.length && !dateEvents.length && (
                    <p className="text-xs text-muted-foreground">Nothing scheduled.</p>
                  )}
                  <ul className="flex flex-col">
                    {dateTasks.map((task) => (
                      <li key={`task-${task.id}`}>
                        <button
                          type="button"
                          className="w-full rounded-sm px-1 py-1 text-left text-sm hover:bg-muted"
                          onClick={() => {
                            const params = new URLSearchParams(location.search)
                            params.set('task', String(task.id))
                            navigate({ pathname: location.pathname, search: params.toString() })
                          }}
                        >
                          {task.title}
                        </button>
                      </li>
                    ))}
                    {dateEvents.map(({ event, sourceName }) => (
                      <li key={`event-${event.id}`}>
                        <button
                          type="button"
                          className="w-full rounded-sm px-1 py-1 text-left text-sm text-muted-foreground hover:bg-muted"
                          onClick={() => {
                            if (date <= today) {
                              setDayDate(date)
                              setImportedEvent({ event, sourceName, date })
                            } else if (event.url) {
                              window.open(event.url, '_blank', 'noopener,noreferrer')
                            }
                          }}
                        >
                          {event.title}
                        </button>
                      </li>
                    ))}
                  </ul>
                </section>
              )
            },
          )}
        </div>
      ) : (
      <div
        ref={calendarRef}
        className="fai-calendar flex flex-col rounded-lg border p-3"
      >
        <div className="min-w-0">
          <FullCalendar
            plugins={[dayGridPlugin, interactionPlugin]}
            initialView="dayGridMonth"
            headerToolbar={{ left: 'prev,next today', center: 'title', right: '' }}
            firstDay={firstDay}
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
      )}

      {dayDate && boardId && (
        <section ref={agendaRef} className="flex flex-col gap-3 rounded-lg border p-3" aria-label={`Agenda for ${format(parseISO(dayDate), 'd MMM yyyy')}`}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="text-sm font-semibold">{format(parseISO(dayDate), 'EEEE, d MMM yyyy')}</h2>
            <Button size="icon-sm" variant="ghost" aria-label="Close agenda" onClick={() => { setDayDate(null); setImportedEvent(null) }}>
              <X />
            </Button>
          </div>
          <TaskComposer
            boardId={boardId}
            columnId={openColumn?.id}
            dueDate={dayDate}
            placeholder={`Add a task on ${format(parseISO(dayDate), 'd MMM')}`}
          />
          {dayTasks.length > 0 && (
            <ul className="flex flex-col gap-1">
              {dayTasks.map((task) => (
                <li key={task.id}>
                  <button
                    type="button"
                    className="w-full rounded-sm px-1 py-1 text-left text-sm hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring/50"
                    onClick={() => {
                      const params = new URLSearchParams(location.search)
                      params.set('task', String(task.id))
                      navigate({ pathname: location.pathname, search: params.toString() })
                    }}
                  >
                    {task.title}
                  </button>
                </li>
              ))}
            </ul>
          )}
          {dayImported.length > 0 && (
            <div className="flex flex-col gap-1 border-t pt-2">
              <span className="text-xs font-medium text-muted-foreground">Imported</span>
              {dayImported.map(({ event, sourceName }) => (
                <button
                  key={event.id}
                  type="button"
                  className="rounded-sm px-1 py-1 text-left text-sm hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring/50"
                  onClick={() => setImportedEvent({ event, sourceName, date: dayDate })}
                >
                  {event.title}
                </button>
              ))}
            </div>
          )}
          {importedEvent && (
            <div className="flex flex-col gap-2 rounded-md border bg-muted/30 p-2">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-sm font-medium">{importedEvent.event.title}</p>
                  <p className="text-xs text-muted-foreground">
                    {importedEvent.sourceName} · {format(parseISO(importedEvent.date), 'd MMM yyyy')}
                  </p>
                  {importedEvent.event.url && (
                    <a
                      className="text-xs text-primary underline"
                      href={importedEvent.event.url}
                      target="_blank"
                      rel="noreferrer"
                    >
                      Open event
                    </a>
                  )}
                </div>
                <Button size="icon-sm" variant="ghost" aria-label="Close event" onClick={() => setImportedEvent(null)}>
                  <X />
                </Button>
              </div>
              {eventArc && (
                <Button
                  size="sm"
                  variant="secondary"
                  className="self-start"
                  onClick={() => {
                    const params = new URLSearchParams(location.search)
                    params.set('task', String(eventArc.id))
                    navigate({ pathname: location.pathname, search: params.toString() })
                  }}
                >
                  Open Arc: {eventArc.title}
                </Button>
              )}
              {importedEvent.date <= today && (
                <Button size="sm" variant="outline" onClick={() => {
                  openNewOutcome(parseISO(importedEvent.date), importedEvent.event)
                }}>
                  <MessageSquarePlus />
                  Add meeting summary
                </Button>
              )}
            </div>
          )}
          {dayDate <= today && (
            <Button size="sm" variant="outline" className="self-start" onClick={() => openNewOutcome(parseISO(dayDate))}>
              <MessageSquarePlus />
              Add meeting summary
            </Button>
          )}
        </section>
      )}

      {dialogDate && boardId && (
        <LogMeetingOutcomeDialog
          open={!!dialogDate}
          onOpenChange={(open) => !open && setDialogDate(null)}
          outcome={dialogOutcome}
          occurredOn={dialogDate}
          boardId={boardId}
          event={dialogEvent}
        />
      )}
    </div>
  )
}
