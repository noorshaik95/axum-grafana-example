'use client'

import { Button } from '@/components/ui/button'
import { ArrowLeft } from 'lucide-react'
import Link from 'next/link'
import { GradingRulesEditor } from '@/components/grading/GradingRulesEditor'

export default function GradingRulesPage() {
  return (
    <div className="space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href="/grading">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Grading
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Grading Rules</h1>
        <p className="text-slate-500 mt-1">Configure assignment type weights and grade scales.</p>
      </div>

      <GradingRulesEditor />
    </div>
  )
}
