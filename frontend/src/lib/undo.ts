import { toast } from 'sonner'

let latest: (() => Promise<void> | void) | null = null

export function showUndo(message: string, undo: () => Promise<void> | void) {
  let used = false
  const run = async () => {
    if (used) return
    used = true
    if (latest === run) latest = null
    try {
      await undo()
    } catch {
      toast.error('Could not undo')
    }
  }
  latest = run
  toast(message, {
    duration: 8000,
    action: { label: 'Undo', onClick: () => void run() },
    onDismiss: () => {
      if (latest === run) latest = null
    },
    onAutoClose: () => {
      if (latest === run) latest = null
    },
  })
  return run
}

export function undoLatest() {
  const run = latest
  latest = null
  if (run) void run()
}
