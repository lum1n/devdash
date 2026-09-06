export function formatAge(iso: string) {
  if (!iso || iso.startsWith('0001')) return '—'
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return '—'
  const mins = Math.round((Date.now() - t) / 60000)
  if (mins < 60) return `${mins}m`
  const hours = Math.round(mins / 60)
  if (hours < 48) return `${hours}h`
  return `${Math.round(hours / 24)}d`
}

export function toneClass(tone: string) {
  if (tone === 'ok') return 'accent'
  if (tone === 'warn') return 'warn'
  if (tone === 'danger') return 'danger'
  return 'muted'
}

export function formatUSD(n: number) {
  if (!n) return '$0'
  if (n < 0.01) return `$${n.toFixed(4)}`
  return `$${n.toFixed(2)}`
}

export function sparkline(values: number[]) {
  const blocks = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█']
  const max = Math.max(0, ...values)
  if (!values.length || max <= 0) return '▁'.repeat(Math.min(values.length, 16) || 1)
  return values
    .map((v) => blocks[Math.min(blocks.length - 1, Math.round((v / max) * (blocks.length - 1)))])
    .join('')
}
