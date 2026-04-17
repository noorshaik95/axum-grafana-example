'use client'

import { UniversityList } from '@/components/universities/UniversityList'

export default function UniversitiesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Universities</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Manage all registered institutions on the platform
        </p>
      </div>
      <UniversityList />
    </div>
  )
}
