export type Repo = {
  id: string
  name: string
  path: string
  root: string
  branch: string
  dirty: boolean
  changed: number
  stash: number
  ahead: number
  behind: number
  last_commit: string
  last_message: string
  remote?: string
  stack: string
  attention: boolean
}

export type Annotation = {
  plugin: string
  kind: string
  label: string
  detail?: string
  tone?: 'ok' | 'warn' | 'danger'
  repo_id?: string
}

export type Widget = {
  id: string
  title: string
  summary: string
  kind?: string
  data?: AccOverviewData | AccProjectData | TmuxOverviewData | TmuxProjectData | PortsOverviewData | PortsProjectData | GhOverviewData | GhProjectData
}

export type AccFleet = {
  connected: boolean
  skillcp_ok: boolean
  agents: number
  attention: number
  error?: string
}

export type AccPlan = {
  id: string
  name: string
  used_pct: number
  label?: string
  health: string
  detail?: string
}

export type AccCost = {
  today: number
  week: number
  month: number
  spark: number[]
}

export type AccOverviewData = {
  fleet: AccFleet
  plans: AccPlan[]
  cost: AccCost
}

export type AccAgent = {
  session: string
  kind: string
  state: string
  path?: string
}

export type AccHarness = {
  id: string
  label: string
  ready: boolean
}

export type AccSession = {
  source: string
  id: string
  title?: string
  updated_at?: string
}

export type AccProjectData = {
  agents: AccAgent[]
  cost_30d: number
  harnesses: AccHarness[]
  sessions: AccSession[]
}

export type TmuxWindow = {
  index: number
  name: string
  path?: string
  command?: string
}

export type TmuxSession = {
  name: string
  windows: TmuxWindow[]
  attached?: boolean
}

export type TmuxOverviewData = {
  ready: boolean
  sessions: number
  error?: string
}

export type TmuxProjectData = {
  ready: boolean
  sessions: TmuxSession[]
}

export type PortListener = {
  pid: number
  port: number
  addr: string
  cwd: string
  command: string
  label: string
  url: string
}

export type PortsOverviewData = {
  ready: boolean
  listening: number
  error?: string
}

export type PortsProjectData = {
  ready: boolean
  listeners: PortListener[]
}

export type GhPull = {
  number: number
  title: string
  url: string
  draft: boolean
  review: boolean
  checks?: string
  repo?: string
}

export type GhOverviewData = {
  ready: boolean
  open: number
  review: number
  error?: string
}

export type GhProjectData = {
  ready: boolean
  slug?: string
  pulls: GhPull[]
}

export type Command = {
  id: string
  title: string
  keys?: string
}

export type Counts = {
  repos: number
  dirty: number
  behind: number
  ahead: number
  attention: number
  today: number
  archived: number
}

export type Reason = {
  id: string
  label: string
  detail?: string
  tone: string
}

export type TodayItem = {
  repo: Repo
  score: number
  why: Reason[]
  next?: string
  pinned: boolean
  snoozed: boolean
  opened_at?: string
}

export type NoteMeta = {
  name: string
  title: string
  updated_at: string
}

export type Note = NoteMeta & {
  content: string
}

export type PaletteItem = {
  id: string
  kind: 'jump' | 'action' | 'plugin' | string
  title: string
  subtitle?: string
  keys?: string
  repo_id?: string
  action?: string
  plugin?: string
}

export type FocusView = {
  pinned: string[]
  archived: string[]
  snoozed: string[]
}

export type Signal = {
  id: string
  label: string
  detail?: string
  tone: 'ok' | 'warn' | 'danger' | 'muted' | string
}

export type WeekBucket = {
  start: string
  count: number
}

export type Commit = {
  hash: string
  when: string
  subject: string
  author: string
}

export type FileChange = {
  path: string
  status: string
}

export type Author = {
  name: string
  count: number
}

export type Shortcut = {
  id: string
  key: string
  label: string
  enabled: boolean
}

export type ProjectDetail = {
  repo: Repo
  summary: string
  signals: Signal[]
  activity: WeekBucket[]
  weekdays: number[]
  commits: Commit[]
  files: FileChange[]
  authors: Author[]
  shortcuts: Shortcut[]
  remote_url?: string
  commit_n_16w: number
  plugins?: Widget[]
  next?: string
  notes?: NoteMeta[]
  archived?: boolean
}

export type RepoDiff = {
  ref: string
  title: string
  subject?: string
  path?: string
  patch: string
  files: string[]
  empty: boolean
  truncated?: boolean
}

export type OpenResult = {
  ok: boolean
  action: string
  detail?: string
}

export type Workspace = {
  id: string
  name: string
  kind: 'local' | 'ssh' | string
  roots?: string[]
  host?: string
  url?: string
  active: boolean
  ready: boolean
}

export type Overview = {
  workspace: Workspace
  workspaces: Workspace[]
  roots: string[]
  repos: Repo[]
  counts: Counts
  today: TodayItem[]
  palette: PaletteItem[]
  focus: FocusView
  widgets: Widget[] | null
  commands: Command[] | null
  annotations: Annotation[] | null
  scanned_at: string
}
