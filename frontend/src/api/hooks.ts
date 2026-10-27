import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import type { Board, BoardColumn, Note, Tag, Task } from './types'

export function useBoards() {
  return useQuery({ queryKey: ['boards'], queryFn: () => api.get<Board[]>('/boards') })
}

export function useColumns(boardId: number | undefined) {
  return useQuery({
    queryKey: ['columns', boardId],
    queryFn: () => api.get<BoardColumn[]>(`/boards/${boardId}/columns`),
    enabled: boardId !== undefined,
  })
}

export function useTasks(boardId: number | undefined) {
  return useQuery({
    queryKey: ['tasks', boardId],
    queryFn: () => api.get<Task[]>(`/tasks?board_id=${boardId}`),
    enabled: boardId !== undefined,
  })
}

export function useTags() {
  return useQuery({ queryKey: ['tags'], queryFn: () => api.get<Tag[]>('/tags') })
}

export function useCreateTask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { board_id: number; column_id: number; title: string }) =>
      api.post<Task>('/tasks', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', boardId] }),
  })
}

export function useUpdateTask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: number; title: string; description_md: string; priority: string }) =>
      api.patch<Task>(`/tasks/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', boardId] }),
  })
}

export function useMoveTask(boardId: number | undefined) {
  const qc = useQueryClient()
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
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', boardId] }),
  })
}

export function useDeleteTask(boardId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/tasks/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks', boardId] }),
  })
}

export function useNotes() {
  return useQuery({ queryKey: ['notes'], queryFn: () => api.get<Note[]>('/notes') })
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

export function useCreateNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { title: string; body_md: string }) => api.post<Note>('/notes', body),
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

export function useDeleteNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/notes/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })
}
