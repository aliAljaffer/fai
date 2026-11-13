import { ChevronDown, Pencil, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import type { Board } from '../../api/types'

export type BoardEditorMode = 'new' | 'rename'

export function BoardSwitcher({
  boards,
  boardId,
  editor,
  name,
  saving,
  onSelect,
  onStartEditing,
  onNameChange,
  onSubmit,
  onCancel,
}: {
  boards: Board[]
  boardId: number | undefined
  editor: BoardEditorMode | null
  name: string
  saving: boolean
  onSelect: (boardId: number) => void
  onStartEditing: (mode: BoardEditorMode) => void
  onNameChange: (name: string) => void
  onSubmit: () => void
  onCancel: () => void
}) {
  const current = boards.find((board) => board.id === boardId)

  if (editor) {
    return (
      <div className="flex min-w-0 items-center gap-1">
        <Input
          autoFocus
          value={name}
          placeholder="Board name"
          aria-label={editor === 'rename' ? 'Rename board' : 'New board name'}
          className="h-8 w-56"
          disabled={saving}
          onChange={(event) => onNameChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === 'Enter') {
              event.preventDefault()
              onSubmit()
            }
            if (event.key === 'Escape') {
              event.preventDefault()
              onCancel()
            }
          }}
        />
        <Button size="xs" disabled={!name.trim() || saving} onClick={onSubmit}>
          {editor === 'rename' ? 'Save' : 'Create'}
        </Button>
        <Button size="xs" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    )
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            aria-label="Switch board"
            className="-ml-2 max-w-full min-w-0 gap-1 px-2 text-sm font-semibold"
          />
        }
      >
        <span className="truncate">{current?.name ?? 'Board'}</span>
        <ChevronDown className="size-4 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-48">
        <DropdownMenuRadioGroup
          value={boardId ? String(boardId) : undefined}
          onValueChange={(value) => onSelect(Number(value))}
        >
          {boards.map((board) => (
            <DropdownMenuRadioItem key={board.id} value={String(board.id)}>
              {board.name}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled={!current} onClick={() => onStartEditing('rename')}>
          <Pencil />
          Rename board
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onStartEditing('new')}>
          <Plus />
          New board
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
