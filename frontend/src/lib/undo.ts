import { createElement } from 'react'
import { toast } from 'sonner'
import { UndoToast } from '../components/UndoToast'

let latest: (() => Promise<void> | void) | null = null

export function showUndo(message: string, undo: () => Promise<void> | void) {
  let used = false
  let toastId: string | number | undefined
  const run = async () => {
    if (used) return
    used = true
    if (latest === run) latest = null
    if (toastId !== undefined) toast.dismiss(toastId)
    try {
      await undo()
    } catch {
      toast.error('Could not undo')
    }
  }
  latest = run
  toastId = toast.custom(
    (id) =>
      createElement(UndoToast, {
        toastId: id,
        message,
        onUndo: () => void run(),
      }),
    {
      duration: Infinity,
      onDismiss: () => {
        if (latest === run) latest = null
      },
      onAutoClose: () => {
        if (latest === run) latest = null
      },
    },
  )
  return run
}

export function undoLatest() {
  const run = latest
  latest = null
  if (run) void run()
}
