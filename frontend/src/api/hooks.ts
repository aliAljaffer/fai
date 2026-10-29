import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import type {
  AISettings,
  AppearanceSettings,
  Attachment,
  BackupRestoreResult,
  Board,
  BoardColumn,
  CalendarSource,
  ExtractedTask,
  ImportedCalendar,
  JiraExportResponse,
  JiraLink,
  JiraSuggestion,
  LabelSuggestions,
  MeetingOutcome,
  Note,
  Reminder,
  Tag,
  Task,
  TaskBacklink,
  TaskComment,
  TaskRef,
  TimeEntry,
  Workspace,
  WorkspaceResetResult,
  WorkspaceTasksExport,
  WorkspaceTasksImportResult,
} from './types'
import { useWorkspaceStore } from '../store/workspace'

const taskWriteQueues = new Map<number, Promise<unknown>>()

function queueTaskWrite<T>(id: number, run: () => Promise<T>): Promise<T> {
  const previous = taskWriteQueues.get(id) ?? Promise.resolve()
  const next = previous.catch(() => undefined).then(run)
  taskWriteQueues.set(id, next)
  const cleanup = () => {
    if (taskWriteQueues.get(id) === next) taskWriteQueues.delete(id)
  }
  next.then(cleanup, cleanup)
  return next
}

function workspaceParam(workspaceId: number | undefined) {
  return workspaceId === undefined ? '' : `?workspace_id=${workspaceId}`
}

export function useWorkspaces() {
  return useQuery({ queryKey: ['workspaces'], queryFn: () => api.get<Workspace[]>('/workspaces') })
}

export function useCreateWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; tag: string }) => api.post<Workspace>('/workspaces', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['workspaces'] }),
  })
}

export function useUpdateWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number; name?: string; tag?: string }) =>
      api.patch<Workspace>(`/workspaces/${id}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workspaces'] })
      qc.invalidateQueries({ queryKey: ['boards'] })
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['notes'] })
    },
  })
}

export function useDeleteBoard() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/boards/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['boards'] })
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['columns'] })
    },
  })
}

export function useExportBackup() {
  return useMutation({
    mutationFn: () => {
      const stamp = new Date().toISOString().replace(/[-:]/g, '').replace(/\..+$/, '')
      return api.streamDownload('/backup', `fai-backup-${stamp}.zip`)
    },
  })
}

export function useRestoreBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ file, confirm }: { file: File; confirm: string }) => {
      const form = new FormData()
      form.append('file', file)
      form.append('confirm', confirm)
      return api.postForm<BackupRestoreResult>('/backup/restore', form)
    },
    onSuccess: () => {
      qc.invalidateQueries()
    },
  })
}

export function useExportWorkspaceTasks() {
  return useMutation({
    mutationFn: (workspaceId: number) =>
      api.get<WorkspaceTasksExport>(`/workspaces/${workspaceId}/tasks/export`),
  })
}

export function useImportWorkspaceTasks() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ workspaceId, data }: { workspaceId: number; data: WorkspaceTasksExport }) =>
      api.post<WorkspaceTasksImportResult>(`/workspaces/${workspaceId}/tasks/import`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['task'] })
      qc.invalidateQueries({ queryKey: ['tags'] })
      qc.invalidateQueries({ queryKey: ['boards'] })
      qc.invalidateQueries({ queryKey: ['columns'] })
    },
  })
}

export function useResetWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (workspaceId: number) =>
      api.post<WorkspaceResetResult>(`/workspaces/${workspaceId}/reset`, {
        confirm: 'I understand',
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['task'] })
      qc.invalidateQueries({ queryKey: ['tags'] })
      qc.invalidateQueries({ queryKey: ['boards'] })
      qc.invalidateQueries({ queryKey: ['columns'] })
    },
  })
}

export function useBoards(workspaceId?: number) {
  return useQuery({
    queryKey: ['boards', workspaceId ?? null],
    queryFn: () => api.get<Board[]>(`/boards${workspaceParam(workspaceId)}`),
  })
}

export function useColumns(boardId: number | undefined) {
  return useQuery({
    queryKey: ['columns', boardId],
    queryFn: () => api.get<BoardColumn[]>(`/boards/${boardId}/columns`),
    enabled: boardId !== undefined,
  })
}

export function useCreateBoard(workspaceId?: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.post<Board>('/boards', { name, workspace_id: workspaceId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['boards'] }),
  })
}

export function useUpdateBoard() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: number; name: string }) =>
      api.patch<Board>(`/boards/${id}`, { name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['boards'] }),
  })
}

export function useCreateColumn(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; position: number; is_done: boolean }) =>
      api.post<BoardColumn>(`/boards/${boardId}/columns`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['columns', boardId] }),
  })
}

export function useUpdateColumn(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number; name: string; is_done: boolean }) =>
      api.patch<BoardColumn>(`/boards/${boardId}/columns/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['columns', boardId] }),
  })
}

