'use client'

import * as React from 'react'
import { cn } from '../../utils/index'

export interface CommandPaletteResult {
  id: string
  label: string
  /** Optional short supporting line, e.g. course code. */
  description?: string
  /** Optional icon to the left of the label. */
  icon?: React.ReactNode
  /** Either `route` (navigate via router) or `onSelect` (custom handler) must resolve the result. */
  route?: string
  /** Free-form group label — "Courses", "People", "Pages". Shown as a header. */
  group?: string
}

export interface CommandPaletteProps {
  /** Async fetcher each portal wires to its own API client. Receives the raw query string. */
  onSearch: (query: string) => Promise<CommandPaletteResult[]>
  /** Fired when a user picks a result. Default behavior: if `route` is set, navigate to it. */
  onSelect?: (result: CommandPaletteResult) => void
  /** Router `push` function — typically `useRouter().push` from next/navigation. */
  routerPush?: (path: string) => void
  /** Shortcut descriptor shown in the trigger label. Default `cmd+k`. */
  shortcut?: string
  /** Controls whether the palette is rendered open (for testing / external control). */
  open?: boolean
  onOpenChange?: (open: boolean) => void
  /** Debounce window for search (ms). Default 150. */
  debounceMs?: number
  /** Placeholder for the input. */
  placeholder?: string
  className?: string
}

function useKeyboardShortcut(
  toggle: () => void,
  { disabled = false }: { disabled?: boolean } = {}
) {
  React.useEffect(() => {
    if (disabled) return
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        toggle()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [toggle, disabled])
}

export function CommandPalette({
  onSearch,
  onSelect,
  routerPush,
  shortcut = 'cmd+k',
  open: controlledOpen,
  onOpenChange,
  debounceMs = 150,
  placeholder = 'Search courses, people, pages…',
  className,
}: CommandPaletteProps) {
  const isControlled = typeof controlledOpen === 'boolean'
  const [uncontrolledOpen, setUncontrolledOpen] = React.useState(false)
  const open = isControlled ? controlledOpen : uncontrolledOpen
  const setOpen = React.useCallback(
    (next: boolean) => {
      if (!isControlled) setUncontrolledOpen(next)
      onOpenChange?.(next)
    },
    [isControlled, onOpenChange]
  )

  const [query, setQuery] = React.useState('')
  const [results, setResults] = React.useState<CommandPaletteResult[]>([])
  const [highlight, setHighlight] = React.useState(0)
  const [loading, setLoading] = React.useState(false)
  const inputRef = React.useRef<HTMLInputElement | null>(null)
  const debounceTimer = React.useRef<ReturnType<typeof setTimeout> | null>(null)
  const requestSeqRef = React.useRef(0)

  useKeyboardShortcut(() => setOpen(!open))

  React.useEffect(() => {
    if (open) {
      setHighlight(0)
      // Defer focus until after the portal element is in the DOM.
      const t = setTimeout(() => inputRef.current?.focus(), 0)
      return () => clearTimeout(t)
    }
    return
  }, [open])

  React.useEffect(() => {
    if (!open) return
    if (debounceTimer.current) clearTimeout(debounceTimer.current)
    const q = query
    debounceTimer.current = setTimeout(async () => {
      const seq = ++requestSeqRef.current
      setLoading(true)
      try {
        const r = await onSearch(q)
        if (seq === requestSeqRef.current) {
          setResults(r)
          setHighlight(0)
        }
      } finally {
        if (seq === requestSeqRef.current) setLoading(false)
      }
    }, debounceMs)
    return () => {
      if (debounceTimer.current) clearTimeout(debounceTimer.current)
    }
  }, [query, open, debounceMs, onSearch])

  const select = React.useCallback(
    (r: CommandPaletteResult) => {
      setOpen(false)
      setQuery('')
      if (onSelect) {
        onSelect(r)
        return
      }
      if (r.route && routerPush) routerPush(r.route)
    },
    [onSelect, routerPush, setOpen]
  )

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      setOpen(false)
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHighlight((i) => Math.min(i + 1, Math.max(results.length - 1, 0)))
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlight((i) => Math.max(i - 1, 0))
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      const r = results[highlight]
      if (r) select(r)
    }
  }

  if (!open) return null

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      data-testid="command-palette"
      className={cn(
        'fixed inset-0 z-[70] flex items-start justify-center bg-warm-900/40 px-4 pt-24 backdrop-blur-sm',
        className
      )}
      onClick={(e) => {
        if (e.target === e.currentTarget) setOpen(false)
      }}
    >
      <div className="w-full max-w-xl overflow-hidden rounded-card border border-warm-200 bg-paper shadow-lg">
        <div className="flex items-center gap-2 border-b border-warm-200 bg-paper px-4 py-3">
          <svg
            className="h-4 w-4 text-warm-700"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            ref={inputRef}
            data-testid="command-palette-input"
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder={placeholder}
            aria-label="Search"
            className="flex-1 bg-transparent text-sm text-warm-900 outline-none placeholder:text-warm-700 focus-visible:outline-none"
          />
          <span className="hidden rounded border border-warm-200 px-1.5 py-0.5 font-mono text-[10px] uppercase text-warm-700 sm:inline">
            {shortcut === 'cmd+k' ? '⌘K' : shortcut}
          </span>
        </div>
        <ul
          role="listbox"
          className="max-h-80 overflow-y-auto py-1"
          aria-busy={loading || undefined}
        >
          {results.length === 0 ? (
            <li className="px-4 py-8 text-center text-sm text-warm-700">
              {loading ? 'Searching…' : query ? 'No results' : 'Type to search'}
            </li>
          ) : (
            results.map((r, i) => (
              <li
                key={r.id}
                role="option"
                aria-selected={i === highlight}
                data-testid={`command-palette-result-${r.id}`}
                data-highlighted={i === highlight ? 'true' : undefined}
                className={cn(
                  'flex cursor-pointer items-center gap-3 px-4 py-2 text-sm',
                  i === highlight ? 'bg-forest-100 text-forest-900' : 'text-warm-900'
                )}
                onMouseEnter={() => setHighlight(i)}
                onClick={() => select(r)}
              >
                {r.icon ? <span className="shrink-0 text-forest-600">{r.icon}</span> : null}
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-medium">{r.label}</span>
                  {r.description ? (
                    <span className="truncate text-xs text-warm-700">{r.description}</span>
                  ) : null}
                </span>
                {r.group ? (
                  <span className="shrink-0 rounded-full bg-warm-100 px-2 py-0.5 text-[10px] uppercase tracking-wide text-warm-700">
                    {r.group}
                  </span>
                ) : null}
              </li>
            ))
          )}
        </ul>
      </div>
    </div>
  )
}

CommandPalette.displayName = 'CommandPalette'
