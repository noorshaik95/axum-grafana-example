'use client'

import { useKafkaLag } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { Loader2, Radio } from 'lucide-react'

export function KafkaLagWidget() {
  const { data: lagData, isLoading } = useKafkaLag()

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Radio className="h-4 w-4" />
            Kafka Consumer Lag
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    )
  }

  const groupedByTopic = (lagData ?? []).reduce(
    (acc, item) => {
      if (!acc[item.topic]) acc[item.topic] = []
      acc[item.topic]!.push(item)
      return acc
    },
    {} as Record<string, typeof lagData>
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Radio className="h-4 w-4" />
          Kafka Consumer Lag
        </CardTitle>
      </CardHeader>
      <CardContent>
        {Object.keys(groupedByTopic).length === 0 ? (
          <p className="py-4 text-center text-sm text-muted-foreground">No consumer lag data</p>
        ) : (
          <div className="space-y-3">
            {Object.entries(groupedByTopic).map(([topic, partitions]) => {
              const totalLag = (partitions ?? []).reduce((sum, p) => sum + p.lag, 0)
              const lagLevel =
                totalLag === 0 ? 'success' : totalLag < 1000 ? 'warning' : 'destructive'
              return (
                <div
                  key={topic}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div>
                    <p className="text-sm font-medium">{topic}</p>
                    <p className="text-xs text-muted-foreground">
                      {partitions?.length ?? 0} partition
                      {(partitions?.length ?? 0) !== 1 ? 's' : ''} |{' '}
                      {partitions?.[0]?.consumerGroup ?? 'unknown'}
                    </p>
                  </div>
                  <Badge
                    variant={lagLevel as 'success' | 'warning' | 'destructive'}
                    className={cn(
                      'text-xs',
                      lagLevel === 'success' && 'bg-green-100 text-green-800'
                    )}
                  >
                    {totalLag.toLocaleString()} lag
                  </Badge>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