export function useReorderColumns(boardId: number | undefined) {
  const qc = useQueryClient()
  const key = ['columns', boardId] as const
  return useMutation({
    mutationFn: (ids: number[]) => api.patch<BoardColumn[]>(`/boards/${boardId}/columns`, { ids }),
    onMutate: async (ids) => {
      await qc.cancelQueries({ queryKey: key })
      const previous = qc.getQueryData<BoardColumn[]>(key)
      if (previous) {
        const byId = new Map(previous.map((column) => [column.id, column]))
        qc.setQueryData<BoardColumn[]>(
          key,
          ids.flatMap((id, index) => {
            const column = byId.get(id)
            return column ? [{ ...column, position: index + 1 }] : []
          }),
        )
      }
      return { previous }
    },
    onError: (_error, _ids, context) => {
      if (context?.previous) qc.setQueryData(key, context.previous)
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

export interface TaskFilters {
  q?: string
  columnId?: number
  tag?: string
}

export function useTasks(boardId: number | undefined, filters: TaskFilters = {}) {
  const { q, columnId, tag } = filters
  return useQuery({
    queryKey: ['tasks', boardId, q ?? '', columnId ?? null, tag ?? ''],
    queryFn: () => {
      const params = new URLSearchParams()
      if (boardId !== undefined) params.set('board_id', String(boardId))
      if (q) params.set('q', q)
      if (columnId !== undefined) params.set('column_id', String(columnId))
      if (tag) params.set('tag', tag)
      return api.get<Task[]>(`/tasks?${params}`)
    },
    enabled: boardId !== undefined,
  })
}

export function useAllTasks(enabled = true) {
  return useQuery({
    queryKey: ['tasks', 'all', '', null, ''],
    queryFn: () => api.get<Task[]>('/tasks'),
    enabled,
  })
}

export function useWorkspaceTasks(workspaceId?: number) {
  const { data: boards } = useBoards(workspaceId)
  const results = useQueries({
    queries: (boards ?? []).map((board) => ({
      queryKey: ['tasks', board.id, '', null, ''],
      queryFn: () => api.get<Task[]>(`/tasks?board_id=${board.id}`),
    })),
  })
  return {
    data: results.flatMap((result) => result.data ?? []),
    isLoading: results.some((result) => result.isLoading),
  }
}

export function useAllColumns(boards: Board[] | undefined) {
  return useQueries({
    queries: (boards ?? []).map((board) => ({
      queryKey: ['columns', board.id],
      queryFn: () => api.get<BoardColumn[]>(`/boards/${board.id}/columns`),
    })),
  })
}

export function useTask(id: number | undefined) {
  return useQuery({
    queryKey: ['task', id],
    queryFn: () => api.get<Task>(`/tasks/${id}`),
    enabled: id !== undefined,
  })
}

export function useResolveRefs(q: string, workspaceId?: number) {
  const trimmed = q.trim()
  return useQuery({
    queryKey: ['refs', trimmed.toLowerCase(), workspaceId ?? null],
    queryFn: () => {
      const params = new URLSearchParams({ q: trimmed })
      if (workspaceId !== undefined) params.set('workspace_id', String(workspaceId))
      return api.get<TaskRef[]>(`/refs/resolve?${params}`)
    },
    enabled: trimmed.length > 0,
    staleTime: 30_000,
    retry: false,
  })
}

export function useTaskReferences(taskId: number | undefined) {
  return useQuery({
    queryKey: ['task', taskId, 'references'],
    queryFn: () => api.get<TaskBacklink[]>(`/tasks/${taskId}/references`),
    enabled: taskId !== undefined,
  })
}

export function useTaskComments(taskId: number | undefined) {
  return useQuery({
    queryKey: ['task', taskId, 'comments'],
    queryFn: () => api.get<TaskComment[]>(`/tasks/${taskId}/comments`),
    enabled: taskId !== undefined,
  })
}

export function useCreateTaskComment(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (bodyMd: string) =>
      api.post<TaskComment>(`/tasks/${taskId}/comments`, { body_md: bodyMd }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', taskId, 'comments'] }),
  })
}

export function useDeleteTaskComment(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (commentId: number) => api.delete<void>(`/tasks/${taskId}/comments/${commentId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', taskId, 'comments'] }),
  })
}

export function useTags() {
  return useQuery({ queryKey: ['tags'], queryFn: () => api.get<Tag[]>('/tags') })
}

export function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; color?: string }) => api.post<Tag>('/tags', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}

export function useAddTaskTag(taskId: number, boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tagId: number) => api.put<void>(`/tasks/${taskId}/tags/${tagId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['task', taskId] })
      qc.invalidateQueries({ queryKey: ['tasks', boardId] })
    },
  })
}

