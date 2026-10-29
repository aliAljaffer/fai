const TASK_REF = /(^|[\s(])@([A-Za-z]{3,4})-([0-9]+)\b/g

export function taskRefsToMarkdownLinks(md: string): string {
  return md.replace(TASK_REF, (_, lead: string, tag: string, num: string) => {
    const ref = `${tag.toUpperCase()}-${num}`
    return `${lead}[${ref}](taskref:${ref})`
  })
}

export type TaskRefPart = { text: string; ref?: string }

export function splitTaskRefs(text: string): TaskRefPart[] {
  const parts: TaskRefPart[] = []
  let last = 0
  for (const match of text.matchAll(TASK_REF)) {
    const index = match.index ?? 0
    const lead = match[1]
    const start = index + lead.length
    if (start > last) parts.push({ text: text.slice(last, start) })
    parts.push({ text: '', ref: `${match[2].toUpperCase()}-${match[3]}` })
    last = index + match[0].length
  }
  if (last < text.length) parts.push({ text: text.slice(last) })
  return parts
}
