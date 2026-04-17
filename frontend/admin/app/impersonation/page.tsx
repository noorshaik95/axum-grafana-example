'use client'

import { useState } from 'react'
import { useImpersonationSessions, useStartImpersonation } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatDate } from '@/lib/utils'
import { UserCog, Search, X, AlertTriangle, Loader2, LogIn, History } from 'lucide-react'

function ImpersonationBanner({
  user,
  onEnd,
}: {
  user: { name: string; tenantName: string }
  onEnd: () => void
}) {
  return (
    <div className="fixed top-0 left-0 right-0 z-50 flex items-center justify-between bg-yellow-400 px-6 py-2 shadow-md">
      <div className="flex items-center gap-2 text-sm font-medium text-yellow-900">
        <AlertTriangle className="h-4 w-4" />
        Impersonating <strong>{user.name}</strong> at <strong>{user.tenantName}</strong>
      </div>
      <Button
        size="sm"
        variant="outline"
        className="border-yellow-800 text-yellow-900 hover:bg-yellow-500"
        onClick={onEnd}
      >
        End Session
      </Button>
    </div>
  )
}

export default function ImpersonationPage() {
  const { data, isLoading } = useImpersonationSessions({ page: 1, pageSize: 30 })
  const startImpersonation = useStartImpersonation()

  const [activeSession, setActiveSession] = useState<{ name: string; tenantName: string } | null>(
    null
  )
  const [showModal, setShowModal] = useState(false)
  const [targetUserId, setTargetUserId] = useState('')
  const [reason, setReason] = useState('')
  const [search, setSearch] = useState('')

  const sessions = (data?.data ?? []).filter(
    (s) =>
      !search ||
      s.targetUserName.toLowerCase().includes(search.toLowerCase()) ||
      s.targetTenantName.toLowerCase().includes(search.toLowerCase())
  )

  async function handleStart() {
    if (!targetUserId || !reason) return
    const result = await startImpersonation.mutateAsync({ targetUserId, reason })
    // Store impersonation token
    if (typeof window !== 'undefined') {
      localStorage.setItem('impersonation_token', result.token)
      localStorage.setItem('impersonation_user', JSON.stringify(result.user))
    }
    setActiveSession({ name: result.user.name, tenantName: result.user.tenantName })
    setShowModal(false)
    setTargetUserId('')
    setReason('')
  }

  function handleEnd() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('impersonation_token')
      localStorage.removeItem('impersonation_user')
    }
    setActiveSession(null)
  }

  return (
    <>
      {activeSession && <ImpersonationBanner user={activeSession} onEnd={handleEnd} />}

      <div className={`space-y-8 ${activeSession ? 'pt-10' : ''}`}>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Impersonation</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Impersonate a university user to investigate or provide support. All sessions are
              logged.
            </p>
          </div>
          <Button onClick={() => setShowModal(true)} disabled={!!activeSession}>
            <LogIn className="mr-2 h-4 w-4" />
            Start Impersonation
          </Button>
        </div>

        {activeSession && (
          <div className="rounded-xl border border-yellow-200 bg-yellow-50 p-4 flex items-center gap-3">
            <AlertTriangle className="h-5 w-5 text-yellow-600 flex-shrink-0" />
            <div>
              <p className="font-medium text-yellow-900">Active impersonation session</p>
              <p className="text-sm text-yellow-700">
                You are currently impersonating <strong>{activeSession.name}</strong> at{' '}
                <strong>{activeSession.tenantName}</strong>. All API calls are made as this user.
              </p>
            </div>
            <Button
              size="sm"
              variant="outline"
              className="ml-auto border-yellow-600 text-yellow-800"
              onClick={handleEnd}
            >
              End Session
            </Button>
          </div>
        )}

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-10"
            placeholder="Search by user or university…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {/* Session History */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <History className="h-4 w-4" />
              Session History
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="flex justify-center py-10">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : sessions.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">
                No impersonation sessions yet
              </p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="pb-3 text-left font-medium">Admin</th>
                    <th className="pb-3 text-left font-medium">Target User</th>
                    <th className="pb-3 text-left font-medium">University</th>
                    <th className="pb-3 text-left font-medium">Reason</th>
                    <th className="pb-3 text-left font-medium">Started</th>
                    <th className="pb-3 text-left font-medium">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {sessions.map((s) => (
                    <tr key={s.id} className="hover:bg-muted/20 transition-colors">
                      <td className="py-3">
                        <div className="flex items-center gap-2">
                          <div className="h-7 w-7 rounded-full bg-muted flex items-center justify-center text-xs font-semibold">
                            {s.adminUserName.charAt(0)}
                          </div>
                          <span>{s.adminUserName}</span>
                        </div>
                      </td>
                      <td className="py-3 font-medium">{s.targetUserName}</td>
                      <td className="py-3 text-muted-foreground">{s.targetTenantName}</td>
                      <td className="py-3 text-muted-foreground max-w-[200px] truncate">
                        {s.reason}
                      </td>
                      <td className="py-3 text-muted-foreground">{formatDate(s.startedAt)}</td>
                      <td className="py-3">
                        {s.endedAt ? (
                          <Badge variant="outline" className="bg-gray-100 text-gray-600">
                            ended
                          </Badge>
                        ) : (
                          <Badge className="bg-yellow-100 text-yellow-800">active</Badge>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Start Impersonation Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-md rounded-2xl bg-background p-6 shadow-xl">
            <div className="flex items-center justify-between mb-5">
              <div className="flex items-center gap-2">
                <UserCog className="h-5 w-5" />
                <h2 className="text-lg font-semibold">Start Impersonation</h2>
              </div>
              <Button variant="ghost" size="icon" onClick={() => setShowModal(false)}>
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="rounded-lg bg-yellow-50 border border-yellow-200 p-3 mb-4">
              <p className="text-sm text-yellow-800">
                <AlertTriangle className="inline h-3.5 w-3.5 mr-1" />
                This session will be recorded in the audit log. Only impersonate users for
                legitimate support purposes.
              </p>
            </div>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Target User ID</Label>
                <Input
                  placeholder="UUID of the user to impersonate"
                  value={targetUserId}
                  onChange={(e) => setTargetUserId(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Reason (required for audit)</Label>
                <Input
                  placeholder="e.g. User reported login issue – ticket #1234"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="outline" onClick={() => setShowModal(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={handleStart}
                  disabled={!targetUserId || !reason || startImpersonation.isPending}
                  className="bg-yellow-500 hover:bg-yellow-600 text-yellow-900"
                >
                  {startImpersonation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Start Session
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