export function useRemoveTaskTag(taskId: number, boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tagId: number) => api.delete<void>(`/tasks/${taskId}/tags/${tagId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['task', taskId] })
      qc.invalidateQueries({ queryKey: ['tasks', boardId] })
    },
  })
}

export function useCreateTask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      board_id: number
      column_id: number
      title: string
      due_date?: string
      parent_task_id?: number
      kind?: 'arc' | 'task' | 'step'
    }) => api.post<Task>('/tasks', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', boardId] }),
  })
}

export function useAITask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      text: string
      board_id?: number
      column_id?: number
      workspace_id?: number
    }) =>
      api.post<Task>('/tasks/ai', {
        ...body,
        tz_offset: -new Date().getTimezoneOffset(),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', boardId] }),
  })
}

export type TaskPatch = Partial<Pick<Task, 'title' | 'description_md' | 'priority' | 'due_date'>>

export function useUpdateTask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number } & TaskPatch) =>
      queueTaskWrite(id, () => api.patch<Task>(`/tasks/${id}`, body)),
    onMutate: async ({ id, ...body }) => {
      await qc.cancelQueries({ queryKey: ['tasks'] })
      await qc.cancelQueries({ queryKey: ['task', id] })
      const previousLists = qc.getQueriesData<Task[]>({ queryKey: ['tasks'] })
      const previousTask = qc.getQueryData<Task>(['task', id])
      qc.setQueriesData<Task[]>({ queryKey: ['tasks'] }, (current) =>
        current?.map((task) => (task.id === id ? { ...task, ...body } : task)),
      )
      qc.setQueryData<Task>(['task', id], (current) => (current ? { ...current, ...body } : current))
      return { previousLists, previousTask }
    },
    onError: (_error, variables, context) => {
      for (const [key, value] of context?.previousLists ?? []) qc.setQueryData(key, value)
      if (context?.previousTask) qc.setQueryData(['task', variables.id], context.previousTask)
    },
    onSuccess: (task) => {
      qc.setQueryData<Task>(['task', task.id], task)
      qc.setQueriesData<Task[]>({ queryKey: ['tasks'] }, (current) =>
        current?.map((item) => (item.id === task.id ? { ...task } : item)),
      )
      qc.invalidateQueries({ queryKey: ['tags'] })
      if (boardId !== undefined) qc.invalidateQueries({ queryKey: ['boards', boardId, 'tags'] })
    },
  })
}

function reorderTasks(
  tasks: Task[],
  move: { id: number; column_id: number; before_task_id?: number },
) {
  const moving = tasks.find((t) => t.id === move.id)
  if (!moving) return tasks

  const rest = tasks.filter((t) => t.id !== move.id)
  const movingUpdated = { ...moving, column_id: move.column_id }
  const target = rest
    .filter((t) => t.column_id === move.column_id)
    .sort((a, b) => a.position - b.position)

  let insertAt = target.length
  if (move.before_task_id !== undefined) {
    const index = target.findIndex((t) => t.id === move.before_task_id)
    if (index !== -1) insertAt = index
  }

  const ordered = [...target.slice(0, insertAt), movingUpdated, ...target.slice(insertAt)]
  const positions = new Map(ordered.map((t, i) => [t.id, (i + 1) * 1024]))
  return [...rest, movingUpdated].map((t) => {
    const position = positions.get(t.id)
    return position === undefined ? t : { ...t, position }
  })
}

export function useMoveTask(boardId: number | undefined) {
  const qc = useQueryClient()
  const key = ['tasks', boardId, '', null, ''] as const
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: {
      id: number
      column_id: number
      after_task_id?: number
      before_task_id?: number
    }) => api.post<Task>(`/tasks/${id}/move`, body),
    onMutate: async (move) => {
      await qc.cancelQueries({ queryKey: key })
      const previous = qc.getQueryData<Task[]>(key)
      if (previous) qc.setQueryData<Task[]>(key, reorderTasks(previous, move))
      return { previous }
    },
    onError: (_error, _move, context) => {
      if (context?.previous) qc.setQueryData(key, context.previous)
    },
    onSuccess: (task) => {
      qc.setQueryData<Task[]>(key, (current) =>
        current?.map((t) =>
          t.id === task.id ? { ...task, tags: task.tags?.length ? task.tags : t.tags } : t,
        ),
      )
      qc.invalidateQueries({ queryKey: ['task', task.id] })
      qc.invalidateQueries({ queryKey: ['tasks'] })
    },
  })
}

export function useDeleteTask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/tasks/${id}`),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ['tasks'] })
      const previousLists = qc.getQueriesData<Task[]>({ queryKey: ['tasks'] })
      qc.setQueriesData<Task[]>({ queryKey: ['tasks'] }, (current) =>
        current?.filter((task) => task.id !== id),
      )
      return { previousLists }
    },
    onError: (_error, _id, context) => {
      for (const [key, value] of context?.previousLists ?? []) qc.setQueryData(key, value)
    },
    onSuccess: (_data, id) => {
      qc.removeQueries({ queryKey: ['task', id] })
      qc.invalidateQueries({ queryKey: ['tasks', boardId] })
    },
  })
}

