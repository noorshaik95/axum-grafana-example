'use client'

import { PlatformStatsCards } from '@/components/dashboard/PlatformStatsCards'
import { ServiceHealthGrid } from '@/components/dashboard/ServiceHealthGrid'
import { KafkaLagWidget } from '@/components/dashboard/KafkaLagWidget'
import Link from 'next/link'
import { ArrowRight } from 'lucide-react'

function QuickActionCard({
  title,
  description,
  href,
  label,
}: {
  title: string
  description: string
  href: string
  label: string
}) {
  return (
    <div className="rounded-xl border bg-card p-5 shadow-sm">
      <h3 className="text-base font-semibold">{title}</h3>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      <Link
        href={href}
        className="mt-4 inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium hover:bg-accent transition-colors"
      >
        {label}
        <ArrowRight className="h-4 w-4" />
      </Link>
    </div>
  )
}

export default function DashboardPage() {
  const today = new Date().toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="mt-1 text-sm text-muted-foreground">{today}</p>
      </div>

      <PlatformStatsCards />

      <div className="grid gap-6 lg:grid-cols-2">
        <ServiceHealthGrid />
        <KafkaLagWidget />
      </div>

      <div>
        <h2 className="mb-4 text-lg font-semibold">Quick Actions</h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <QuickActionCard
            title="Onboard University"
            description="Start the onboarding process for a new institution."
            href="/onboarding/new"
            label="Start Onboarding"
          />
          <QuickActionCard
            title="User Management"
            description="View, create, and manage admin user accounts."
            href="/iam/users"
            label="Manage Users"
          />
          <QuickActionCard
            title="System Health"
            description="Check backend service status and infrastructure health."
            href="/system/health"
            label="View Status"
          />
        </div>
      </div>
    </div>
  )
}
