'use client'

import Link from 'next/link'
import { Search } from 'lucide-react'

const SCHOOLS = [
  {
    id: '1',
    name: 'Stanford University',
    location: 'Palo Alto, CA',
    joined: 'Jan 2024',
    tier: 'Enterprise',
    seats: { used: 12400, cap: 15000 },
    mau: 9800,
    health: 'P1',
    renew: 'Dec 2026',
    initials: 'SU',
    color: '#8B0000',
  },
  {
    id: '2',
    name: 'MIT',
    location: 'Cambridge, MA',
    joined: 'Feb 2024',
    tier: 'Enterprise',
    seats: { used: 11200, cap: 12000 },
    mau: 8900,
    health: 'healthy',
    renew: 'Jan 2027',
    initials: 'MIT',
    color: '#A31F34',
  },
  {
    id: '3',
    name: 'UC Berkeley',
    location: 'Berkeley, CA',
    joined: 'Mar 2024',
    tier: 'Growth',
    seats: { used: 8400, cap: 10000 },
    mau: 6700,
    health: 'healthy',
    renew: 'Mar 2027',
    initials: 'UCB',
    color: '#003262',
  },
  {
    id: '4',
    name: 'Harvard University',
    location: 'Cambridge, MA',
    joined: 'Apr 2024',
    tier: 'Enterprise',
    seats: { used: 10900, cap: 12000 },
    mau: 8100,
    health: 'watch',
    renew: 'Apr 2027',
    initials: 'HU',
    color: '#A51C30',
  },
  {
    id: '5',
    name: 'Yale University',
    location: 'New Haven, CT',
    joined: 'May 2024',
    tier: 'Growth',
    seats: { used: 7200, cap: 8000 },
    mau: 5400,
    health: 'healthy',
    renew: 'May 2027',
    initials: 'YU',
    color: '#00356B',
  },
  {
    id: '6',
    name: 'Cornell University',
    location: 'Ithaca, NY',
    joined: 'Jun 2024',
    tier: 'Starter',
    seats: { used: 4100, cap: 5000 },
    mau: 3200,
    health: 'healthy',
    renew: 'Jun 2027',
    initials: 'CU',
    color: '#B31B1B',
  },
  {
    id: '7',
    name: 'Oxford University',
    location: 'Oxford, UK',
    joined: 'Jul 2024',
    tier: 'Enterprise',
    seats: { used: 14500, cap: 15000 },
    mau: 11200,
    health: 'healthy',
    renew: 'Jul 2027',
    initials: 'OU',
    color: '#002147',
  },
]

const HEALTH_BADGE: Record<string, { label: string; cls: string }> = {
  healthy: { label: 'Healthy', cls: 'bg-[#dde9df] text-[#234e32]' },
  watch: { label: 'Watch', cls: 'bg-[#fff3cd] text-[#856404]' },
  P1: { label: 'P1', cls: 'bg-[#fde8e0] text-[#d97757] font-semibold' },
}

const TIER_BADGE: Record<string, string> = {
  Enterprise: 'bg-[#234e32] text-white',
  Growth: 'bg-[#dde9df] text-[#234e32]',
  Starter: 'bg-[#f6f3ec] text-[#6a6e62]',
}

export default function SchoolsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1
          className="text-2xl font-bold text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          Schools
        </h1>
        <p className="text-sm text-[#6a6e62] mt-1">All tenant institutions on the platform</p>
      </div>

      {/* Search / filter bar */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 flex-1 max-w-sm px-3 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm">
          <Search className="h-4 w-4 text-[#6a6e62] shrink-0" />
          <input
            className="flex-1 bg-transparent outline-none text-[#12170f] placeholder-[#6a6e62]"
            placeholder="Search · filter by tier, region, health"
          />
        </div>
        <select className="px-3 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm text-[#6a6e62]">
          <option>All tiers</option>
          <option>Enterprise</option>
          <option>Growth</option>
          <option>Starter</option>
        </select>
        <select className="px-3 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm text-[#6a6e62]">
          <option>All health</option>
          <option>Healthy</option>
          <option>Watch</option>
          <option>P1</option>
        </select>
        <Link
          href="/onboard/school"
          className="ml-auto px-4 py-2 rounded-lg bg-[#234e32] text-white text-sm font-medium hover:bg-[#1a3a26] transition-colors"
        >
          + Onboard school
        </Link>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[#e4e0d4] bg-[#f6f3ec]">
              <th className="text-left px-4 py-3 font-medium text-[#6a6e62]">School</th>
              <th className="text-left px-4 py-3 font-medium text-[#6a6e62]">Tier</th>
              <th className="text-left px-4 py-3 font-medium text-[#6a6e62]">Seats</th>
              <th className="text-left px-4 py-3 font-medium text-[#6a6e62]">MAU</th>
              <th className="text-left px-4 py-3 font-medium text-[#6a6e62]">Health</th>
              <th className="text-left px-4 py-3 font-medium text-[#6a6e62]">Renew</th>
            </tr>
          </thead>
          <tbody>
            {SCHOOLS.map((school, i) => {
              const healthInfo = HEALTH_BADGE[school.health] ?? HEALTH_BADGE.healthy
              const seatPct = Math.round((school.seats.used / school.seats.cap) * 100)
              return (
                <tr
                  key={school.id}
                  className={`border-b border-[#e4e0d4] hover:bg-[#f2f7f3] transition-colors ${i === SCHOOLS.length - 1 ? 'border-b-0' : ''}`}
                >
                  <td className="px-4 py-3">
                    <Link href={`/schools/${school.id}`} className="flex items-center gap-3">
                      <div
                        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-white text-[10px] font-bold"
                        style={{ backgroundColor: school.color }}
                      >
                        {school.initials}
                      </div>
                      <div>
                        <p className="font-medium text-[#12170f] hover:text-[#234e32]">
                          {school.name}
                        </p>
                        <p className="text-xs text-[#6a6e62]">
                          {school.location} · joined {school.joined}
                        </p>
                      </div>
                    </Link>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-0.5 rounded-full text-xs font-medium ${TIER_BADGE[school.tier]}`}
                    >
                      {school.tier}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <p className="text-[#12170f] text-xs">
                      {school.seats.used.toLocaleString()} / {school.seats.cap.toLocaleString()}
                    </p>
                    <div className="mt-1 h-1.5 w-20 rounded-full bg-[#e4e0d4] overflow-hidden">
                      <div
                        className="h-full rounded-full"
                        style={{
                          width: `${seatPct}%`,
                          backgroundColor: seatPct > 90 ? '#d97757' : '#3e7d4f',
                        }}
                      />
                    </div>
                  </td>
                  <td className="px-4 py-3 text-[#12170f]">{school.mau.toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs ${healthInfo.cls}`}>
                      {healthInfo.label}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-[#6a6e62] text-xs">{school.renew}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
        <div className="px-4 py-3 border-t border-[#e4e0d4] bg-[#f6f3ec]">
          <Link href="#" className="text-sm text-[#234e32] hover:underline">
            + 35 more schools
          </Link>
        </div>
      </div>
    </div>
  )
}