export function useRestoreTasks(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (ids: number[]) =>
      Promise.all(ids.map((id) => api.post<Task>(`/tasks/${id}/restore`, {}))),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['task'] })
      if (boardId !== undefined) qc.invalidateQueries({ queryKey: ['tasks', boardId] })
    },
  })
}

export type BulkTaskAction =
  | { type: 'move'; columnId: number }
  | { type: 'priority'; priority: string }
  | { type: 'dueDate'; dueDate: string | null }
  | { type: 'addTag'; tagId: number }
  | { type: 'removeTag'; tagId: number }
  | { type: 'delete' }

// Bulk operations reuse the single-task endpoints. Moves run sequentially so each
// moved task lands after the previous one instead of racing for the same position.
export function useBulkTaskActions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ tasks, action }: { tasks: Task[]; action: BulkTaskAction }) => {
      const run = (task: Task) => {
        switch (action.type) {
          case 'move':
            return api.post<void>(`/tasks/${task.id}/move`, { column_id: action.columnId })
          case 'priority':
            return api.patch<void>(`/tasks/${task.id}`, { priority: action.priority })
          case 'dueDate':
            return api.patch<void>(`/tasks/${task.id}`, { due_date: action.dueDate })
          case 'addTag':
            return api.put<void>(`/tasks/${task.id}/tags/${action.tagId}`)
          case 'removeTag':
            return api.delete<void>(`/tasks/${task.id}/tags/${action.tagId}`)
          case 'delete':
            return api.delete<void>(`/tasks/${task.id}`)
        }
      }

      if (action.type === 'move') {
        for (const task of tasks) await run(task)
        return
      }
      await Promise.all(tasks.map(run))
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['task'] })
    },
  })
}

