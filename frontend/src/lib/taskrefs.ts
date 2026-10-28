const TASK_REF = /(^|[\s(])@([A-Za-z]{3,4})-([0-9]+)\b/g

export function taskRefsToMarkdownLinks(md: string): string {
  return md.replace(TASK_REF, (_, lead: string, tag: string, num: string) => {
    const ref = `${tag.toUpperCase()}-${num}`
    return `${lead}[${ref}](taskref:${ref})`
  })
}
