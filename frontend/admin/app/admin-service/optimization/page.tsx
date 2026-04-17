'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { useToast } from '@/hooks/use-toast'
import {
  TrendingDown,
  Lightbulb,
  DollarSign,
  Zap,
  HardDrive,
  Users,
  CheckCircle2,
} from 'lucide-react'

const RECOMMENDATIONS = [
  {
    id: '1',
    title: 'Optimize Storage Usage',
    description: 'Remove duplicate files and compress media content to save 45 GB of storage',
    potential_savings: 22.5,
    impact: 'high',
    implementation_effort: 'easy',
    category: 'storage',
  },
  {
    id: '2',
    title: 'Reduce Inactive User Licenses',
    description: "Deactivate 450 users who haven't logged in for 90+ days",
    potential_savings: 67.5,
    impact: 'high',
    implementation_effort: 'medium',
    category: 'users',
  },
  {
    id: '3',
    title: 'Optimize AI Credit Usage',
    description:
      'Switch to batch processing for automated tasks to reduce AI credit consumption by 25%',
    potential_savings: 49.75,
    impact: 'medium',
    implementation_effort: 'medium',
    category: 'ai',
  },
  {
    id: '4',
    title: 'Enable Content Caching',
    description: 'Implement CDN caching to reduce bandwidth usage by 30%',
    potential_savings: 35.0,
    impact: 'medium',
    implementation_effort: 'easy',
    category: 'bandwidth',
  },
  {
    id: '5',
    title: 'Downgrade Unused Features',
    description: 'Switch to a lower tier plan for departments with minimal usage',
    potential_savings: 75.0,
    impact: 'low',
    implementation_effort: 'hard',
    category: 'plan',
  },
]

export default function OptimizationPage() {
  const { toast } = useToast()

  const totalSavings = RECOMMENDATIONS.reduce((sum, rec) => sum + rec.potential_savings, 0)
  const savingsPercentage = ((totalSavings / 299) * 100).toFixed(1)

  const handleApply = (id: string, title: string) => {
    toast({
      title: 'Optimization Applied',
      description: `Successfully applied: ${title}`,
    })
  }

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'high':
        return 'text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400'
      case 'medium':
        return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400'
      case 'low':
        return 'text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400'
      default:
        return ''
    }
  }

  const getEffortColor = (effort: string) => {
    switch (effort) {
      case 'easy':
        return 'success'
      case 'medium':
        return 'warning'
      case 'hard':
        return 'destructive'
      default:
        return 'default'
    }
  }

  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Cost Optimization</h1>
          <p className="text-muted-foreground">
            AI-powered recommendations to reduce costs by up to 30%
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center gap-2">
                <DollarSign className="h-5 w-5 text-muted-foreground" />
                <CardDescription>Current Monthly Cost</CardDescription>
              </div>
              <CardTitle className="text-3xl">$299.00</CardTitle>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center gap-2">
                <TrendingDown className="h-5 w-5 text-green-600" />
                <CardDescription>Potential Savings</CardDescription>
              </div>
              <CardTitle className="text-3xl text-green-600">${totalSavings.toFixed(2)}</CardTitle>
              <p className="text-xs text-muted-foreground mt-1">
                {savingsPercentage}% reduction possible
              </p>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center gap-2">
                <Lightbulb className="h-5 w-5 text-yellow-600" />
                <CardDescription>Recommendations</CardDescription>
              </div>
              <CardTitle className="text-3xl">{RECOMMENDATIONS.length}</CardTitle>
              <p className="text-xs text-muted-foreground mt-1">Actionable insights available</p>
            </CardHeader>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Savings Breakdown</CardTitle>
            <CardDescription>Estimated monthly savings by category</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <HardDrive className="h-4 w-4" />
                  Storage Optimization
                </span>
                <span className="font-medium">$22.50</span>
              </div>
              <Progress value={(22.5 / totalSavings) * 100} />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <Users className="h-4 w-4" />
                  User License Optimization
                </span>
                <span className="font-medium">$67.50</span>
              </div>
              <Progress value={(67.5 / totalSavings) * 100} />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <Zap className="h-4 w-4" />
                  AI Credit Optimization
                </span>
                <span className="font-medium">$49.75</span>
              </div>
              <Progress value={(49.75 / totalSavings) * 100} />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <TrendingDown className="h-4 w-4" />
                  Other Optimizations
                </span>
                <span className="font-medium">$110.00</span>
              </div>
              <Progress value={(110 / totalSavings) * 100} />
            </div>
          </CardContent>
        </Card>

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-bold">Recommendations</h2>
            <Badge variant="outline">{RECOMMENDATIONS.length} opportunities</Badge>
          </div>

          {RECOMMENDATIONS.map((rec) => (
            <Card key={rec.id}>
              <CardHeader>
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <CardTitle className="text-lg">{rec.title}</CardTitle>
                      <Badge variant={getEffortColor(rec.implementation_effort) as any}>
                        {rec.implementation_effort}
                      </Badge>
                    </div>
                    <CardDescription>{rec.description}</CardDescription>
                  </div>
                  <div className="text-right">
                    <div className="text-2xl font-bold text-green-600">
                      ${rec.potential_savings.toFixed(2)}
                    </div>
                    <div className="text-xs text-muted-foreground">per month</div>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-muted-foreground">Impact:</span>
                      <Badge className={getImpactColor(rec.impact)}>{rec.impact}</Badge>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-muted-foreground">Category:</span>
                      <Badge variant="outline">{rec.category}</Badge>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm">
                      Learn More
                    </Button>
                    <Button size="sm" onClick={() => handleApply(rec.id, rec.title)}>
                      <CheckCircle2 className="mr-2 h-4 w-4" />
                      Apply
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </>
  )
}