export interface SearchResults {
  tasks: Task[]
  notes: Note[]
}

export function useSearch(q: string, workspaceId?: number) {
  return useQuery({
    queryKey: ['search', q, workspaceId ?? null],
    queryFn: () => {
      const params = new URLSearchParams({ q })
      if (workspaceId !== undefined) params.set('workspace_id', String(workspaceId))
      return api.get<SearchResults>(`/search?${params}`)
    },
    enabled: q.length > 0,
  })
}

export function useNotes(workspaceId?: number) {
  return useQuery({
    queryKey: ['notes', 'list', workspaceId ?? null],
    queryFn: () => api.get<Note[]>(`/notes${workspaceParam(workspaceId)}`),
  })
}

export function useNote(id: number | undefined) {
  return useQuery({
    queryKey: ['notes', id],
    queryFn: () => api.get<{ note: Note; backlinks: Note[] }>(`/notes/${id}`),
    enabled: id !== undefined,
  })
}

export function useNoteByTitle(title: string | undefined) {
  return useQuery({
    queryKey: ['notes', 'by-title', title],
    queryFn: () => api.get<Note>(`/notes/by-title/${encodeURIComponent(title!)}`),
    enabled: !!title,
    retry: false,
  })
}

export function useCreateNote(workspaceId?: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { title: string; body_md: string }) =>
      api.post<Note>('/notes', { ...body, workspace_id: workspaceId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })
}

export function useUpdateNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number; title: string; body_md: string }) =>
      api.patch<Note>(`/notes/${id}`, body),
    onSuccess: (note) => {
      qc.invalidateQueries({ queryKey: ['notes'] })
      qc.invalidateQueries({ queryKey: ['notes', note.id] })
    },
  })
}

export function useReminders(workspaceId?: number) {
  return useQuery({
    queryKey: ['reminders', workspaceId ?? null],
    queryFn: () => api.get<Reminder[]>(`/reminders${workspaceParam(workspaceId)}`),
    refetchInterval: 60_000,
  })
}

export function useCreateReminder(workspaceId?: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { title: string; remind_at: string; task_id?: number }) =>
      api.post<Reminder>('/reminders', { ...body, workspace_id: workspaceId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['reminders'] }),
  })
}

