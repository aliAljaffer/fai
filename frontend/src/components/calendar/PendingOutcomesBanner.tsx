import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { usePendingMeetingOutcomes } from '../../api/hooks'

export function PendingOutcomesBanner() {
  const { data } = usePendingMeetingOutcomes()
  const navigate = useNavigate()

  if (!data?.length) return null

  return (
    <div className="flex flex-col gap-2 px-6 pt-3">
      {data.map((outcome) => (
        <div
          key={outcome.id}
          className="flex items-center justify-between rounded-md border bg-accent px-3 py-2 text-sm"
        >
          <span>Meeting "{outcome.title || 'Untitled'}" just ended — log the outcome?</span>
          <Button
            size="sm"
            onClick={() => navigate('/calendar', { state: { outcome } })}
          >
            Log outcome
          </Button>
        </div>
      ))}
    </div>
  )
}
