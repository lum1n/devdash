/** First path token; quotes and junk after it are dropped. */
export function cleanRepoId(id: string) {
  let value = id.trim()
  try {
    value = decodeURIComponent(value)
  } catch {
    // keep raw
  }
  return value.split(/["'`/\\\s]+/).find(Boolean) ?? ''
}

/** Route param without .md so Vite does not treat the URL as a static file. */
export function noteRouteName(name: string) {
  return name.replace(/\.md$/i, '')
}

/** Current path from a git status row, including renames. */
export function statusPath(raw: string) {
  const cut = raw.lastIndexOf(' -> ')
  let value = (cut >= 0 ? raw.slice(cut + 4) : raw).trim()
  if (value.startsWith('"') && value.endsWith('"')) value = value.slice(1, -1)
  return value
}

/** API filename from a route param. */
export function noteFileName(param: string) {
  let value = param.trim()
  try {
    value = decodeURIComponent(value)
  } catch {
    // keep raw
  }
  if (!value.toLowerCase().endsWith('.md')) value += '.md'
  return value
}
