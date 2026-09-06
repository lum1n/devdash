import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { PageTrail, useRepoName } from '#/components/trail'
import { formatAge } from '#/lib/format'
import { noteRouteName } from '#/lib/repo-id'
import { createNote, deleteNote, saveNote } from '#/lib/server-functions'
import type { Note, NoteMeta } from '#/lib/types'

const tools: { id: string; label: string; title: string }[] = [
  { id: 'h1', label: 'h1', title: 'heading 1' },
  { id: 'h2', label: 'h2', title: 'heading 2' },
  { id: 'h3', label: 'h3', title: 'heading 3' },
  { id: 'bold', label: 'b', title: 'bold' },
  { id: 'italic', label: 'i', title: 'italic' },
  { id: 'code', label: '`', title: 'inline code' },
  { id: 'ul', label: 'ul', title: 'bullet list' },
  { id: 'ol', label: 'ol', title: 'numbered list' },
  { id: 'check', label: '[ ]', title: 'checkbox' },
  { id: 'quote', label: '>', title: 'quote' },
  { id: 'link', label: '[]', title: 'link' },
]

export function ProjectNotes({ repoId, notes }: { repoId: string; notes: NoteMeta[] }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [flash, setFlash] = useState('')

  const create = useMutation({
    mutationFn: () => createNote({ data: { id: repoId, title: 'note' } }),
    onSuccess: (note) => {
      queryClient.invalidateQueries({ queryKey: ['project', repoId] })
      void navigate({ to: '/repos/$repoId/notes/$noteName', params: { repoId, noteName: noteRouteName(note.name) } })
    },
    onError: (err) => setFlash(err.message),
  })

  return (
    <section className="section">
      <p className="section-title">
        notes
        <span className="muted"> {notes.length} in notes/</span>
        {flash ? <span className="accent"> · {flash}</span> : null}
      </p>
      <div className="now-row note-cards" role="group" aria-label="notes">
        {notes.map((n) => (
          <Link
            key={n.name}
            className="now-metric hit"
            to="/repos/$repoId/notes/$noteName"
            params={{ repoId, noteName: noteRouteName(n.name) }}
          >
            <span className="now-label">{formatAge(n.updated_at)}</span>
            <span className="now-value">{n.title || n.name}</span>
          </Link>
        ))}
        <button className="now-metric hit" type="button" disabled={create.isPending} onClick={() => create.mutate()}>
          <span className="now-label">{create.isPending ? 'creating' : 'new'}</span>
          <span className="now-value">+</span>
        </button>
      </div>
    </section>
  )
}