export function useUpdateReminder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number; title: string; remind_at: string; task_id?: number }) =>
      api.patch<Reminder>(`/reminders/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['reminders'] }),
  })
}

export function useDeleteReminder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/reminders/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['reminders'] }),
  })
}

export function useAIReminder(workspaceId?: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (text: string) =>
      api.post<Reminder>('/reminders/ai', {
        text,
        tz_offset: -new Date().getTimezoneOffset(),
        workspace_id: workspaceId,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['reminders'] }),
  })
}

export function useRunningTime() {
  return useQuery({
    queryKey: ['time', 'running'],
    queryFn: () => api.get<TimeEntry | null>('/time/running'),
    refetchInterval: 1000,
  })
}

export function useStartTime() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (taskId: number) => api.post<TimeEntry>(`/tasks/${taskId}/time/start`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['time'] }),
  })
}

export function useStopTime() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (taskId: number) => api.post<TimeEntry>(`/tasks/${taskId}/time/stop`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['time'] }),
  })
}

export function useTaskTimeEntries(taskId: number | undefined) {
  return useQuery({
    queryKey: ['time', 'task', taskId],
    queryFn: () => api.get<TimeEntry[]>(`/tasks/${taskId}/time`),
    enabled: taskId !== undefined,
  })
}

export interface TimePoint {
  date: string
  seconds: number
}

export function useTimeStats(from: string, to: string) {
  return useQuery({
    queryKey: ['stats', 'time', from, to],
    queryFn: () => api.get<TimePoint[]>(`/stats/time?from=${from}&to=${to}`),
  })
}

export function useJiraLink(taskId: number) {
  return useQuery({
    queryKey: ['jira', 'link', taskId],
    queryFn: () => api.get<JiraLink | null>(`/tasks/${taskId}/jira/link`),
  })
}

export function useJiraSuggestions(taskId: number, query: string) {
  const trimmed = query.trim()
  return useQuery({
    queryKey: ['jira', 'suggestions', taskId, trimmed],
    queryFn: () =>
      api.get<JiraSuggestion[]>(
        `/tasks/${taskId}/jira/suggestions?q=${encodeURIComponent(trimmed)}`,
      ),
    enabled: trimmed.length >= 2,
    staleTime: 30_000,
    retry: false,
  })
}

export function useLinkJira(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (jiraKey: string) => api.post<JiraLink>(`/tasks/${taskId}/jira/link`, { jira_key: jiraKey }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['jira', 'link', taskId] })
      qc.invalidateQueries({ queryKey: ['jira', 'links'] })
    },
  })
}

export function useUnlinkJira(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.delete<void>(`/tasks/${taskId}/jira/link`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['jira', 'link', taskId] })
      qc.invalidateQueries({ queryKey: ['jira', 'links'] })
    },
  })
}

export function useExportJira(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<JiraExportResponse>('/jira/export', { task_id: taskId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['jira'] }),
  })
}

export function useAIStatus() {
  return useQuery({ queryKey: ['ai', 'status'], queryFn: () => api.get<{ configured: boolean }>('/ai/status') })
}

export function useAISettings() {
  return useQuery({
    queryKey: ['ai', 'settings'],
    queryFn: () => api.get<AISettings>('/ai/settings'),
  })
}

export function useUpdateAISettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      provider: AISettings['provider']
      base_url: string
      model: string
      api_key?: string
      clear_api_key?: boolean
    }) => api.put<AISettings>('/ai/settings', body),
    onSuccess: (settings) => {
      qc.setQueryData(['ai', 'settings'], settings)
      qc.invalidateQueries({ queryKey: ['ai', 'status'] })
    },
  })
}

export function useAppearance() {
  return useQuery({
    queryKey: ['appearance'],
    queryFn: () => api.get<AppearanceSettings>('/appearance'),
  })
}

export function useUpdateAppearance() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Omit<AppearanceSettings, 'updated_at'>) =>
      api.put<AppearanceSettings>('/appearance', body),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: ['appearance'] })
      const previous = qc.getQueryData<AppearanceSettings>(['appearance'])
      qc.setQueryData<AppearanceSettings>(['appearance'], (current) => ({
        global_palette_id: next.global_palette_id,
        workspace_palettes: next.workspace_palettes,
        custom_palettes: next.custom_palettes,
        updated_at: current?.updated_at,
      }))
      return { previous }
    },
    onError: (_error, _next, context) => {
      if (context?.previous) qc.setQueryData(['appearance'], context.previous)
    },
    onSuccess: (settings) => qc.setQueryData(['appearance'], settings),
  })
}

export function useSetTaskType(taskId: number, boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { kind: 'arc' | 'task' | 'step'; parent_task_id?: number | null }) =>
      api.patch<Task>(`/tasks/${taskId}/type`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks', boardId] })
      qc.invalidateQueries({ queryKey: ['task', taskId] })
      qc.invalidateQueries({ queryKey: ['tasks', taskId, 'subtasks'] })
    },
  })
}

export function useSubtasks(taskId: number) {
  return useQuery({ queryKey: ['tasks', taskId, 'subtasks'], queryFn: () => api.get<Task[]>(`/tasks/${taskId}/subtasks`) })
}

export function useCreateSubtask(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      board_id: number
      column_id: number
      title: string
      kind?: 'arc' | 'task' | 'step'
      parent_task_id?: number
    }) => api.post<Task>('/tasks', { ...body, parent_task_id: body.parent_task_id ?? taskId }),
    onSuccess: (_created, body) => {
      qc.invalidateQueries({ queryKey: ['tasks', taskId, 'subtasks'] })
      if (body.parent_task_id && body.parent_task_id !== taskId) {
        qc.invalidateQueries({ queryKey: ['tasks', body.parent_task_id, 'subtasks'] })
      }
    },
  })
}

export function useAIBreakdown(taskId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<{ id: number; title: string }[]>(`/tasks/${taskId}/ai/breakdown`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', taskId, 'subtasks'] }),
  })
}

export function useSuggestLabels(taskId: number) {
  return useMutation({
    mutationFn: () => api.post<LabelSuggestions>(`/tasks/${taskId}/ai/labels`),
  })
}

export interface AIChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface PendingChatAction {
  type: string
  task_id: number
  task_ref: string
  task_title: string
  column_id: number
  column_name: string
}

export function useAIChat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ messages, workspaceId }: { messages: AIChatMessage[]; workspaceId?: number }) =>
      api.post<{
        reply: string
        created?: { type: string; id: number; title: string }
        pending?: PendingChatAction
      }>('/ai/chat', {
        messages,
        workspace_id: workspaceId,
        tz_offset: -new Date().getTimezoneOffset(),
      }),
    onSuccess: (result) => {
      qc.invalidateQueries({ queryKey: ['chat-messages'] })
      if (!result.created) return
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['reminders'] })
    },
  })
}

export function useChatHistory(workspaceId?: number) {
  return useQuery({
    queryKey: ['chat-messages', workspaceId ?? null],
    queryFn: () =>
      api.get<ChatMessage[]>(`/ai/chat/messages?workspace_id=${workspaceId}&limit=50`),
    enabled: workspaceId !== undefined,
  })
}

export interface ChatMessage {
  id: number
  workspace_id: number
  role: 'user' | 'assistant'
  content: string
  created_at: string
}

export function useConfirmChatAction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (pending: PendingChatAction) =>
      api.post<{ reply: string }>('/ai/chat/confirm', pending),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['task'] })
    },
  })
}

export interface NoteTaskPlan {
  title: string
  subtasks: string[]
}

export function useNoteTaskPlan(noteId: number) {
  return useMutation({
    mutationFn: (body: { title: string; body_md: string }) =>
      api.post<NoteTaskPlan>(`/notes/${noteId}/ai/plan`, body),
  })
}

export function useCreateTaskFromNote(noteId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      board_id: number
      title: string
      subtasks: string[]
      note_body?: string
    }) => api.post<{ task: Task; subtasks: Task[] }>(`/notes/${noteId}/tasks`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['notes'] })
      qc.invalidateQueries({ queryKey: ['notes', noteId] })
    },
  })
}

export function useCalendarFeedURL(workspaceId?: number) {
  return useQuery({
    queryKey: ['calendar-feed', workspaceId ?? null],
    queryFn: () =>
      api.get<{ url: string }>(
        workspaceId === undefined ? '/calendar/feed' : `/calendar/feed?workspace_id=${workspaceId}`,
      ),
  })
}

export function useCalendarSources(workspaceId?: number) {
  return useQuery({
    queryKey: ['calendar-sources', workspaceId ?? null],
    queryFn: () => api.get<CalendarSource[]>(`/calendar/sources${workspaceParam(workspaceId)}`),
  })
}

export function useCreateCalendarSource(workspaceId?: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; url: string; color: string }) =>
      api.post<CalendarSource>('/calendar/sources', { ...body, workspace_id: workspaceId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar-sources'] })
      qc.invalidateQueries({ queryKey: ['calendar-imported'] })
    },
  })
}

export function useUpdateCalendarSourceColor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, color }: { id: number; color: string }) =>
      api.patch<CalendarSource>(`/calendar/sources/${id}`, { color }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar-sources'] })
      qc.invalidateQueries({ queryKey: ['calendar-imported'] })
    },
  })
}

export function useDeleteCalendarSource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/calendar/sources/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar-sources'] })
      qc.invalidateQueries({ queryKey: ['calendar-imported'] })
    },
  })
}

export function useImportedCalendars(from: string, to: string, workspaceId?: number) {
  return useQuery({
    queryKey: ['calendar-imported', from, to, workspaceId ?? null],
    queryFn: () =>
      api.get<ImportedCalendar[]>(
        `/calendar/imported?from=${from}&to=${to}${workspaceId === undefined ? '' : `&workspace_id=${workspaceId}`}`,
      ),
    placeholderData: (previous) => previous,
  })
}

export function usePendingMeetingOutcomes(workspaceId?: number) {
  return useQuery({
    queryKey: ['meeting-outcomes', 'pending', workspaceId ?? null],
    queryFn: () =>
      api.get<MeetingOutcome[]>(
        `/meeting-outcomes?pending=true${workspaceId === undefined ? '' : `&workspace_id=${workspaceId}`}`,
      ),
    refetchOnWindowFocus: true,
    refetchInterval: 60_000,
  })
}

export function useCreateMeetingOutcome(workspaceId?: number) {
  const qc = useQueryClient()
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  return useMutation({
    mutationFn: (body: { occurred_on: string; title: string; notes_md: string; ends_at?: string }) =>
      api.post<MeetingOutcome>('/meeting-outcomes', {
        ...body,
        workspace_id: workspaceId ?? activeWorkspaceId,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meeting-outcomes'] }),
  })
}

export function useUpdateMeetingOutcome() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: {
      id: number
      title: string
      notes_md: string
      ends_at?: string
    }) => api.patch<MeetingOutcome>(`/meeting-outcomes/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meeting-outcomes'] }),
  })
}

