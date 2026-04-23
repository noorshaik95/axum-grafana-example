'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/components/ui/card'
import { Button } from '../../../shared/components/ui/button'
import { Input } from '../../../shared/components/ui/input'
import { Label } from '../../../shared/components/ui/label'
import { Textarea } from '../../../shared/components/ui/textarea'
import { Badge } from '../../../shared/components/ui/badge'
import { Loader2, ChevronLeft, ChevronRight, FileText, CheckCircle } from 'lucide-react'
import { useGradeSubmission } from '@/lib/api/hooks'
import { formatDateTime } from '@/lib/utils'
import type { Submission } from '../../../shared/lib/api/types'

interface SubmissionGraderProps {
  assignmentId: string
  maxPoints: number
  submissions: Submission[]
}

export function SubmissionGrader({ assignmentId, maxPoints, submissions }: SubmissionGraderProps) {
  const [currentIndex, setCurrentIndex] = useState(0)
  const [score, setScore] = useState('')
  const [feedback, setFeedback] = useState('')
  const [override, setOverride] = useState(false)
  const [overrideJustification, setOverrideJustification] = useState('')
  const gradeSubmission = useGradeSubmission()

  if (submissions.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <FileText className="h-10 w-10 text-slate-300 mb-3" />
          <p className="text-slate-500">No submissions to grade.</p>
        </CardContent>
      </Card>
    )
  }

  const currentSubmission = submissions[currentIndex]
  const pendingSubmissions = submissions.filter((s) => s.status === 'submitted')
  const gradedSubmissions = submissions.filter((s) => s.status === 'graded')

  function navigateTo(index: number) {
    if (index < 0 || index >= submissions.length) return
    setCurrentIndex(index)
    setScore('')
    setFeedback('')
    setOverride(false)
    setOverrideJustification('')
  }

  async function handleGrade(e: React.FormEvent) {
    e.preventDefault()
    if (!score) return
    await gradeSubmission.mutateAsync({
      submissionId: currentSubmission.id,
      assignmentId,
      data: {
        score: parseFloat(score),
        feedback: feedback.trim() || undefined,
        override,
        overrideJustification: override ? overrideJustification : undefined,
      },
    })
    // Move to next pending submission
    const nextPendingIdx = submissions.findIndex(
      (s, i) => i > currentIndex && s.status === 'submitted'
    )
    if (nextPendingIdx >= 0) navigateTo(nextPendingIdx)
  }

  return (
    <div className="space-y-4">
      {/* Stats bar */}
      <div className="flex items-center gap-4">
        <Badge variant="outline">{pendingSubmissions.length} pending</Badge>
        <Badge variant="secondary">{gradedSubmissions.length} graded</Badge>
        <span className="text-sm text-slate-500">
          Viewing {currentIndex + 1} of {submissions.length}
        </span>
        <div className="flex items-center gap-1 ml-auto">
          <Button
            size="sm"
            variant="outline"
            disabled={currentIndex === 0}
            onClick={() => navigateTo(currentIndex - 1)}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={currentIndex === submissions.length - 1}
            onClick={() => navigateTo(currentIndex + 1)}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Split view */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Left: Submission viewer */}
        <Card className="min-h-[400px]">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Submission</CardTitle>
              <Badge variant={currentSubmission.status === 'graded' ? 'default' : 'secondary'}>
                {currentSubmission.status}
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-sm space-y-1">
              <p className="text-slate-500">
                Student ID:{' '}
                <span className="text-slate-900 font-mono">{currentSubmission.studentId}</span>
              </p>
              <p className="text-slate-500">
                Submitted:{' '}
                <span className="text-slate-900">
                  {formatDateTime(currentSubmission.submittedAt)}
                </span>
              </p>
              {currentSubmission.isLate && (
                <p className="text-amber-600">
                  Late by {currentSubmission.daysLate} day
                  {currentSubmission.daysLate !== 1 ? 's' : ''}
                </p>
              )}
            </div>

            {/* File preview area */}
            <div className="rounded-lg border bg-slate-50 flex items-center justify-center min-h-[200px]">
              {currentSubmission.filePath ? (
                currentSubmission.filePath.endsWith('.pdf') ? (
                  <iframe
                    src={currentSubmission.filePath}
                    className="w-full h-[300px] rounded-lg"
                    title="Submission PDF"
                  />
                ) : currentSubmission.filePath.match(/\.(png|jpg|jpeg|gif|webp)$/i) ? (
                  <img
                    src={currentSubmission.filePath}
                    alt="Submission"
                    className="max-h-[300px] object-contain"
                  />
                ) : (
                  <div className="text-center p-6">
                    <FileText className="h-8 w-8 text-slate-400 mx-auto mb-2" />
                    <a
                      href={currentSubmission.filePath}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sm text-indigo-600 hover:underline"
                    >
                      View submission file
                    </a>
                  </div>
                )
              ) : (
                <p className="text-sm text-slate-400">No file attached</p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Right: Grading form */}
        <Card className="min-h-[400px]">
          <CardHeader>
            <CardTitle className="text-base">Grade</CardTitle>
          </CardHeader>
          <CardContent>
            {currentSubmission.status === 'graded' ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <CheckCircle className="h-10 w-10 text-emerald-500 mb-3" />
                <p className="text-slate-700 font-medium">Already graded</p>
                <p className="text-sm text-slate-500 mt-1">
                  This submission has been graded. Use the form below to re-grade if needed.
                </p>
              </div>
            ) : null}

            <form onSubmit={handleGrade} className="space-y-4">
              <div>
                <Label htmlFor="score">Score (out of {maxPoints})</Label>
                <Input
                  id="score"
                  type="number"
                  value={score}
                  onChange={(e) => setScore(e.target.value)}
                  min="0"
                  max={maxPoints}
                  step="0.5"
                  placeholder={`0 - ${maxPoints}`}
                  className="mt-1.5"
                />
              </div>

              <div>
                <Label htmlFor="feedback">Feedback</Label>
                <Textarea
                  id="feedback"
                  value={feedback}
                  onChange={(e) => setFeedback(e.target.value)}
                  placeholder="Provide feedback for the student..."
                  rows={4}
                  className="mt-1.5"
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="override"
                    checked={override}
                    onChange={(e) => setOverride(e.target.checked)}
                    className="rounded border-slate-300"
                  />
                  <Label htmlFor="override">Grade override</Label>
                </div>
                {override && (
                  <div>
                    <Label htmlFor="justification">Override Justification</Label>
                    <Textarea
                      id="justification"
                      value={overrideJustification}
                      onChange={(e) => setOverrideJustification(e.target.value)}
                      placeholder="Reason for grade override..."
                      rows={2}
                      className="mt-1.5"
                    />
                  </div>
                )}
              </div>

              <Button
                type="submit"
                disabled={gradeSubmission.isPending || !score}
                className="w-full"
              >
                {gradeSubmission.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {gradeSubmission.isPending ? 'Submitting Grade...' : 'Submit Grade'}
              </Button>

              {gradeSubmission.error && (
                <p className="text-sm text-red-600">Failed to submit grade. Please try again.</p>
              )}
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