export function NoteEditor({ repoId, note }: { repoId: string; note: Note }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const repoName = useRepoName(repoId)
  const [draft, setDraft] = useState(note.content)
  const [dirty, setDirty] = useState(false)
  const [flash, setFlash] = useState('')
  const ta = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    setDraft(note.content)
    setDirty(false)
  }, [note.name, note.content])

  const save = useMutation({
    mutationFn: () => saveNote({ data: { id: repoId, name: note.name, content: draft } }),
    onSuccess: (saved) => {
      setDraft(saved.content)
      setDirty(false)
      setFlash('saved')
      queryClient.setQueryData(['note', repoId, saved.name], saved)
      queryClient.invalidateQueries({ queryKey: ['project', repoId] })
    },
    onError: (err) => setFlash(err.message),
  })

  const remove = useMutation({
    mutationFn: () => deleteNote({ data: { id: repoId, name: note.name } }),
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: ['note', repoId, note.name] })
      queryClient.invalidateQueries({ queryKey: ['project', repoId] })
      void navigate({ to: '/repos/$repoId', params: { repoId } })
    },
    onError: (err) => setFlash(err.message),
  })

  function apply(id: string) {
    const el = ta.current
    if (!el) return
    setDraft(formatMarkdown(el, draft, id))
    setDirty(true)
  }

  return (
    <>
      <header className="topbar">
        <PageTrail
          crumbs={[
            { label: 'overview', to: '/' },
            { label: repoName, to: '/repos/$repoId', repoId },
            { label: note.title || note.name },
          ]}
        />
        <div className="flex flex-wrap items-center gap-2">
          {flash ? <span className="accent">{flash}</span> : null}
          {dirty ? <span className="warn">unsaved</span> : null}
          <Link className="btn" to="/repos/$repoId" params={{ repoId }}>
            back
          </Link>
        </div>
      </header>
      <div className="content note-screen">
        <div className="note-toolbar" role="toolbar" aria-label="markdown">
          {tools.map((t) => (
            <button key={t.id} className="tab" type="button" title={t.title} onClick={() => apply(t.id)}>
              {t.label}
            </button>
          ))}
          <span className="note-toolbar-gap" />
          <button className="btn" type="button" disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
            save
          </button>
          <button
            className="tab"
            type="button"
            disabled={remove.isPending}
            onClick={() => {
              if (window.confirm(`delete ${note.name}?`)) remove.mutate()
            }}
          >
            delete
          </button>
        </div>
        <textarea
          ref={ta}
          className="note-editor"
          aria-label="note markdown"
          spellCheck={false}
          value={draft}
          onChange={(e) => {
            setDraft(e.target.value)
            setDirty(true)
          }}
          onKeyDown={(e) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 's') {
              e.preventDefault()
              if (dirty) save.mutate()
              return
            }
            if ((e.metaKey || e.ctrlKey) && e.key === 'b') {
              e.preventDefault()
              apply('bold')
              return
            }
            if ((e.metaKey || e.ctrlKey) && e.key === 'i') {
              e.preventDefault()
              apply('italic')
            }
          }}
        />
        <p className="muted">{note.name} · ctrl/cmd-s save · b bold · i italic</p>
      </div>
    </>
  )
}

function formatMarkdown(el: HTMLTextAreaElement, value: string, id: string): string {
  const start = el.selectionStart
  const end = el.selectionEnd
  if (id === 'h1' || id === 'h2' || id === 'h3') {
    const marks: Record<string, string> = { h1: '# ', h2: '## ', h3: '### ' }
    return prefixLines(el, value, () => marks[id], true)
  }
  if (id === 'ul') return prefixLines(el, value, () => '- ')
  if (id === 'ol') return prefixLines(el, value, (i) => `${i + 1}. `)
  if (id === 'check') return prefixLines(el, value, () => '- [ ] ')
  if (id === 'quote') return prefixLines(el, value, () => '> ')
  if (id === 'bold') return wrap(el, value, '**', '**', 'bold')
  if (id === 'italic') return wrap(el, value, '_', '_', 'italic')
  if (id === 'code') return wrap(el, value, '`', '`', 'code')
  if (id === 'link') {
    const selected = value.slice(start, end) || 'text'
    return wrap(el, value, '[', '](url)', selected)
  }
  return value
}

function wrap(el: HTMLTextAreaElement, value: string, before: string, after: string, fallback: string): string {
  const start = el.selectionStart
  const end = el.selectionEnd
  const selected = value.slice(start, end) || fallback
  const next = value.slice(0, start) + before + selected + after + value.slice(end)
  const from = start + before.length
  queueSelection(el, from, from + selected.length)
  return next
}

function prefixLines(
  el: HTMLTextAreaElement,
  value: string,
  prefix: (i: number) => string,
  replaceHeading = false,
): string {
  const start = el.selectionStart
  const end = el.selectionEnd
  const lineStart = value.lastIndexOf('\n', start - 1) + 1
  const after = value.indexOf('\n', end)
  const lineEnd = after === -1 ? value.length : after
  const block = value.slice(lineStart, lineEnd) || ''
  const lines = block.split('\n')
  const nextBlock = lines
    .map((line, i) => {
      let body = line
      if (replaceHeading) body = body.replace(/^#{1,6}\s+/, '')
      const p = prefix(i)
      if (body.startsWith(p)) return body
      return p + body
    })
    .join('\n')
  const next = value.slice(0, lineStart) + nextBlock + value.slice(lineEnd)
  queueSelection(el, lineStart, lineStart + nextBlock.length)
  return next
}

function queueSelection(el: HTMLTextAreaElement, start: number, end: number) {
  requestAnimationFrame(() => {
    el.focus()
    el.setSelectionRange(start, end)
  })
}