export function useExtractTasksFromOutcome() {
  return useMutation({
    mutationFn: (outcomeId: number) =>
      api.post<ExtractedTask[]>(`/meeting-outcomes/${outcomeId}/ai/extract-tasks`),
  })
}

export function useCreateTaskFromOutcome(outcomeId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { title: string; mark_done: boolean; board_id: number; due_date?: string | null }) =>
      api.post<Task>(`/meeting-outcomes/${outcomeId}/tasks`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })
}

export interface JiraImportResult {
  imported: number
  updated: number
  skipped: number
  linked?: number
}

export function useJiraImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { board_id: number; jql?: string; force_update?: boolean }) =>
      api.post<JiraImportResult>('/jira/import', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['jira', 'links'] }),
  })
}

export function useJiraLinks(workspaceId?: number) {
  return useQuery({
    queryKey: ['jira', 'links', workspaceId],
    queryFn: () => api.get<JiraLink[]>(`/jira/links?workspace_id=${workspaceId}`),
    enabled: workspaceId != null,
  })
}

export interface CompletionPoint {
  date: string
  count: number
}

export function useCompletions(from: string, to: string) {
  return useQuery({
    queryKey: ['stats', 'completions', from, to],
    queryFn: () => api.get<CompletionPoint[]>(`/stats/completions?from=${from}&to=${to}`),
  })
}

