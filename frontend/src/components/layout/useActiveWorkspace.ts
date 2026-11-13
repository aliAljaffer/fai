import { useWorkspaces } from '@/api/hooks'
import { useWorkspaceStore } from '@/store/workspace'

export function useActiveWorkspace() {
  const { data: workspaces } = useWorkspaces()
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const stored = workspaces?.find((workspace) => workspace.id === activeWorkspaceId)
  return { workspaces, stored, active: stored ?? workspaces?.[0] }
}
