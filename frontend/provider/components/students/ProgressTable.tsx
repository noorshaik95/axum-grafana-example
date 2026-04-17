'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Loader2, Search, AlertTriangle, ArrowUpDown } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

async function fetchStudentProgress(courseId: string): Promise<StudentProgress[]> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('slate_token') : null
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(`${API_URL}/api/courses/${courseId}/students/progress`, { headers })
  if (!res.ok) return []
  return res.json()
}

interface StudentProgress {
  studentId: string
  studentName: string
  email: string
  completionPercent: number
  grade: number | null
  lastActive: string | null
  status: 'on-track' | 'at-risk' | 'inactive'
}

interface ProgressTableProps {
  courseId: string
}

type SortField = 'studentName' | 'completionPercent' | 'grade' | 'lastActive' | 'status'
type SortDir = 'asc' | 'desc'

export function ProgressTable({ courseId }: ProgressTableProps) {
  const { data: students, isLoading } = useQuery({
    queryKey: ['student-progress', courseId],
    queryFn: () => fetchStudentProgress(courseId),
    enabled: !!courseId,
  })

  const [search, setSearch] = useState('')
  const [showAtRiskOnly, setShowAtRiskOnly] = useState(false)
  const [sortField, setSortField] = useState<SortField>('studentName')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  function toggleSort(field: SortField) {
    if (sortField === field) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

  const filtered = useMemo(() => {
    let result = students ?? []
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(
        (s) => s.studentName.toLowerCase().includes(q) || s.email.toLowerCase().includes(q)
      )
    }
    if (showAtRiskOnly) {
      result = result.filter((s) => s.status === 'at-risk')
    }
    result = [...result].sort((a, b) => {
      const dir = sortDir === 'asc' ? 1 : -1
      switch (sortField) {
        case 'studentName':
          return a.studentName.localeCompare(b.studentName) * dir
        case 'completionPercent':
          return (a.completionPercent - b.completionPercent) * dir
        case 'grade':
          return ((a.grade ?? 0) - (b.grade ?? 0)) * dir
        case 'lastActive':
          return (a.lastActive ?? '').localeCompare(b.lastActive ?? '') * dir
        case 'status': {
          const order = { 'at-risk': 0, inactive: 1, 'on-track': 2 }
          return (order[a.status] - order[b.status]) * dir
        }
        default:
          return 0
      }
    })
    return result
  }, [students, search, showAtRiskOnly, sortField, sortDir])

  const atRiskCount = (students ?? []).filter((s) => s.status === 'at-risk').length

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Controls */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search students..."
            className="pl-9"
          />
        </div>
        <Button
          variant={showAtRiskOnly ? 'default' : 'outline'}
          size="sm"
          onClick={() => setShowAtRiskOnly(!showAtRiskOnly)}
        >
          <AlertTriangle className="h-4 w-4 mr-1" />
          At Risk ({atRiskCount})
        </Button>
      </div>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-slate-50">
                  <SortableHeader
                    field="studentName"
                    label="Student Name"
                    current={sortField}
                    dir={sortDir}
                    onSort={toggleSort}
                  />
                  <SortableHeader
                    field="completionPercent"
                    label="Completion %"
                    current={sortField}
                    dir={sortDir}
                    onSort={toggleSort}
                  />
                  <SortableHeader
                    field="grade"
                    label="Grade"
                    current={sortField}
                    dir={sortDir}
                    onSort={toggleSort}
                  />
                  <SortableHeader
                    field="lastActive"
                    label="Last Active"
                    current={sortField}
                    dir={sortDir}
                    onSort={toggleSort}
                  />
                  <SortableHeader
                    field="status"
                    label="Status"
                    current={sortField}
                    dir={sortDir}
                    onSort={toggleSort}
                  />
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="text-center py-8 text-slate-500">
                      {search || showAtRiskOnly
                        ? 'No matching students found.'
                        : 'No student data available.'}
                    </td>
                  </tr>
                ) : (
                  filtered.map((student) => (
                    <tr
                      key={student.studentId}
                      className="border-b last:border-0 hover:bg-slate-50"
                    >
                      <td className="p-3">
                        <div>
                          <p className="font-medium text-slate-900">{student.studentName}</p>
                          <p className="text-xs text-slate-400">{student.email}</p>
                        </div>
                      </td>
                      <td className="p-3">
                        <div className="flex items-center gap-2">
                          <div className="w-24 h-2 bg-slate-100 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-indigo-500 rounded-full"
                              style={{ width: `${student.completionPercent}%` }}
                            />
                          </div>
                          <span className="text-xs text-slate-600">
                            {student.completionPercent}%
                          </span>
                        </div>
                      </td>
                      <td className="p-3">
                        {student.grade !== null ? (
                          <span
                            className={`font-medium ${
                              student.grade >= 90
                                ? 'text-emerald-600'
                                : student.grade >= 70
                                  ? 'text-amber-600'
                                  : 'text-red-600'
                            }`}
                          >
                            {student.grade.toFixed(1)}%
                          </span>
                        ) : (
                          <span className="text-slate-400">--</span>
                        )}
                      </td>
                      <td className="p-3 text-slate-500">
                        {student.lastActive
                          ? new Date(student.lastActive).toLocaleDateString()
                          : '--'}
                      </td>
                      <td className="p-3">
                        <Badge
                          variant={
                            student.status === 'at-risk'
                              ? 'destructive'
                              : student.status === 'inactive'
                                ? 'secondary'
                                : 'default'
                          }
                        >
                          {student.status}
                        </Badge>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function SortableHeader({
  field,
  label,
  current,
  dir,
  onSort,
}: {
  field: SortField
  label: string
  current: SortField
  dir: SortDir
  onSort: (field: SortField) => void
}) {
  return (
    <th className="text-left p-3 font-medium text-slate-600">
      <button
        onClick={() => onSort(field)}
        className="flex items-center gap-1 hover:text-slate-900"
      >
        {label}
        <ArrowUpDown
          className={`h-3 w-3 ${current === field ? 'text-indigo-600' : 'text-slate-300'}`}
        />
      </button>
    </th>
  )
}
