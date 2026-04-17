'use client'

import { useState } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useToast } from '@/hooks/use-toast'
import { CheckCircle2, XCircle, Loader2, Eye, Trash2, Clock } from 'lucide-react'

const MOCK_JOBS = [
  {
    id: '1',
    method: 'CSV Import',
    status: 'completed',
    totalRecords: 10000,
    processedRecords: 10000,
    failedRecords: 20,
    progress: 100,
    startedAt: '2024-01-15 10:30:00',
    completedAt: '2024-01-15 10:31:45',
    duration: '1m 45s',
  },
  {
    id: '2',
    method: 'LDAP Sync',
    status: 'processing',
    totalRecords: 5000,
    processedRecords: 3200,
    failedRecords: 0,
    progress: 64,
    startedAt: '2024-01-15 11:00:00',
    completedAt: null,
    duration: '30s',
  },
  {
    id: '3',
    method: 'Google Workspace',
    status: 'completed',
    totalRecords: 8500,
    processedRecords: 8500,
    failedRecords: 15,
    progress: 100,
    startedAt: '2024-01-15 09:15:00',
    completedAt: '2024-01-15 09:16:30',
    duration: '1m 30s',
  },
  {
    id: '4',
    method: 'API Import',
    status: 'failed',
    totalRecords: 1000,
    processedRecords: 450,
    failedRecords: 550,
    progress: 45,
    startedAt: '2024-01-15 08:00:00',
    completedAt: '2024-01-15 08:01:30',
    duration: '1m 30s',
  },
  {
    id: '5',
    method: 'CSV Import',
    status: 'pending',
    totalRecords: 12000,
    processedRecords: 0,
    failedRecords: 0,
    progress: 0,
    startedAt: null,
    completedAt: null,
    duration: '-',
  },
]

export default function JobsPage() {
  const [jobs, setJobs] = useState(MOCK_JOBS)
  const { toast } = useToast()

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'completed':
        return (
          <Badge variant="success">
            <CheckCircle2 className="mr-1 h-3 w-3" />
            Completed
          </Badge>
        )
      case 'processing':
        return (
          <Badge variant="info">
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
            Processing
          </Badge>
        )
      case 'failed':
        return (
          <Badge variant="destructive">
            <XCircle className="mr-1 h-3 w-3" />
            Failed
          </Badge>
        )
      case 'pending':
        return (
          <Badge variant="warning">
            <Clock className="mr-1 h-3 w-3" />
            Pending
          </Badge>
        )
      default:
        return <Badge>{status}</Badge>
    }
  }

  const handleCancelJob = (id: string) => {
    setJobs(
      jobs.map((job) =>
        job.id === id && job.status === 'processing' ? { ...job, status: 'failed' } : job
      )
    )
    toast({
      title: 'Job Cancelled',
      description: 'The import job has been cancelled',
    })
  }

  const handleDeleteJob = (id: string) => {
    setJobs(jobs.filter((job) => job.id !== id))
    toast({
      title: 'Job Deleted',
      description: 'The job record has been removed',
    })
  }

  const totalJobs = jobs.length
  const completedJobs = jobs.filter((j) => j.status === 'completed').length
  const processingJobs = jobs.filter((j) => j.status === 'processing').length
  const failedJobs = jobs.filter((j) => j.status === 'failed').length

  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Import Jobs</h1>
          <p className="text-muted-foreground">Monitor and manage bulk import operations</p>
        </div>

        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="pb-3">
              <CardDescription>Total Jobs</CardDescription>
              <CardTitle className="text-3xl">{totalJobs}</CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-3">
              <CardDescription>Completed</CardDescription>
              <CardTitle className="text-3xl text-green-600">{completedJobs}</CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-3">
              <CardDescription>Processing</CardDescription>
              <CardTitle className="text-3xl text-blue-600">{processingJobs}</CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-3">
              <CardDescription>Failed</CardDescription>
              <CardTitle className="text-3xl text-red-600">{failedJobs}</CardTitle>
            </CardHeader>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Recent Jobs</CardTitle>
            <CardDescription>Real-time tracking of import operations</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Method</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Progress</TableHead>
                  <TableHead>Records</TableHead>
                  <TableHead>Failed</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {jobs.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell className="font-medium">{job.method}</TableCell>
                    <TableCell>{getStatusBadge(job.status)}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Progress value={job.progress} className="w-20" />
                        <span className="text-sm text-muted-foreground">{job.progress}%</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        {job.processedRecords.toLocaleString()} /{' '}
                        {job.totalRecords.toLocaleString()}
                      </div>
                    </TableCell>
                    <TableCell>
                      {job.failedRecords > 0 ? (
                        <span className="text-red-600 font-medium">{job.failedRecords}</span>
                      ) : (
                        <span className="text-muted-foreground">0</span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{job.duration}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button variant="ghost" size="sm">
                          <Eye className="h-4 w-4" />
                        </Button>
                        {job.status === 'processing' && (
                          <Button variant="ghost" size="sm" onClick={() => handleCancelJob(job.id)}>
                            <XCircle className="h-4 w-4" />
                          </Button>
                        )}
                        {job.status !== 'processing' && (
                          <Button variant="ghost" size="sm" onClick={() => handleDeleteJob(job.id)}>
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
