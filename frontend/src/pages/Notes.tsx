import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { useCreateNote, useNotes } from '../api/hooks'

export function Notes() {
  const { data: notes } = useNotes()
  const createNote = useCreateNote()
  const navigate = useNavigate()

  return (
    <div className="flex flex-col gap-3">
      <div className="flex justify-end">
        <Button
          onClick={async () => {
            const title = prompt('Note title?')
            if (!title) return
            const note = await createNote.mutateAsync({ title, body_md: '' })
            navigate(`/notes/${note.id}`)
          }}
        >
          + New note
        </Button>
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