export function useDeleteNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/notes/${id}`),
    onSuccess: (_data, id) => {
      // Refetching the deleted note would 404 and stall the mutation, which keeps
      // the editor from navigating away until retries finish.
      qc.removeQueries({ queryKey: ['notes', id] })
      qc.invalidateQueries({ queryKey: ['notes', 'list'] })
    },
  })
}

export function useTaskAttachments(taskId: number | undefined) {
  return useQuery({
    queryKey: ['attachments', 'task', taskId],
    queryFn: () => api.get<Attachment[]>(`/tasks/${taskId}/attachments`),
    enabled: taskId !== undefined,
  })
}

export function useNoteAttachments(noteId: number | undefined) {
  return useQuery({
    queryKey: ['attachments', 'note', noteId],
    queryFn: () => api.get<Attachment[]>(`/notes/${noteId}/attachments`),
    enabled: noteId !== undefined,
  })
}

export function useUploadTaskAttachment(taskId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData()
      body.append('file', file)
      return api.postForm<Attachment>(`/tasks/${taskId}/attachments`, body)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['attachments', 'task', taskId] }),
  })
}

export function useUploadNoteAttachment(noteId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData()
      body.append('file', file)
      return api.postForm<Attachment>(`/notes/${noteId}/attachments`, body)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['attachments', 'note', noteId] }),
  })
}

export function useDeleteAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/attachments/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['attachments'] }),
  })
}
