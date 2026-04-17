'use client'

import { useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ChevronDown,
  ChevronRight,
  GripVertical,
  Plus,
  Pencil,
  Trash2,
  Check,
  X,
  Loader2,
  FileText,
} from 'lucide-react'
import {
  useModules,
  useCreateModule,
  useDeleteModule,
  useReorderModules,
  useLessons,
  useCreateLesson,
  useDeleteLesson,
} from '@/lib/api/hooks'
import type { Module, Lesson } from '../../../../../../shared/lib/api/types'

interface ContentManagerProps {
  courseId: string
}

export function ContentManager({ courseId }: ContentManagerProps) {
  const { data: modules, isLoading } = useModules(courseId)
  const createModule = useCreateModule(courseId)
  const deleteModule = useDeleteModule(courseId)
  const reorderModules = useReorderModules(courseId)

  const [expandedModules, setExpandedModules] = useState<Set<string>>(new Set())
  const [newModuleName, setNewModuleName] = useState('')
  const [showNewModule, setShowNewModule] = useState(false)

  const toggleModule = useCallback((id: string) => {
    setExpandedModules((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  async function handleCreateModule() {
    if (!newModuleName.trim()) return
    await createModule.mutateAsync({ name: newModuleName.trim() })
    setNewModuleName('')
    setShowNewModule(false)
  }

  async function handleMoveModule(moduleId: string, direction: 'up' | 'down') {
    if (!modules) return
    const idx = modules.findIndex((m) => m.id === moduleId)
    if (idx < 0) return
    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    if (swapIdx < 0 || swapIdx >= modules.length) return

    const newOrder = modules.map((m, i) => {
      if (i === idx) return { moduleId: m.id, displayOrder: swapIdx }
      if (i === swapIdx) return { moduleId: m.id, displayOrder: idx }
      return { moduleId: m.id, displayOrder: i }
    })
    await reorderModules.mutateAsync(newOrder)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    )
  }

  const sortedModules = [...(modules ?? [])].sort((a, b) => a.displayOrder - b.displayOrder)

  return (
    <div className="space-y-4">
      {sortedModules.length === 0 && !showNewModule && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <FileText className="h-10 w-10 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">No modules yet. Create your first module.</p>
          </CardContent>
        </Card>
      )}

      {sortedModules.map((mod, idx) => (
        <ModuleCard
          key={mod.id}
          module={mod}
          courseId={courseId}
          isExpanded={expandedModules.has(mod.id)}
          onToggle={() => toggleModule(mod.id)}
          onDelete={() => deleteModule.mutate(mod.id)}
          onMoveUp={idx > 0 ? () => handleMoveModule(mod.id, 'up') : undefined}
          onMoveDown={
            idx < sortedModules.length - 1 ? () => handleMoveModule(mod.id, 'down') : undefined
          }
          isReordering={reorderModules.isPending}
        />
      ))}

      {showNewModule ? (
        <Card>
          <CardContent className="flex items-center gap-2 p-4">
            <Input
              value={newModuleName}
              onChange={(e) => setNewModuleName(e.target.value)}
              placeholder="Module name"
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateModule()
                if (e.key === 'Escape') setShowNewModule(false)
              }}
              autoFocus
            />
            <Button
              size="sm"
              onClick={handleCreateModule}
              disabled={createModule.isPending || !newModuleName.trim()}
            >
              {createModule.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Check className="h-4 w-4" />
              )}
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setShowNewModule(false)}>
              <X className="h-4 w-4" />
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Button variant="outline" className="w-full" onClick={() => setShowNewModule(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Add Module
        </Button>
      )}
    </div>
  )
}

interface ModuleCardProps {
  module: Module
  courseId: string
  isExpanded: boolean
  onToggle: () => void
  onDelete: () => void
  onMoveUp?: () => void
  onMoveDown?: () => void
  isReordering: boolean
}

