import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react'
import { Download, Trash2, TriangleAlert, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PaletteSelect, PaletteSwatches } from '@/components/ThemePalettePicker'
import { CalendarSourcesSection } from '@/components/calendar/CalendarSourcesSection'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  useAISettings,
  useAppearance,
  useBoards,
  useCalendarFeedURL,
  useColumns,
  useCreateWorkspace,
  useDeleteBoard,
  useExportBackup,
  useExportWorkspaceTasks,
  useImportWorkspaceTasks,
  useJiraImport,
  useJiraImportStatus,
  useResetWorkspace,
  useRestoreBackup,
  useUpdateAISettings,
  useUpdateAppearance,
  useUpdateWorkspace,
  useWorkspaces,
} from '../api/hooks'
import {
  appVersion,
  checkForUpdates,
  downloadUpdate,
  installUpdate,
  isDesktop,
  isManualUpdate,
  setAutoDownload,
  useUpdateState,
} from '../lib/updates'
import { allPalettes, customPaletteId, DEFAULT_PALETTE_ID, findPalette } from '../lib/themes'
import { columnColor, useColumnColors } from '../store/columnColors'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import type { AppearanceSettings, Workspace, WorkspaceTasksExport } from '../api/types'

const TAG_PATTERN = /^[A-Za-z]{3,4}$/

function serverError(error: unknown) {
  const message = error instanceof Error ? error.message : 'Request failed'
  const body = message.match(/\{.*\}$/)?.[0]
  if (!body) return message
  try {
    const parsed = JSON.parse(body) as { error?: string }
    return parsed.error ?? message
  } catch {
    return message
  }
}

