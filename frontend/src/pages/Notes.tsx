import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useCreateNote, useNotes } from '../api/hooks'

export function Notes() {
  const { data: notes } = useNotes()
  const createNote = useCreateNote()
  const navigate = useNavigate()
  const [adding, setAdding] = useState(false)
  const [title, setTitle] = useState('')

  async function create() {
    const value = title.trim()
    if (!value) return
    try {
      const note = await createNote.mutateAsync({ title: value, body_md: '' })
      setTitle('')
      setAdding(false)
      navigate(`/notes/${note.id}`)
    } catch (error) {
      toast.error((error as Error).message)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex justify-end">
        {adding ? (
          <div className="flex items-center gap-2">
            <Input
              autoFocus
              value={title}
              placeholder="Note title"
              className="w-64"
              onChange={(event) => setTitle(event.target.value)}
              onKeyDown={(event) => {
                if (event.nativeEvent.isComposing) return
                if (event.key === 'Enter') void create()
                if (event.key === 'Escape') {
                  setAdding(false)
                  setTitle('')
                }
              }}
            />
            <Button disabled={!title.trim() || createNote.isPending} onClick={() => void create()}>
              Create
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setAdding(false)
                setTitle('')
              }}
            >
              Cancel
            </Button>
          </div>
        ) : (
          <Button onClick={() => setAdding(true)}>+ New note</Button>
        )}
      </div>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {notes?.map((note) => (
          <Card key={note.id} className="cursor-pointer" onClick={() => navigate(`/notes/${note.id}`)}>
            <CardContent>
              <div className="font-medium">{note.title}</div>
              <div className="mt-1 line-clamp-2 text-sm text-muted-foreground">{note.body_md}</div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