function ModuleCard({
  module,
  courseId,
  isExpanded,
  onToggle,
  onDelete,
  onMoveUp,
  onMoveDown,
  isReordering,
}: ModuleCardProps) {
  const { data: lessons, isLoading } = useLessons(courseId, module.id)
  const createLesson = useCreateLesson(courseId, module.id)
  const deleteLesson = useDeleteLesson(courseId, module.id)

  const [showNewLesson, setShowNewLesson] = useState(false)
  const [newLessonName, setNewLessonName] = useState('')

  async function handleCreateLesson() {
    if (!newLessonName.trim()) return
    await createLesson.mutateAsync({ name: newLessonName.trim() })
    setNewLessonName('')
    setShowNewLesson(false)
  }

  const sortedLessons = [...(lessons ?? [])].sort((a, b) => a.displayOrder - b.displayOrder)

  return (
    <Card>
      <CardHeader className="p-4">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1">
            {onMoveUp && (
              <button
                onClick={onMoveUp}
                disabled={isReordering}
                className="p-1 text-slate-400 hover:text-slate-600 disabled:opacity-50"
              >
                <GripVertical className="h-4 w-4 rotate-90 scale-x-[-1]" />
              </button>
            )}
            {onMoveDown && (
              <button
                onClick={onMoveDown}
                disabled={isReordering}
                className="p-1 text-slate-400 hover:text-slate-600 disabled:opacity-50"
              >
                <GripVertical className="h-4 w-4 rotate-90" />
              </button>
            )}
          </div>
          <button onClick={onToggle} className="flex items-center gap-2 flex-1 text-left">
            {isExpanded ? (
              <ChevronDown className="h-4 w-4 text-slate-400" />
            ) : (
              <ChevronRight className="h-4 w-4 text-slate-400" />
            )}
            <span className="text-sm font-semibold text-slate-900">{module.name}</span>
            <span className="text-xs text-slate-400">
              ({sortedLessons.length} lesson{sortedLessons.length !== 1 ? 's' : ''})
            </span>
          </button>
          <Button
            size="sm"
            variant="ghost"
            onClick={onDelete}
            className="text-red-500 hover:text-red-700"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>

      {isExpanded && (
        <CardContent className="p-4 pt-0 space-y-2">
          {isLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-4 w-4 animate-spin text-slate-400" />
            </div>
          ) : sortedLessons.length === 0 && !showNewLesson ? (
            <p className="text-sm text-slate-400 py-2 pl-6">No lessons yet.</p>
          ) : (
            sortedLessons.map((lesson) => (
              <div
                key={lesson.id}
                className="flex items-center justify-between rounded-lg border p-3 pl-8"
              >
                <div>
                  <p className="text-sm font-medium text-slate-700">{lesson.name}</p>
                  {lesson.description && (
                    <p className="text-xs text-slate-400 mt-0.5">{lesson.description}</p>
                  )}
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => deleteLesson.mutate(lesson.id)}
                  className="text-red-500 hover:text-red-700"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>
            ))
          )}

          {showNewLesson ? (
            <div className="flex items-center gap-2 pl-6">
              <Input
                value={newLessonName}
                onChange={(e) => setNewLessonName(e.target.value)}
                placeholder="Lesson name"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleCreateLesson()
                  if (e.key === 'Escape') setShowNewLesson(false)
                }}
                autoFocus
                className="h-9"
              />
              <Button
                size="sm"
                onClick={handleCreateLesson}
                disabled={createLesson.isPending || !newLessonName.trim()}
              >
                {createLesson.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Check className="h-4 w-4" />
                )}
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setShowNewLesson(false)}>
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <Button
              size="sm"
              variant="ghost"
              className="ml-6 text-indigo-600 hover:text-indigo-800"
              onClick={() => setShowNewLesson(true)}
            >
              <Plus className="h-3 w-3 mr-1" />
              Add Lesson
            </Button>
          )}
        </CardContent>
      )}
    </Card>
  )
}
