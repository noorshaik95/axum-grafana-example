'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Loader2, Plus, Trash2, AlertTriangle } from 'lucide-react'
import {
  useGradingRules,
  useCreateGradingRule,
  useUpdateGradingRule,
  useDeleteGradingRule,
} from '@/lib/api/hooks'

const DEFAULT_GRADE_SCALE = [
  { letter: 'A', minScore: 90 },
  { letter: 'B', minScore: 80 },
  { letter: 'C', minScore: 70 },
  { letter: 'D', minScore: 60 },
  { letter: 'F', minScore: 0 },
]

export function GradingRulesEditor() {
  const { data: rules, isLoading } = useGradingRules()
  const createRule = useCreateGradingRule()
  const updateRule = useUpdateGradingRule()
  const deleteRule = useDeleteGradingRule()

  const [newType, setNewType] = useState('')
  const [newWeight, setNewWeight] = useState('')
  const [showAddForm, setShowAddForm] = useState(false)

  const totalWeight = (rules ?? []).reduce((sum, r) => sum + r.weight, 0)
  const isValid = Math.abs(totalWeight - 100) < 0.01

  async function handleAdd() {
    if (!newType.trim() || !newWeight) return
    await createRule.mutateAsync({
      assignmentType: newType.trim(),
      weight: parseFloat(newWeight),
    })
    setNewType('')
    setNewWeight('')
    setShowAddForm(false)
  }

  async function handleWeightChange(ruleId: string, assignmentType: string, weight: string) {
    const parsed = parseFloat(weight)
    if (isNaN(parsed)) return
    await updateRule.mutateAsync({
      id: ruleId,
      data: { assignmentType, weight: parsed },
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Assignment type weights */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Assignment Type Weights</CardTitle>
            <div className="flex items-center gap-2">
              {!isValid && rules && rules.length > 0 && (
                <Badge variant="destructive" className="flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3" />
                  Weights must sum to 100% (currently {totalWeight.toFixed(1)}%)
                </Badge>
              )}
              {isValid && rules && rules.length > 0 && (
                <Badge variant="default">100% allocated</Badge>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {(!rules || rules.length === 0) && !showAddForm ? (
            <p className="text-sm text-slate-500 text-center py-4">
              No grading rules configured. Add assignment types and their weights.
            </p>
          ) : (
            <div className="space-y-2">
              {/* Header */}
              <div className="grid grid-cols-12 gap-3 text-xs font-medium text-slate-500 px-1">
                <div className="col-span-5">Assignment Type</div>
                <div className="col-span-4">Weight (%)</div>
                <div className="col-span-3">Actions</div>
              </div>

              {(rules ?? []).map((rule) => (
                <div key={rule.id} className="grid grid-cols-12 gap-3 items-center">
                  <div className="col-span-5">
                    <span className="text-sm font-medium text-slate-900">
                      {rule.assignmentType}
                    </span>
                  </div>
                  <div className="col-span-4">
                    <Input
                      type="number"
                      defaultValue={rule.weight}
                      min="0"
                      max="100"
                      step="0.5"
                      onBlur={(e) =>
                        handleWeightChange(rule.id, rule.assignmentType, e.target.value)
                      }
                      className="h-9"
                    />
                  </div>
                  <div className="col-span-3">
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => deleteRule.mutate(rule.id)}
                      disabled={deleteRule.isPending}
                      className="text-red-500 hover:text-red-700"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {showAddForm ? (
            <div className="grid grid-cols-12 gap-3 items-end border-t pt-3">
              <div className="col-span-5">
                <Label className="text-xs">Type</Label>
                <Input
                  value={newType}
                  onChange={(e) => setNewType(e.target.value)}
                  placeholder="e.g., Homework"
                  className="h-9 mt-1"
                  autoFocus
                />
              </div>
              <div className="col-span-4">
                <Label className="text-xs">Weight (%)</Label>
                <Input
                  type="number"
                  value={newWeight}
                  onChange={(e) => setNewWeight(e.target.value)}
                  placeholder="25"
                  min="0"
                  max="100"
                  className="h-9 mt-1"
                />
              </div>
              <div className="col-span-3 flex gap-1">
                <Button
                  size="sm"
                  onClick={handleAdd}
                  disabled={createRule.isPending || !newType.trim() || !newWeight}
                >
                  {createRule.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Add'}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setShowAddForm(false)}>
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAddForm(true)}
              className="mt-2"
            >
              <Plus className="h-4 w-4 mr-1" />
              Add Type
            </Button>
          )}
        </CardContent>
      </Card>

      {/* Grade Scale */}
      <Card>
        <CardHeader>
          <CardTitle>Grade Scale</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-3 text-xs font-medium text-slate-500 px-1">
              <div>Letter Grade</div>
              <div>Minimum Score</div>
            </div>
            {DEFAULT_GRADE_SCALE.map((grade) => (
              <div key={grade.letter} className="grid grid-cols-2 gap-3 items-center">
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      grade.minScore >= 70
                        ? 'default'
                        : grade.minScore >= 60
                          ? 'secondary'
                          : 'destructive'
                    }
                  >
                    {grade.letter}
                  </Badge>
                </div>
                <span className="text-sm text-slate-600">{grade.minScore}%+</span>
              </div>
            ))}
          </div>
          <p className="text-xs text-slate-400 mt-4">
            Grade scale is currently read-only. Contact your administrator to modify.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
