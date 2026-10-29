import { ArrowDownToLine, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { downloadUpdate, installUpdate, useUpdateState } from '@/lib/updates'

export function UpdateIndicator() {
  const state = useUpdateState()
  if (
    state.status !== 'available' &&
    state.status !== 'downloading' &&
    state.status !== 'downloaded'
  ) {
    return null
  }

  const ready = state.status === 'downloaded'
  const available = state.status === 'available'
  const label = ready
    ? `Restart to update ${state.version ?? ''}`.trim()
    : available
      ? `Update ${state.version ?? ''}`.trim()
      : `Downloading ${Math.round(state.progress)}%`

  return (
    <Button
      size="sm"
      className="update-indicator ml-1 font-medium"
      title={
        ready
          ? `Restart fai to install version ${state.version ?? ''}`
          : available
            ? `Download version ${state.version ?? ''}`
            : `Version ${state.version ?? ''} is downloading`
      }
      aria-label={ready ? `Install update ${state.version ?? ''}` : 'Update available'}
      onClick={() => {
        if (ready) void installUpdate()
        else if (available) void downloadUpdate()
      }}
    >
      {state.status === 'downloading' ? (
        <Loader2 className="size-3.5 animate-spin" />
      ) : (
        <ArrowDownToLine className="size-3.5" />
      )}
      {label}
      {ready && <span className="update-indicator-dot size-1.5 animate-pulse rounded-full" />}
    </Button>
  )
}
