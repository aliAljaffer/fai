import { useMemo, useState, type KeyboardEvent } from 'react'
import { useDebounced } from '../../lib/useDebounced'
import { useResolveRefs, useTags } from '../../api/hooks'
import { useWorkspaceStore } from '../../store/workspace'

export type RefField = HTMLInputElement | HTMLTextAreaElement

export interface RefToken {
  kind: '@' | '#'
  start: number
  end: number
  query: string
}

export interface RefOption {
  key: string
  label: string
  detail?: string
  group?: string
  value: string
}

export function findToken(value: string, caret: number): RefToken | null {
  const before = value.slice(0, caret)
  const lineStart = before.lastIndexOf('\n') + 1
  const line = before.slice(lineStart)
  const at = line.lastIndexOf('@')
  const hash = line.lastIndexOf('#')
  const index = Math.max(at, hash)
  if (index < 0) return null
  const kind = index === at ? '@' : '#'
  const prev = line[index - 1]
  if (prev !== undefined && !/[\s(]/.test(prev)) return null
  const query = line.slice(index + 1)
  if (kind === '#' && query.startsWith(' ')) return null
  return { kind, start: lineStart + index, end: caret, query }
}

export function labelFromQuery(query: string): string {
  return query.trim().replace(/\s+/g, '-').replace(/-+/g, '-')
}

export function useRefPicker({
  value,
  onChange,
}: {
  value: string
  onChange: (next: string) => void
}) {
  const [token, setToken] = useState<RefToken | null>(null)
  const [field, setField] = useState<RefField | null>(null)
  const [active, setActive] = useState(0)
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const debouncedQuery = useDebounced(token?.kind === '@' ? token.query : '', 150)
  const { data: refs } = useResolveRefs(debouncedQuery, workspaceId)
  const { data: tags } = useTags()

  const options = useMemo<RefOption[]>(() => {
    if (!token) return []
    if (token.kind === '@') {
      return (refs ?? []).map((item) => ({
        key: `task-${item.id}`,
        label: item.ref,
        detail: item.title,
        group: item.workspace_name,
        value: `@${item.ref}`,
      }))
    }
    const wanted = labelFromQuery(token.query).toLowerCase()
    const matches = (tags ?? [])
      .filter((tag) => !wanted || tag.name.toLowerCase().includes(wanted))
      .slice(0, 8)
    const labelOptions: RefOption[] = matches.map((tag) => ({
      key: `tag-${tag.id}`,
      label: tag.name,
      value: `#${tag.name}`,
    }))
    if (wanted && !matches.some((tag) => tag.name.toLowerCase() === wanted)) {
      labelOptions.push({
        key: 'create-label',
        label: `Create label ${wanted}`,
        value: `#${wanted}`,
      })
    }
    return labelOptions
  }, [token, refs, tags])

  function handleChange(next: string, element: RefField | null) {
    const caret = element?.selectionStart ?? next.length
    if (token?.kind === '#' && caret > token.start && next[caret - 1] === ' ') {
      const dashed = next.slice(0, caret - 1) + '-' + next.slice(caret)
      onChange(dashed)
      if (element) {
        setField(element)
        requestAnimationFrame(() => element.setSelectionRange(caret, caret))
      }
      setToken(findToken(dashed, caret))
      setActive(0)
      return
    }
    onChange(next)
    if (element) setField(element)
    setToken(findToken(next, caret))
    setActive(0)
  }

  function sync(event: { currentTarget: RefField }) {
    setField(event.currentTarget)
    setToken(findToken(value, event.currentTarget.selectionStart ?? value.length))
    setActive(0)
  }

  function dismiss() {
    setToken(null)
  }

  function handleBlur() {
    setTimeout(() => setToken(null), 120)
  }

  function select(option: RefOption) {
    if (!token) return
    const suffix = value.slice(token.end)
    const pad = /^\s/.test(suffix) ? '' : ' '
    const next = value.slice(0, token.start) + option.value + pad + suffix
    onChange(next)
    setToken(null)
    const caret = token.start + option.value.length + pad.length
    const element = field
    if (element) {
      requestAnimationFrame(() => {
        element.focus()
        element.setSelectionRange(caret, caret)
      })
    }
  }

  function handleKeyDown(event: KeyboardEvent<RefField>): boolean {
    if (!token || event.nativeEvent.isComposing) return false
    if (event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      setToken(null)
      return true
    }
    if (!options.length) return false
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setActive((current) => (current + 1) % options.length)
      return true
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setActive((current) => (current - 1 + options.length) % options.length)
      return true
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      select(options[Math.min(active, options.length - 1)])
      return true
    }
    return false
  }

  return {
    token,
    field,
    options,
    active: options.length ? Math.min(active, options.length - 1) : 0,
    handleChange,
    handleKeyDown,
    handleBlur,
    sync,
    select,
    dismiss,
  }
}

export type RefPickerState = ReturnType<typeof useRefPicker>