function WorkspaceRow({ workspace }: { workspace: Workspace }) {
  const updateWorkspace = useUpdateWorkspace()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(workspace.name)
  const [tag, setTag] = useState(workspace.tag)
  const [error, setError] = useState('')

  function startEditing() {
    setName(workspace.name)
    setTag(workspace.tag)
    setError('')
    setEditing(true)
  }

  function save() {
    const trimmedName = name.trim()
    const trimmedTag = tag.trim()
    if (!trimmedName) {
      setError('Name is required.')
      return
    }
    if (!TAG_PATTERN.test(trimmedTag)) {
      setError('Tag must be 3 or 4 letters.')
      return
    }
    updateWorkspace.mutate(
      { id: workspace.id, name: trimmedName, tag: trimmedTag },
      {
        onSuccess: () => setEditing(false),
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  if (!editing) {
    return (
      <div className="flex items-center justify-between gap-2 rounded-md border px-2 py-1.5">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-sm">{workspace.name}</span>
          <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
            {workspace.tag}
          </span>
        </div>
        <Button size="xs" variant="ghost" onClick={startEditing}>
          Rename
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border px-2 py-1.5">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          autoFocus
          value={name}
          aria-label={`Name for ${workspace.name}`}
          aria-invalid={Boolean(error)}
          placeholder="Workspace name"
          className="h-7 max-w-48"
          onChange={(event) => setName(event.target.value)}
        />
        <Input
          value={tag}
          aria-label={`Tag for ${workspace.name}`}
          aria-invalid={Boolean(error)}
          placeholder="TAG"
          className="h-7 w-20 font-mono uppercase"
          onChange={(event) => setTag(event.target.value)}
        />
        <Button size="xs" disabled={updateWorkspace.isPending} onClick={save}>
          Save
        </Button>
        <Button size="xs" variant="ghost" onClick={() => setEditing(false)}>
          Cancel
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}

function WorkspacesCard() {
  const { data: workspaces } = useWorkspaces()
  const createWorkspace = useCreateWorkspace()
  const [name, setName] = useState('')
  const [tag, setTag] = useState('')
  const [error, setError] = useState('')

  function submit() {
    const trimmedName = name.trim()
    const trimmedTag = tag.trim()
    if (!trimmedName) {
      setError('Name is required.')
      return
    }
    if (!TAG_PATTERN.test(trimmedTag)) {
      setError('Tag must be 3 or 4 letters.')
      return
    }
    createWorkspace.mutate(
      { name: trimmedName, tag: trimmedTag },
      {
        onSuccess: () => {
          setName('')
          setTag('')
          setError('')
        },
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Workspaces</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Workspaces separate boards, notes, reminders, and calendars. Tasks are numbered per
          workspace and keep the tag they were created with after a rename.
        </p>
        <div className="flex flex-col gap-1">
          {workspaces?.map((workspace) => (
            <WorkspaceRow key={workspace.id} workspace={workspace} />
          ))}
          {!workspaces?.length && (
            <p className="text-sm text-muted-foreground">No workspaces yet.</p>
          )}
        </div>
        <div className="flex flex-col gap-2 border-t pt-3">
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={name}
              aria-label="New workspace name"
              aria-invalid={Boolean(error)}
              placeholder="Workspace name"
              className="max-w-48"
              onChange={(event) => setName(event.target.value)}
            />
            <Input
              value={tag}
              aria-label="New workspace tag"
              aria-invalid={Boolean(error)}
              placeholder="TAG"
              className="w-20 font-mono uppercase"
              onChange={(event) => setTag(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') submit()
              }}
            />
            <Button size="sm" disabled={createWorkspace.isPending} onClick={submit}>
              Create workspace
            </Button>
          </div>
          {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
        </div>
      </CardContent>
    </Card>
  )
}

function BoardsCard() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const deleteBoard = useDeleteBoard()
  const { selectedBoardId, setSelectedBoardId } = useUIStore()
  const [confirming, setConfirming] = useState<number | null>(null)
  const [confirmText, setConfirmText] = useState('')
  const board = boards?.find((item) => item.id === confirming)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Boards</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Deleting a board also deletes its columns and tasks. This cannot be undone.
        </p>
        {!boards?.length && <p className="text-sm text-muted-foreground">No boards yet.</p>}
        {boards?.map((item) => (
          <div key={item.id} className="flex flex-col gap-1 rounded-md border px-2 py-1.5">
            <div className="flex items-center gap-2">
              <span className="min-w-0 flex-1 truncate text-sm font-medium">{item.name}</span>
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label={`Delete ${item.name}`}
                onClick={() => {
                  setConfirming(item.id)
                  setConfirmText('')
                }}
              >
                <Trash2 />
              </Button>
            </div>
            {confirming === item.id && board && (
              <div className="flex flex-col gap-1 border-t pt-1">
                <label className="text-xs text-muted-foreground" htmlFor="board-delete-confirm">
                  Type {board.name} to confirm
                </label>
                <div className="flex items-center gap-2">
                  <Input
                    id="board-delete-confirm"
                    value={confirmText}
                    onChange={(event) => setConfirmText(event.target.value)}
                    className="h-7 text-xs"
                  />
                  <Button size="xs" variant="ghost" onClick={() => setConfirming(null)}>
                    Cancel
                  </Button>
                  <Button
                    size="xs"
                    variant="destructive"
                    disabled={confirmText !== board.name || deleteBoard.isPending}
                    onClick={() =>
                      deleteBoard.mutate(board.id, {
                        onSuccess: () => {
                          if (selectedBoardId === board.id) setSelectedBoardId(undefined)
                          setConfirming(null)
                          setConfirmText('')
                        },
                        onError: (error) => toast.error(serverError(error)),
                      })
                    }
                  >
                    Delete
                  </Button>
                </div>
              </div>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

function WorkspaceDataCard() {
  const { data: workspaces } = useWorkspaces()
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const exportTasks = useExportWorkspaceTasks()
  const importTasks = useImportWorkspaceTasks()
  const resetWorkspace = useResetWorkspace()
  const [workspaceId, setWorkspaceId] = useState<number>()
  const [confirmText, setConfirmText] = useState('')
  const [importError, setImportError] = useState('')
  const fileInput = useRef<HTMLInputElement>(null)

  const selectedWorkspaceId =
    workspaceId ??
    workspaces?.find((item) => item.id === activeWorkspaceId)?.id ??
    workspaces?.[0]?.id
  const workspace = workspaces?.find((item) => item.id === selectedWorkspaceId)
  const canReset = confirmText.trim() === 'I understand'

  function download() {
    if (!workspace) return
    exportTasks.mutate(workspace.id, {
      onSuccess: (data) => {
        const url = URL.createObjectURL(
          new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' }),
        )
        const link = document.createElement('a')
        link.href = url
        link.download = `fai-${workspace.tag.toLowerCase()}-tasks.json`
        document.body.appendChild(link)
        link.click()
        link.remove()
        URL.revokeObjectURL(url)
        toast.success(`Exported ${data.tasks.length} tasks`)
      },
      onError: (error) => toast.error(serverError(error)),
    })
  }

  async function importFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file || !workspace) return
    setImportError('')
    try {
      const data = JSON.parse(await file.text()) as WorkspaceTasksExport
      if (!data || !Array.isArray(data.tasks)) {
        throw new Error('This file is not a workspace task export.')
      }
      importTasks.mutate(
        { workspaceId: workspace.id, data },
        {
          onSuccess: (result) => toast.success(`Imported ${result.imported} tasks`),
          onError: (error) => setImportError(serverError(error)),
        },
      )
    } catch (error) {
      setImportError(error instanceof Error ? error.message : 'Could not read the file.')
    }
  }

  function reset() {
    if (!workspace || !canReset) return
    resetWorkspace.mutate(workspace.id, {
      onSuccess: (result) => {
        toast.success(`Deleted ${result.deleted} tasks`)
        setConfirmText('')
      },
      onError: (error) => toast.error(serverError(error)),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Workspace data</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Export the selected workspace's tasks as JSON, or import a previous export. Import
          adds tasks to the workspace and keeps existing tasks. Task numbers are preserved
          from the file, so an import fails if a number is already in use. Boards and columns
          referenced by the file are created when missing.
        </p>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium" htmlFor="workspace-data-workspace">
            Workspace
          </label>
          <Select
            items={(workspaces ?? []).map((item) => ({
              value: String(item.id),
              label: `${item.name} (${item.tag})`,
            }))}
            value={workspace ? String(workspace.id) : undefined}
            onValueChange={(value) => value && setWorkspaceId(Number(value))}
          >
            <SelectTrigger id="workspace-data-workspace" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {workspaces?.map((item) => (
                <SelectItem key={item.id} value={String(item.id)}>
                  {item.name}
                  <span className="text-muted-foreground">({item.tag})</span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={!workspace || exportTasks.isPending}
            onClick={download}
          >
            <Download />
            {exportTasks.isPending ? 'Exporting…' : 'Export JSON'}
          </Button>
          <input
            ref={fileInput}
            type="file"
            accept="application/json,.json"
            className="hidden"
            onChange={(event) => void importFile(event)}
          />
          <Button
            size="sm"
            variant="outline"
            disabled={!workspace || importTasks.isPending}
            onClick={() => fileInput.current?.click()}
          >
            <Upload />
            {importTasks.isPending ? 'Importing…' : 'Import JSON'}
          </Button>
        </div>
        {importError && (
          <p role="alert" className="text-xs text-destructive">
            {importError}
          </p>
        )}

        <div className="flex flex-col gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-3">
          <div className="flex items-center gap-2 text-sm font-medium text-destructive">
            <TriangleAlert className="size-4" />
            Reset workspace
          </div>
          <p className="text-xs text-muted-foreground">
            Deletes every task in {workspace?.name ?? 'the selected workspace'} and resets the
            task number to 1. Boards and columns stay. This cannot be undone. Export first if
            you need a copy.
          </p>
          <Input
            value={confirmText}
            aria-label="Reset confirmation"
            placeholder='Type "I understand"'
            onChange={(event) => setConfirmText(event.target.value)}
          />
          <Button
            size="sm"
            variant="destructive"
            disabled={!workspace || !canReset || resetWorkspace.isPending}
            onClick={reset}
          >
            {resetWorkspace.isPending ? 'Resetting…' : 'Delete all tasks and reset'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function BackupCard() {
  const exportBackup = useExportBackup()
  const restoreBackup = useRestoreBackup()
  const [file, setFile] = useState<File | null>(null)
  const [confirmText, setConfirmText] = useState('')
  const [error, setError] = useState('')
  const fileInput = useRef<HTMLInputElement>(null)

  const canRestore = file !== null && confirmText.trim() === 'I understand'

  function pickFile(event: ChangeEvent<HTMLInputElement>) {
    const chosen = event.target.files?.[0]
    event.target.value = ''
    setFile(chosen ?? null)
    setError('')
  }

  function restore() {
    if (!file || !canRestore) return
    setError('')
    restoreBackup.mutate(
      { file, confirm: confirmText.trim() },
      {
        onSuccess: (result) => {
          toast.success(
            `Restored ${result.counts.tasks ?? 0} tasks, ${result.counts.notes ?? 0} notes, and ${result.counts.attachments ?? 0} attachments`,
          )
          setFile(null)
          setConfirmText('')
          useWorkspaceStore.getState().setActiveWorkspaceId(undefined)
          useUIStore.getState().setSelectedBoardId(undefined)
        },
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Backup &amp; restore</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Download a full backup of this Fai instance: workspaces, boards, tasks, notes,
          reminders, comments, time entries, calendar sources, Jira links, attachments, and
          appearance. AI keys and integration secrets are not included; re-enter them after a
          restore. Treat the backup file as sensitive: it contains notes and private calendar
          URLs.
        </p>

        <div>
          <Button
            size="sm"
            variant="outline"
            disabled={exportBackup.isPending}
            onClick={() =>
              exportBackup.mutate(undefined, {
                onSuccess: () => toast.success('Backup downloaded'),
                onError: (err) => {
                  if (err instanceof Error && err.name === 'AbortError') return
                  toast.error(serverError(err))
                },
              })
            }
          >
            <Download />
            {exportBackup.isPending ? 'Preparing…' : 'Download backup'}
          </Button>
        </div>

        <div className="flex flex-col gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-3">
          <div className="flex items-center gap-2 text-sm font-medium text-destructive">
            <TriangleAlert className="size-4" />
            Restore from backup
          </div>
          <p className="text-xs text-muted-foreground">
            Restoring replaces every workspace, task, note, and attachment in this instance.
            Create a backup first if you need the current data. This cannot be undone.
          </p>
          <input
            ref={fileInput}
            type="file"
            accept="application/zip,.zip"
            className="hidden"
            onChange={pickFile}
          />
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" variant="outline" onClick={() => fileInput.current?.click()}>
              <Upload />
              {file ? 'Choose another file' : 'Choose backup file'}
            </Button>
            {file && <span className="truncate text-xs text-muted-foreground">{file.name}</span>}
          </div>
          <Input
            value={confirmText}
            aria-label="Restore confirmation"
            placeholder='Type "I understand"'
            onChange={(event) => setConfirmText(event.target.value)}
          />
          <Button
            size="sm"
            variant="destructive"
            disabled={!canRestore || restoreBackup.isPending}
            onClick={restore}
          >
            {restoreBackup.isPending ? 'Restoring…' : 'Replace instance'}
          </Button>
          {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
        </div>
      </CardContent>
    </Card>
  )
}

const CUSTOM_PALETTE_COLORS = ['#F8F9FA', '#E9ECEF', '#ADB5BD', '#6C757D', '#212529']
const NO_CUSTOM_PALETTES: AppearanceSettings['custom_palettes'] = []

function AppearanceCard() {
  const { data: workspaces } = useWorkspaces()
  const { data: appearance, isError, refetch } = useAppearance()
  const updateAppearance = useUpdateAppearance()
  const customPalettes = appearance?.custom_palettes ?? NO_CUSTOM_PALETTES
  const workspacePalettes = appearance?.workspace_palettes ?? {}
  const globalPaletteId = appearance?.global_palette_id ?? DEFAULT_PALETTE_ID
  const palettes = useMemo(() => allPalettes(customPalettes), [customPalettes])
  const globalPalette = findPalette(globalPaletteId, customPalettes)
  const [name, setName] = useState('')
  const [colors, setColors] = useState<string[]>(CUSTOM_PALETTE_COLORS)
  const pending = updateAppearance.isPending

  function save(next: AppearanceSettings) {
    updateAppearance.mutate(
      {
        global_palette_id: next.global_palette_id,
        workspace_palettes: next.workspace_palettes,
        custom_palettes: next.custom_palettes,
      },
      { onError: (error) => toast.error(serverError(error)) },
    )
  }

  function addPalette() {
    if (!appearance) return
    const trimmed = name.trim()
    if (!trimmed) return
    save({
      ...appearance,
      custom_palettes: [
        ...customPalettes,
        { id: customPaletteId(trimmed), name: trimmed, colors: [...colors] },
      ],
    })
    setName('')
  }

  function removePalette(id: string) {
    if (!appearance) return
    const nextWorkspaces = Object.fromEntries(
      Object.entries(workspacePalettes).filter(([, paletteId]) => paletteId !== id),
    )
    save({
      ...appearance,
      global_palette_id: globalPaletteId === id ? DEFAULT_PALETTE_ID : globalPaletteId,
      workspace_palettes: nextWorkspaces,
      custom_palettes: customPalettes.filter((palette) => palette.id !== id),
    })
  }

  function setGlobal(id: string) {
    if (appearance) save({ ...appearance, global_palette_id: id })
  }

  function setWorkspace(workspaceId: number, id: string | null) {
    if (!appearance) return
    const next = { ...workspacePalettes }
    if (id) next[String(workspaceId)] = id
    else delete next[String(workspaceId)]
    save({ ...appearance, workspace_palettes: next })
  }

  if (!appearance) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Appearance</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            {isError ? 'Could not load appearance settings.' : 'Loading appearance settings…'}
          </p>
          {isError && (
            <Button size="sm" variant="outline" onClick={() => void refetch()}>
              Retry
            </Button>
          )}
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Appearance</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Pick one palette for every workspace, then override it per workspace. Workspace
          overrides follow the active workspace. Changes apply to every machine.
        </p>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium">Global theme</span>
            {pending && <span className="text-xs text-muted-foreground">Saving…</span>}
          </div>
          <PaletteSelect
            value={globalPalette.id}
            palettes={palettes}
            onChange={setGlobal}
            ariaLabel="Global theme"
            disabled={pending}
          />
          <div className="flex items-center gap-2 rounded-md border px-2 py-1.5">
            <PaletteSwatches colors={globalPalette.colors} />
            <span className="text-sm">{globalPalette.name}</span>
            {globalPalette.custom && (
              <span className="rounded border px-1 text-[0.65rem] text-muted-foreground">
                Custom
              </span>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2 border-t pt-3">
          <span className="text-sm font-medium">Workspace overrides</span>
          {workspaces?.map((workspace) => {
            const overrideId = workspacePalettes[String(workspace.id)]
            const effective = findPalette(overrideId ?? globalPalette.id, customPalettes)
            const overrideValue =
              overrideId && palettes.some((palette) => palette.id === overrideId)
                ? overrideId
                : 'inherit'
            return (
              <div
                key={workspace.id}
                className="flex flex-col gap-2 rounded-md border px-2 py-1.5 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex min-w-0 items-center gap-2">
                  <span className="truncate text-sm">{workspace.name}</span>
                  <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
                    {workspace.tag}
                  </span>
                  <PaletteSwatches colors={effective.colors} className="hidden sm:inline-flex" />
                </div>
                <div className="w-full sm:w-64">
                  <PaletteSelect
                    value={overrideValue}
                    palettes={palettes}
                    inheritLabel={`Use global (${globalPalette.name})`}
                    onChange={(value) =>
                      setWorkspace(workspace.id, value === 'inherit' ? null : value)
                    }
                    ariaLabel={`Theme for ${workspace.name}`}
                    disabled={pending}
                  />
                </div>
              </div>
            )
          })}
          {!workspaces?.length && (
            <p className="text-sm text-muted-foreground">No workspaces yet.</p>
          )}
        </div>

        <div className="flex flex-col gap-3 border-t pt-3">
          <div>
            <span className="text-sm font-medium">Custom palette</span>
            <p className="text-xs text-muted-foreground">
              Pick five colors. Custom palettes appear in both selectors.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <input
              value={name}
              aria-label="Custom palette name"
              placeholder="Palette name"
              className="h-8 min-w-40 flex-1 rounded-lg border border-input bg-transparent px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
              onChange={(event) => setName(event.target.value)}
              onKeyDown={(event) => {
                if (event.nativeEvent.isComposing) return
                if (event.key === 'Enter') addPalette()
              }}
            />
            <Button size="sm" disabled={!name.trim() || pending} onClick={addPalette}>
              Add palette
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {colors.map((color, index) => (
              <input
                key={index}
                type="color"
                value={color}
                aria-label={`Custom palette color ${index + 1}`}
                className="h-8 w-10 shrink-0 cursor-pointer rounded border border-input bg-transparent p-0.5"
                onChange={(event) => {
                  const value = event.target.value
                  setColors((current) =>
                    current.map((item, itemIndex) => (itemIndex === index ? value : item)),
                  )
                }}
              />
            ))}
            <PaletteSwatches colors={colors} />
          </div>
          {customPalettes.length > 0 && (
            <div className="flex flex-col gap-1">
              {customPalettes.map((palette) => (
                <div
                  key={palette.id}
                  className="flex items-center justify-between gap-2 rounded-md border px-2 py-1.5"
                >
                  <div className="flex min-w-0 items-center gap-2">
                    <PaletteSwatches colors={palette.colors} />
                    <span className="truncate text-sm">{palette.name}</span>
                  </div>
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label={`Remove ${palette.name}`}
                    disabled={pending}
                    onClick={() => removePalette(palette.id)}
                  >
                    <Trash2 />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

const AI_PROVIDERS = [
  { value: 'anthropic', label: 'Anthropic', model: 'claude-sonnet-5' },
  { value: 'openai', label: 'OpenAI', model: 'gpt-4o-mini' },
  { value: 'openai-compat', label: 'OpenAI compatible', model: 'gpt-4o-mini' },
] as const

type AIProviderValue = (typeof AI_PROVIDERS)[number]['value']

function aiModelPlaceholder(provider: AIProviderValue) {
  return AI_PROVIDERS.find((item) => item.value === provider)?.model ?? ''
}

function AICard() {
  const { data: settings } = useAISettings()
  const updateSettings = useUpdateAISettings()
  const [provider, setProvider] = useState<AIProviderValue>('anthropic')
  const [baseURL, setBaseURL] = useState('')
  const [model, setModel] = useState('')
  const [apiKey, setAPIKey] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!settings) return
    setProvider(settings.provider)
    setBaseURL(settings.base_url)
    setModel(settings.model)
    setAPIKey('')
    setError('')
  }, [settings])

  function save() {
    setError('')
    updateSettings.mutate(
      {
        provider,
        base_url: baseURL.trim(),
        model: model.trim(),
        api_key: apiKey.trim() || undefined,
      },
      {
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  const status = !settings
    ? 'Loading AI settings…'
    : settings.configured
      ? 'AI is configured.'
      : 'AI is not configured. Add an API key to enable it.'
  const sourceNote =
    settings?.source === 'environment'
      ? 'Currently using server environment settings. Saving here overrides them.'
      : 'Choose the provider and model used for chat, task breakdown, note plans, and meeting extraction.'

  return (
    <Card>
      <CardHeader>
        <CardTitle>AI provider</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">{status}</p>
        <p className="text-xs text-muted-foreground">{sourceNote}</p>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium" htmlFor="ai-provider">
            Provider
          </label>
          <Select
            items={AI_PROVIDERS.map((item) => ({ value: item.value, label: item.label }))}
            value={provider}
            onValueChange={(value) => value && setProvider(value as AIProviderValue)}
          >
            <SelectTrigger id="ai-provider" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {AI_PROVIDERS.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium" htmlFor="ai-model">
            Model
          </label>
          <Input
            id="ai-model"
            value={model}
            placeholder={aiModelPlaceholder(provider)}
            onChange={(event) => setModel(event.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium" htmlFor="ai-base-url">
            Base URL
          </label>
          <Input
            id="ai-base-url"
            value={baseURL}
            placeholder={
              provider === 'openai'
                ? 'https://api.openai.com/v1'
                : provider === 'openai-compat'
                  ? 'https://your-endpoint/v1'
                  : 'Anthropic default'
            }
            onChange={(event) => setBaseURL(event.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium" htmlFor="ai-api-key">
            API key
          </label>
          <Input
            id="ai-api-key"
            type="password"
            value={apiKey}
            autoComplete="off"
            placeholder={settings?.has_api_key ? 'Saved, leave blank to keep' : 'Paste the API key'}
            onChange={(event) => setAPIKey(event.target.value)}
          />
        </div>

        {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
        <div className="flex items-center gap-2">
          <Button size="sm" disabled={updateSettings.isPending} onClick={save}>
            {updateSettings.isPending ? 'Saving…' : 'Save AI settings'}
          </Button>
          {settings?.source === 'database' && settings.has_api_key && (
            <Button
              size="sm"
              variant="ghost"
              disabled={updateSettings.isPending}
              onClick={() => {
                setError('')
                updateSettings.mutate(
                  {
                    provider,
                    base_url: baseURL.trim(),
                    model: model.trim(),
                    clear_api_key: true,
                  },
                  {
                    onError: (err) => setError(serverError(err)),
                  },
                )
              }}
            >
              Remove stored key
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function AboutCard() {
  const state = useUpdateState()
  const desktop = isDesktop()
  const manual = isManualUpdate()
  const busy = state.status === 'checking' || state.status === 'downloading'

  const status =
    state.status === 'checking'
      ? 'Checking for updates…'
      : state.status === 'available'
        ? manual
          ? `Version ${state.version ?? ''} is available. Download it from GitHub.`
          : `Version ${state.version ?? ''} is available. Download it when you are ready.`
        : state.status === 'downloading'
          ? `Downloading version ${state.version ?? ''} (${Math.round(state.progress)}%).`
          : state.status === 'downloaded'
            ? `Version ${state.version ?? ''} is ready to install.`
            : state.status === 'error'
              ? `Update check failed: ${state.error ?? 'unknown error'}`
              : state.autoDownload
                ? 'fai checks for a new version every six hours and downloads it automatically.'
                : 'fai checks for a new version every six hours and tells you when one is available.'

  return (
    <Card>
      <CardHeader>
        <CardTitle>About</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-center justify-between text-sm">
          <span>Version</span>
          <span className="font-mono text-muted-foreground">{appVersion()}</span>
        </div>
        {desktop ? (
          <>
            <p className="text-sm text-muted-foreground">{status}</p>
            {!manual && (
              <label className="flex items-start justify-between gap-3 rounded-md border px-2 py-1.5">
                <span className="flex flex-col gap-0.5">
                  <span className="text-sm">Download updates automatically</span>
                  <span className="text-xs text-muted-foreground">
                    When off, fai announces a new version and downloads it only when you ask.
                  </span>
                </span>
                <Input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4"
                  checked={state.autoDownload}
                  aria-label="Download updates automatically"
                  onChange={(event) => void setAutoDownload(event.target.checked)}
                />
              </label>
            )}
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={busy}
                onClick={() => void checkForUpdates()}
              >
                Check for updates
              </Button>
              {state.status === 'available' && (
                <Button
                  size="sm"
                  className="update-indicator"
                  onClick={() => void downloadUpdate()}
                >
                  {manual ? 'Open the download page' : `Download ${state.version}`}
                </Button>
              )}
              {state.status === 'downloaded' && (
                <Button
                  size="sm"
                  className="update-indicator"
                  onClick={() => void installUpdate()}
                >
                  Restart to install {state.version}
                </Button>
              )}
            </div>
          </>
        ) : (
          <p className="text-sm text-muted-foreground">
            Updates are delivered by the desktop app.
          </p>
        )}
      </CardContent>
    </Card>
  )
}

export function Settings() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { data: workspaces } = useWorkspaces()
  const jiraImport = useJiraImport()
  const jiraImportStatus = useJiraImportStatus()
  const { data: feed } = useCalendarFeedURL(workspaceId)
  const selectedBoardId = useUIStore((state) => state.selectedBoardId)
  const board =
    boards?.find((b) => b.id === selectedBoardId) ?? boards?.[0]
  const boardId = board?.id
  const { data: columns } = useColumns(boardId)
  const { colors, setColor } = useColumnColors()
  const { data: allBoards } = useBoards()
  const [targetBoardId, setTargetBoardId] = useState<number>()
  const importBoard =
    allBoards?.find((b) => b.id === (targetBoardId ?? boardId)) ?? allBoards?.[0]
  const workspaceName = (id: number) =>
    workspaces?.find((w) => w.id === id)?.name ?? 'Unknown workspace'
  const [jql, setJql] = useState('')
  const [forceUpdate, setForceUpdate] = useState(false)

  return (
    <div className="flex flex-col gap-3">
      <h1 className="text-lg font-semibold">Settings</h1>
      <div className="columns-1 gap-3 [&>*]:mb-3 [&>*]:break-inside-avoid sm:columns-2 xl:columns-3">
      <WorkspacesCard />

      <WorkspaceDataCard />

      <BoardsCard />

      <BackupCard />

      <CalendarSourcesSection />

      <AppearanceCard />

      <AICard />

      <Card>
        <CardHeader>
          <CardTitle>Jira import</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            Imports only Jira issues assigned to you. Epics become arcs, sub-tasks become steps,
            and every other issue type becomes a task. Items go to the selected board and its
            workspace. Leave the JQL field blank to use the server-configured default project.
            Already-imported issues are skipped unless "force update" is checked. Completion dates
            are taken from Jira.
          </p>
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium" htmlFor="jira-import-board">
              Target board
            </label>
            <Select
              items={(allBoards ?? []).map((b) => ({
                value: String(b.id),
                label: `${workspaceName(b.workspace_id)} / ${b.name}`,
              }))}
              value={importBoard ? String(importBoard.id) : undefined}
              onValueChange={(value) => value && setTargetBoardId(Number(value))}
            >
              <SelectTrigger id="jira-import-board" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {allBoards?.map((b) => (
                  <SelectItem key={b.id} value={String(b.id)}>
                    {workspaceName(b.workspace_id)}
                    <span className="text-muted-foreground">/ {b.name}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Textarea
            aria-label="Jira JQL"
            placeholder='Optional JQL, e.g. project = "DEV" AND status != Done'
            value={jql}
            onChange={(e) => setJql(e.target.value)}
            rows={3}
          />
          <label className="flex items-center gap-2 text-sm">
            <Input
              type="checkbox"
              className="h-4 w-4"
              checked={forceUpdate}
              onChange={(e) => setForceUpdate(e.target.checked)}
            />
            Force update already-imported issues
          </label>
          <Button
            disabled={!importBoard || jiraImport.isPending || jiraImportStatus.data?.running}
            onClick={() => {
              if (!importBoard) return
              jiraImport.mutate(
                { board_id: importBoard.id, jql: jql || undefined, force_update: forceUpdate },
                {
                  onSuccess: () => toast.success('Jira import started'),
                  onError: (err) => toast.error(err.message),
                },
              )
            }}
          >
            {jiraImportStatus.data?.running ? 'Import running…' : 'Import from Jira'}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Status colors</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <p className="text-sm text-muted-foreground">
            Override the status pill colors in the List view. Defaults follow the active theme. Saved in this browser only.
          </p>
          {columns?.map((column) => (
            <label
              key={column.id}
              className="flex items-center gap-2 rounded-md border px-2 py-1.5"
            >
              <input
                type="color"
                value={columnColor(column, colors)}
                aria-label={`Status color for ${column.name}`}
                className="h-6 w-6 shrink-0 cursor-pointer rounded border border-input bg-transparent p-0.5"
                onChange={(event) => setColor(column.id, event.target.value)}
              />
              <span className="text-sm">{column.name}</span>
            </label>
          ))}
          {!columns?.length && (
            <p className="text-sm text-muted-foreground">This board has no columns yet.</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Calendar subscription</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            Subscribe from Apple Calendar (File → New Calendar Subscription, or Settings → Calendar →
            Accounts → Add Account → Other → Add Subscribed Calendar). Open tasks with a due date
            appear as all-day events and refresh about every 30 minutes. The feed contains Fai
            tasks only. Calendars imported in the app stay out of it.
          </p>
          <div className="flex gap-2">
            <Input
              readOnly
              aria-label="Calendar subscription URL"
              value={feed?.url ?? ''}
              placeholder={feed ? '' : 'Calendar feed is not configured'}
              onFocus={(e) => e.currentTarget.select()}
            />
            <Button
              variant="outline"
              disabled={!feed}
              onClick={() => {
                if (!feed) return
                navigator.clipboard.writeText(feed.url).then(
                  () => toast.error('Could not copy the URL'),
                )
              }}
            >
              Copy
            </Button>
          </div>
          {feed && (
            <a className="text-sm text-primary underline" href={feed.url.replace(/^https:/, 'webcal:')}>
              Open in Calendar
            </a>
          )}
          <p className="text-xs text-muted-foreground">
            The URL contains a secret token — treat it like a password. Rotating ICS_TOKEN
            invalidates existing subscriptions.
          </p>
        </CardContent>
      </Card>

      <AboutCard />
      </div>
    </div>
  )
}
