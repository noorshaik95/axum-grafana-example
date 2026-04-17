'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loader2, User, Bell, Lock, Globe } from 'lucide-react'
import { useProfile } from '../../../../shared/lib/api/hooks'

export default function SettingsPage() {
  const { data: profile, isLoading } = useProfile()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  const initials = profile
    ? `${profile.firstName?.[0] ?? ''}${profile.lastName?.[0] ?? ''}`.toUpperCase()
    : '??'

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Settings</h1>
        <p className="text-slate-500 mt-1">Manage your account and preferences.</p>
      </div>

      <div className="grid gap-6">
        {/* Profile */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <User className="h-5 w-5" />
              Profile
            </CardTitle>
            <CardDescription>Your account information</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-start gap-4">
              <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-indigo-700 text-lg font-semibold">
                {initials}
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-lg text-slate-900">
                  {profile ? `${profile.firstName} ${profile.lastName}` : '--'}
                </h3>
                <p className="text-slate-500">{profile?.email ?? '--'}</p>
                {profile?.timezone && (
                  <p className="text-sm text-slate-400 mt-1">Timezone: {profile.timezone}</p>
                )}
                {profile?.bio && <p className="text-sm text-slate-600 mt-2">{profile.bio}</p>}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Notifications */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bell className="h-5 w-5" />
              Notification Preferences
            </CardTitle>
            <CardDescription>Choose what notifications you receive</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { label: 'Email Notifications', description: 'Receive notifications via email' },
                { label: 'New Submissions', description: 'Alert when students submit assignments' },
                { label: 'Student Questions', description: 'Alert when students ask questions' },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-slate-900">{item.label}</p>
                    <p className="text-sm text-slate-500">{item.description}</p>
                  </div>
                  <span className="text-xs font-medium text-emerald-600 bg-emerald-50 rounded-full px-2.5 py-0.5">
                    Enabled
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Security */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="h-5 w-5" />
              Security
            </CardTitle>
            <CardDescription>Manage your security settings</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <p className="font-medium text-slate-900">Two-Factor Authentication</p>
                <p className="text-sm text-slate-500 mb-2">
                  Add an extra layer of security to your account.
                </p>
                <Button size="sm" variant="outline">
                  Enable 2FA
                </Button>
              </div>
              <div>
                <p className="font-medium text-slate-900">Active Sessions</p>
                <p className="text-sm text-slate-500 mb-2">Manage your active sessions.</p>
                <Button size="sm" variant="outline">
                  View Sessions
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Course Defaults */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Globe className="h-5 w-5" />
              Course Defaults
            </CardTitle>
            <CardDescription>Default settings for new courses</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <p className="font-medium text-slate-900">Default Grading Scale</p>
                <p className="text-sm text-slate-500 mb-2">
                  A: 90-100, B: 80-89, C: 70-79, D: 60-69, F: 0-59
                </p>
                <Button size="sm" variant="outline">
                  Edit Scale
                </Button>
              </div>
              <div>
                <p className="font-medium text-slate-900">Late Submission Policy</p>
                <p className="text-sm text-slate-500 mb-2">10% deduction per day</p>
                <Button size="sm" variant="outline">
                  Edit Policy
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
