'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { Upload, X, FileText, Clock, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import { useAssignmentSubmissions, useSubmitAssignment, useGradedResult } from '@/lib/api/hooks';
import { formatFileSize, formatDateTime } from '@/lib/utils';
import type { StudentAssignment, SubmissionRecord } from '@/lib/api/assignments';

interface SubmissionUploaderProps {
  assignment: StudentAssignment;
}

function DeadlineCountdown({ dueDate }: { dueDate: string }) {
  const [remaining, setRemaining] = useState('');
  const [isOverdue, setIsOverdue] = useState(false);

  useEffect(() => {
    function update() {
      const now = Date.now();
      const due = new Date(dueDate).getTime();
      const diff = due - now;

      if (diff <= 0) {
        setIsOverdue(true);
        setRemaining('Overdue');
        return;
      }

      const days = Math.floor(diff / (1000 * 60 * 60 * 24));
      const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

      if (days > 0) {
        setRemaining(`${days}d ${hours}h remaining`);
      } else if (hours > 0) {
        setRemaining(`${hours}h ${minutes}m remaining`);
      } else {
        setRemaining(`${minutes}m remaining`);
      }
      setIsOverdue(false);
    }
    update();
    const interval = setInterval(update, 60000);
    return () => clearInterval(interval);
  }, [dueDate]);

  return (
    <div
      className={`flex items-center gap-1.5 text-sm font-medium ${
        isOverdue ? 'text-red-600' : 'text-[var(--color-text-muted)]'
      }`}
    >
      <Clock className="h-4 w-4" />
      <span>{remaining}</span>
    </div>
  );
}

function GradeDisplay({
  assignmentId,
  submission,
}: {
  assignmentId: string;
  submission: SubmissionRecord;
}) {
  const { data: graded } = useGradedResult(assignmentId, submission.id);

  if (!graded) return null;

  return (
    <div className="rounded-xl border border-green-200 bg-green-50 p-5 space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-green-800">Grade Result</h4>
        <span className="text-xl font-bold text-green-700">
          {graded.score}/{graded.maxPoints}
        </span>
      </div>

      {graded.adjustedScore !== graded.score && (
        <p className="text-xs text-green-700">
          Adjusted score: {graded.adjustedScore} (late penalty applied)
        </p>
      )}

      {graded.feedback && (
        <div>
          <p className="text-xs font-medium text-green-800 mb-1">Feedback</p>
          <p className="text-sm text-green-700 whitespace-pre-wrap">{graded.feedback}</p>
        </div>
      )}

      {graded.rubricBreakdown && graded.rubricBreakdown.length > 0 && (
        <div>
          <p className="text-xs font-medium text-green-800 mb-2">Rubric Breakdown</p>
          <div className="space-y-1.5">
            {graded.rubricBreakdown.map((item, i) => (
              <div key={i} className="flex items-center justify-between text-xs">
                <span className="text-green-700">{item.criterion}</span>
                <span className="font-medium text-green-800">
                  {item.earnedPoints}/{item.maxPoints}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export function SubmissionUploader({ assignment }: SubmissionUploaderProps) {
  const [files, setFiles] = useState<File[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: submissions } = useAssignmentSubmissions(assignment.id);
  const submitMutation = useSubmitAssignment();

  const submissionList = Array.isArray(submissions) ? submissions : [];
  const latestSubmission = submissionList[0] as SubmissionRecord | undefined;

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    const droppedFiles = Array.from(e.dataTransfer.files);
    setFiles((prev) => [...prev, ...droppedFiles]);
  }, []);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setFiles((prev) => [...prev, ...Array.from(e.target.files!)]);
    }
  }, []);

  const removeFile = (idx: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleSubmit = () => {
    if (files.length === 0) return;
    submitMutation.mutate(
      { assignmentId: assignment.id, files },
      { onSuccess: () => setFiles([]) }
    );
  };

  const statusBadge = {
    not_started: { label: 'Not Started', class: 'bg-gray-100 text-gray-600' },
    in_progress: { label: 'In Progress', class: 'bg-amber-50 text-amber-700' },
    submitted: { label: 'Submitted', class: 'bg-blue-50 text-blue-700' },
    graded: { label: 'Graded', class: 'bg-green-50 text-green-700' },
  }[assignment.status] ?? { label: assignment.status, class: 'bg-gray-100 text-gray-600' };

  return (
    <div className="space-y-5">
      {/* Status + Deadline */}
      <div className="flex flex-wrap items-center gap-3">
        <span className={`rounded-full px-3 py-1 text-xs font-medium ${statusBadge.class}`}>
          {statusBadge.label}
        </span>
        <DeadlineCountdown dueDate={assignment.dueDate} />
      </div>

      {/* Graded result */}
      {assignment.status === 'graded' && latestSubmission && (
        <GradeDisplay assignmentId={assignment.id} submission={latestSubmission} />
      )}

      {/* File dropzone */}
      {assignment.status !== 'graded' && (
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click();
          }}
          aria-label="Upload files by dropping or clicking"
          className={`cursor-pointer rounded-xl border-2 border-dashed p-8 text-center transition-colors ${
            isDragging
              ? 'border-indigo-400 bg-indigo-50'
              : 'border-[var(--color-border)] bg-[var(--color-bg-muted)] hover:border-indigo-300'
          }`}
        >
          <Upload className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
          <p className="mt-2 text-sm font-medium text-[var(--color-text)]">
            Drop files here or click to browse
          </p>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            PDF, DOC, ZIP, or any file type
          </p>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            onChange={handleFileSelect}
            className="hidden"
            aria-hidden="true"
          />
        </div>
      )}

      {/* Selected files */}
      {files.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
            Selected Files
          </p>
          {files.map((file, idx) => (
            <div
              key={`${file.name}-${idx}`}
              className="flex items-center justify-between rounded-lg border border-[var(--color-border)] bg-white p-3"
            >
              <div className="flex items-center gap-2 min-w-0">
                <FileText className="h-4 w-4 text-[var(--color-text-muted)] shrink-0" />
                <span className="text-sm text-[var(--color-text)] truncate">{file.name}</span>
                <span className="text-xs text-[var(--color-text-muted)] shrink-0">
                  ({formatFileSize(file.size)})
                </span>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  removeFile(idx);
                }}
                className="ml-2 rounded p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-error)]"
                aria-label={`Remove ${file.name}`}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))}
          <button
            onClick={handleSubmit}
            disabled={submitMutation.isPending}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors flex items-center justify-center gap-2"
          >
            {submitMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Submitting...
              </>
            ) : (
              'Submit'
            )}
          </button>
        </div>
      )}

      {submitMutation.isError && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 p-3 text-sm text-red-600">
          <AlertCircle className="h-4 w-4 shrink-0" />
          Submission failed. Please try again.
        </div>
      )}

      {/* Submission history */}
      {submissionList.length > 0 && (
        <div>
          <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-2">
            Submission History
          </p>
          <div className="space-y-2">
            {submissionList.map((sub) => (
              <div
                key={sub.id}
                className="flex items-center justify-between rounded-lg border border-[var(--color-border)] bg-white p-3"
              >
                <div className="flex items-center gap-2 min-w-0">
                  <CheckCircle2 className="h-4 w-4 text-green-600 shrink-0" />
                  <span className="text-sm text-[var(--color-text)] truncate">
                    {sub.fileName || sub.filePath}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
                  {sub.isLate && (
                    <span className="text-red-500 font-medium">Late ({sub.daysLate}d)</span>
                  )}
                  <span>{formatDateTime(sub.submittedAt)}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
