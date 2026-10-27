import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import type { Board, BoardColumn, Tag, Task } from './types'

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
