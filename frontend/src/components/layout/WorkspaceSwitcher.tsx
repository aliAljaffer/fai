import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { useUIStore } from '@/store/ui'
import { useWorkspaceStore } from '@/store/workspace'
import { useActiveWorkspace } from './useActiveWorkspace'

export function WorkspaceSwitcher({ className }: { className?: string }) {
  const { workspaces, active } = useActiveWorkspace()
  const setActiveWorkspaceId = useWorkspaceStore((state) => state.setActiveWorkspaceId)
  const setSelectedBoardId = useUIStore((state) => state.setSelectedBoardId)

  if (!workspaces?.length) return null

  return (
    <Select
      items={workspaces.map((workspace) => ({
        value: String(workspace.id),
        label: workspace.name,
      }))}
      value={active ? String(active.id) : undefined}
      onValueChange={(value) => {
        if (!value) return
        setActiveWorkspaceId(Number(value))
        setSelectedBoardId(undefined)
      }}
    >
      <SelectTrigger aria-label="Switch workspace" className={cn('min-w-0', className)}>
        <span className="flex min-w-0 flex-1 items-center gap-2">
          <span
            aria-hidden
            className="grid size-[22px] shrink-0 place-items-center rounded-[7px] bg-primary font-mono text-[0.6875rem] font-bold text-primary-foreground"
          >
            {active?.name.charAt(0).toUpperCase()}
          </span>
          <span className="truncate">
            <SelectValue />
          </span>
        </span>
      </SelectTrigger>
      <SelectContent align="start">
        {workspaces.map((workspace) => (
          <SelectItem key={workspace.id} value={String(workspace.id)}>
            {workspace.name}
            <span className="font-mono text-[0.65rem] text-muted-foreground">{workspace.tag}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
