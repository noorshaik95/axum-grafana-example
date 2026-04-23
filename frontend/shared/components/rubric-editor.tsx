'use client'

import * as React from 'react'
import { cn } from '../utils/index'
import { Input } from './ui/input'
import { Button } from './ui/button'

export interface RubricRow {
  id: string
  title: string
  maxPoints: number
  sortOrder: number
}

export interface RubricEditorProps {
  rows: RubricRow[]
  onChange: (rows: RubricRow[]) => void
  editable?: boolean
  className?: string
  /** Seed used for generated row ids; replace with a deterministic id scheme in tests. */
  newIdPrefix?: string
}

let localCounter = 0
function fallbackId(prefix: string) {
  localCounter += 1
  return `${prefix}-${Date.now().toString(36)}-${localCounter}`
}

/**
 * Controlled rubric editor. Parents hold the canonical state and handle
 * persistence; this component only emits `onChange` with a sorted + renumbered
 * `sortOrder` array after every mutation.
 */
export function RubricEditor({
  rows,
  onChange,
  editable = true,
  className,
  newIdPrefix = 'rubric-row',
}: RubricEditorProps) {
  const emit = (next: RubricRow[]) => {
    const renumbered = next.map((r, i) => ({ ...r, sortOrder: i }))
    onChange(renumbered)
  }

  const addRow = () => {
    emit([
      ...rows,
      { id: fallbackId(newIdPrefix), title: '', maxPoints: 0, sortOrder: rows.length },
    ])
  }

  const removeRow = (id: string) => {
    emit(rows.filter((r) => r.id !== id))
  }

  const updateRow = (id: string, patch: Partial<RubricRow>) => {
    emit(rows.map((r) => (r.id === id ? { ...r, ...patch } : r)))
  }

  const swap = (index: number, dir: -1 | 1) => {
    const target = index + dir
    if (target < 0 || target >= rows.length) return
    const next = [...rows]
    const [moved] = next.splice(index, 1)
    next.splice(target, 0, moved)
    emit(next)
  }

  return (
    <div data-testid="rubric-editor" className={cn('flex flex-col gap-2', className)}>
      <div className="grid grid-cols-[1fr_120px_auto] items-center gap-2 text-xs font-medium uppercase tracking-wide text-warm-700">
        <span>Criterion</span>
        <span>Max points</span>
        <span className="sr-only">Actions</span>
      </div>
      {rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-warm-700">No rubric rows yet.</p>
      ) : (
        rows.map((row, index) => (
          <div
            key={row.id}
            data-testid={`rubric-row-${row.id}`}
            className="grid grid-cols-[1fr_120px_auto] items-center gap-2"
          >
            <Input
              aria-label={`Rubric row ${index + 1} title`}
              value={row.title}
              disabled={!editable}
              onChange={(e) => updateRow(row.id, { title: e.target.value })}
              placeholder={`Criterion ${index + 1}`}
            />
            <Input
              aria-label={`Rubric row ${index + 1} max points`}
              type="number"
              min={0}
              value={row.maxPoints}
              disabled={!editable}
              onChange={(e) =>
                updateRow(row.id, {
                  maxPoints: Number.isFinite(+e.target.value) ? +e.target.value : 0,
                })
              }
            />
            <div className="flex items-center gap-1">
              <Button
                type="button"
                size="sm"
                variant="ghost"
                aria-label={`Move row ${index + 1} up`}
                disabled={!editable || index === 0}
                onClick={() => swap(index, -1)}
              >
                ↑
              </Button>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                aria-label={`Move row ${index + 1} down`}
                disabled={!editable || index === rows.length - 1}
                onClick={() => swap(index, +1)}
              >
                ↓
              </Button>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                aria-label={`Remove row ${index + 1}`}
                disabled={!editable}
                onClick={() => removeRow(row.id)}
              >
                ×
              </Button>
            </div>
          </div>
        ))
      )}
      {editable ? (
        <div>
          <Button type="button" size="sm" variant="secondary" onClick={addRow}>
            + Add row
          </Button>
        </div>
      ) : null}
    </div>
  )
}

RubricEditor.displayName = 'RubricEditor'
