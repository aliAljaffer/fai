import { ExternalLink, MessageSquarePlus } from 'lucide-react'
import { format, parseISO } from 'date-fns'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { ImportedEvent } from '../../api/types'

export function ImportedEventDialog({
  event,
  sourceName,
  date,
  onOpenChange,
  onMeetingSummary,
}: {
  event: ImportedEvent
  sourceName: string
  date: string
  onOpenChange: (open: boolean) => void
  onMeetingSummary: () => void
}) {
  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md rounded-lg">
        <DialogHeader>
          <DialogTitle>{event.title}</DialogTitle>
          <DialogDescription>
            {sourceName}, {format(parseISO(date), 'd MMM yyyy')}
            {event.location ? `, ${event.location}` : ''}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-wrap gap-2">
          {event.url && (
            <Button
              variant="outline"
              nativeButton={false}
              render={<a href={event.url} target="_blank" rel="noopener noreferrer" />}
            >
              <ExternalLink />
              Open event
            </Button>
          )}
          <Button onClick={onMeetingSummary}>
            <MessageSquarePlus />
            Add meeting summary
          </Button>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
