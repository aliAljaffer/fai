const WIKILINK = /\[\[([^\]]+)\]\]/g

export function wikilinksToMarkdownLinks(md: string): string {
  return md.replace(WIKILINK, (_, title: string) => `[${title}](wikilink:${encodeURIComponent(title)})`)
}
