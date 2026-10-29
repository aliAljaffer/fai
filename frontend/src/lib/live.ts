import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getAccessToken } from '../auth/oidc'

const retryDelay = 3000
const invalidateDelay = 150

// Subscribes to the backend mutation stream and refetches active queries. A
// short debounce folds bursts (drag, bulk edits) into one invalidation.
export function useLiveUpdates() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const controller = new AbortController()
    let stopped = false
    let retryTimer: number | undefined
    let invalidateTimer: number | undefined

    const scheduleInvalidate = () => {
      if (invalidateTimer) return
      invalidateTimer = window.setTimeout(() => {
        invalidateTimer = undefined
        queryClient.invalidateQueries()
      }, invalidateDelay)
    }

    const connect = async () => {
      try {
        const token = await getAccessToken()
        const response = await fetch('/api/events', {
          headers: {
            Accept: 'text/event-stream',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          signal: controller.signal,
        })
        if (!response.ok || !response.body) {
          throw new Error(`events: ${response.status}`)
        }
        scheduleInvalidate()
        const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
        let buffer = ''
        for (;;) {
          const { value, done } = await reader.read()
          if (done) break
          buffer += value
          let boundary = buffer.indexOf('\n\n')
          while (boundary >= 0) {
            const frame = buffer.slice(0, boundary)
            buffer = buffer.slice(boundary + 2)
            if (frame.startsWith('data:')) scheduleInvalidate()
            boundary = buffer.indexOf('\n\n')
          }
        }
      } catch {
        if (stopped) return
      }
      if (!stopped) retryTimer = window.setTimeout(connect, retryDelay)
    }

    connect()
    return () => {
      stopped = true
      controller.abort()
      if (retryTimer) window.clearTimeout(retryTimer)
      if (invalidateTimer) window.clearTimeout(invalidateTimer)
    }
  }, [queryClient])
}
