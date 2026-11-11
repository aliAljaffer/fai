import { Repeat as RepeatIcon } from 'lucide-react'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { formatRepeat, REPEAT_OPTIONS } from '../../lib/repeat'
import type { Repeat } from '../../api/types'

const NONE = 'none'
const ITEMS = [{ value: NONE, label: 'None' }, ...REPEAT_OPTIONS]

export function RepeatIndicator({ repeat, className }: { repeat: Repeat; className?: string }) {
  const label = formatRepeat(repeat)
  return (
    <span title={label} className={cn('inline-flex shrink-0', className)}>
      <RepeatIcon className="size-3" aria-hidden />
      <span className="sr-only">{label}</span>
    </span>
  )
}

export function RepeatPicker({
  id,
  value,
  onChange,
  disabled,
  className,
}: {
  id?: string
  value: Repeat | null
  onChange: (value: Repeat | null) => void
  disabled?: boolean
  className?: string
}) {
  function commitInterval(input: HTMLInputElement) {
    if (!value) return
    const next = Math.max(1, Math.floor(Number(input.value)) || 1)
    input.value = String(next)
    if (next !== value.interval) onChange({ ...value, interval: next })
  }

  return (
    <div className={cn('flex items-center gap-1', className)}>
      <Select
        items={ITEMS}
        value={value?.freq ?? NONE}
        disabled={disabled}
        onValueChange={(next) => {
          if (next === NONE) onChange(null)
          else if (next) onChange({ freq: next as Repeat['freq'], interval: value?.interval ?? 1 })
        }}
      >
        <SelectTrigger id={id} aria-label="Repeat" size="sm" className="min-w-0 flex-1 bg-background">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {ITEMS.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {value && (
        <Input
          type="number"
          min={1}
          step={1}
          key={value.interval}
          defaultValue={value.interval}
          disabled={disabled}
          aria-label="Repeat every"
          className="h-7 w-16 bg-background"
          onBlur={(event) => commitInterval(event.currentTarget)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') commitInterval(event.currentTarget)
          }}
        />
      )}
    </div>
  )
}
