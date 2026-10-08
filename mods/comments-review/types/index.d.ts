export type ReviewReply = {
  id: string
  author: string
  text: string
}

export type ReviewThread = {
  id: string
  author: string
  line: number
  text: string
  type: string
  blocking: boolean
  resolved: boolean
  isSuggestion: boolean
  startLine: number
  endLine: number
  originalText: string
  proposedText: string
  accepted: boolean | null
  sectionPath: string
  orphaned: boolean
  replies: ReviewReply[]
}

export type ReviewGate = {
  decision: string
  blocking: number
  nonBlocking: number
  pendingSuggestions: number
}

export type ReviewRecord = {
  author: string
  decision: string
  note: string
}

export type ReviewSnapshot = {
  docId: string
  name: string
  author: string
  revision: string
  lines: string[]
  threads: ReviewThread[]
  gate: ReviewGate
  reviews: ReviewRecord[]
}

export type ReviewMode = 'browse' | 'comment' | 'blocking' | 'reply' | 'verdict' | 'note'

export type ReviewStatus = 'idle' | 'starting' | 'ready' | 'error'

export type ReviewView = {
  doc: string
  status: ReviewStatus
  cursor: number
  top: number
  threadIndex: number
  mode: ReviewMode
  note: string
  message: string
  showResolved: boolean
  zoom: boolean
}

declare module 'claude-code' {
  interface PluginState {
    'comments-review': {
      view: ReviewView
      snapshot: ReviewSnapshot | null
      activePlan: string | null
      unlockedPlan: string | null
      reminded: boolean
      livingDoc: string | null
      drift: number
    }
  }
}
