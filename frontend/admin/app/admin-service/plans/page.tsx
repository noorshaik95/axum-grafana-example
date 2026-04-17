'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { Check, Zap, Crown, Rocket } from 'lucide-react'

const PLANS = [
  {
    id: 'starter',
    name: 'Starter',
    description: 'Perfect for small institutions',
    price: 99,
    billing: 'monthly',
    icon: Zap,
    features: [
      'Up to 1,000 users',
      '100 GB storage',
      '1 TB bandwidth/month',
      '10,000 AI credits/month',
      'Email support',
      'Basic analytics',
    ],
    limits: {
      users: 1000,
      storage: 100,
      bandwidth: 1000,
      ai_credits: 10000,
    },
  },
  {
    id: 'pro',
    name: 'Pro',
    description: 'Most popular for universities',
    price: 299,
    billing: 'monthly',
    icon: Rocket,
    current: true,
    popular: true,
    features: [
      'Up to 10,000 users',
      '500 GB storage',
      'Unlimited bandwidth',
      '50,000 AI credits/month',
      'Priority support',
      'Advanced analytics',
      'Custom integrations',
      'SSO support',
    ],
    limits: {
      users: 10000,
      storage: 500,
      bandwidth: -1,
      ai_credits: 50000,
    },
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    description: 'For large institutions',
    price: 999,
    billing: 'monthly',
    icon: Crown,
    features: [
      'Unlimited users',
      '2 TB storage',
      'Unlimited bandwidth',
      '200,000 AI credits/month',
      '24/7 dedicated support',
      'Custom analytics',
      'White-label options',
      'SLA guarantees',
      'On-premise deployment',
    ],
    limits: {
      users: -1,
      storage: 2000,
      bandwidth: -1,
      ai_credits: 200000,
    },
  },
]

const ADDONS = [
  {
    id: 'storage',
    name: 'Additional Storage',
    description: '100 GB increments',
    price: 49,
    unit: '100 GB',
  },
  {
    id: 'ai_credits',
    name: 'AI Credit Pack',
    description: 'Extra AI processing credits',
    price: 99,
    unit: '50,000 credits',
  },
  {
    id: 'bandwidth',
    name: 'Bandwidth Boost',
    description: 'Additional data transfer',
    price: 79,
    unit: '5 TB',
  },
]

export default function PlansPage() {
  const { toast } = useToast()

  const handleUpgrade = (planId: string) => {
    toast({
      title: 'Plan Upgrade',
      description: `Upgrading to ${planId} plan...`,
    })
  }

  const handlePurchaseAddon = (addonId: string) => {
    toast({
      title: 'Add-on Purchased',
      description: `Successfully purchased ${addonId}`,
    })
  }

  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Plans & Upgrades</h1>
          <p className="text-muted-foreground">Choose the perfect plan for your institution</p>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          {PLANS.map((plan) => {
            const Icon = plan.icon
            return (
              <Card key={plan.id} className={plan.current ? 'border-primary shadow-lg' : ''}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-primary/10">
                        <Icon className="h-6 w-6 text-primary" />
                      </div>
                      <div>
                        <CardTitle>{plan.name}</CardTitle>
                        <CardDescription>{plan.description}</CardDescription>
                      </div>
                    </div>
                    {plan.popular && <Badge variant="default">Popular</Badge>}
                    {plan.current && <Badge variant="success">Current</Badge>}
                  </div>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div>
                    <div className="flex items-baseline gap-1">
                      <span className="text-4xl font-bold">${plan.price}</span>
                      <span className="text-muted-foreground">/{plan.billing}</span>
                    </div>
                  </div>

                  <div className="space-y-2">
                    {plan.features.map((feature, index) => (
                      <div key={index} className="flex items-start gap-2">
                        <Check className="h-4 w-4 text-green-600 mt-0.5 shrink-0" />
                        <span className="text-sm">{feature}</span>
                      </div>
                    ))}
                  </div>

                  <Button
                    className="w-full"
                    variant={plan.current ? 'outline' : 'default'}
                    disabled={plan.current}
                    onClick={() => handleUpgrade(plan.id)}
                  >
                    {plan.current ? 'Current Plan' : 'Upgrade to ' + plan.name}
                  </Button>
                </CardContent>
              </Card>
            )
          })}
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Add-ons & Resources</CardTitle>
            <CardDescription>Purchase additional resources as needed</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              {ADDONS.map((addon) => (
                <Card key={addon.id}>
                  <CardHeader>
                    <CardTitle className="text-lg">{addon.name}</CardTitle>
                    <CardDescription>{addon.description}</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div>
                      <div className="flex items-baseline gap-1">
                        <span className="text-3xl font-bold">${addon.price}</span>
                        <span className="text-muted-foreground text-sm">/{addon.unit}</span>
                      </div>
                    </div>
                    <Button
                      className="w-full"
                      variant="outline"
                      onClick={() => handlePurchaseAddon(addon.id)}
                    >
                      Purchase
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Plan Comparison</CardTitle>
            <CardDescription>Compare features across all plans</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-4">Feature</th>
                    {PLANS.map((plan) => (
                      <th key={plan.id} className="text-center p-4">
                        {plan.name}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b">
                    <td className="p-4">Users</td>
                    {PLANS.map((plan) => (
                      <td key={plan.id} className="text-center p-4">
                        {plan.limits.users === -1
                          ? 'Unlimited'
                          : plan.limits.users.toLocaleString()}
                      </td>
                    ))}
                  </tr>
                  <tr className="border-b">
                    <td className="p-4">Storage</td>
                    {PLANS.map((plan) => (
                      <td key={plan.id} className="text-center p-4">
                        {plan.limits.storage} GB
                      </td>
                    ))}
                  </tr>
                  <tr className="border-b">
                    <td className="p-4">Bandwidth</td>
                    {PLANS.map((plan) => (
                      <td key={plan.id} className="text-center p-4">
                        {plan.limits.bandwidth === -1 ? 'Unlimited' : `${plan.limits.bandwidth} TB`}
                      </td>
                    ))}
                  </tr>
                  <tr className="border-b">
                    <td className="p-4">AI Credits</td>
                    {PLANS.map((plan) => (
                      <td key={plan.id} className="text-center p-4">
                        {plan.limits.ai_credits.toLocaleString()}/month
                      </td>
                    ))}
                  </tr>
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
