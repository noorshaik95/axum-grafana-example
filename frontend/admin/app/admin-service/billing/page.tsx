'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { CreditCard, Download, Calendar, CheckCircle2 } from 'lucide-react'

const BILLING_HISTORY = [
  {
    id: '1',
    date: '2024-01-01',
    description: 'Pro Plan - Monthly Subscription',
    amount: 299.0,
    status: 'paid',
    invoice: 'INV-2024-001',
  },
  {
    id: '2',
    date: '2023-12-01',
    description: 'Pro Plan - Monthly Subscription',
    amount: 299.0,
    status: 'paid',
    invoice: 'INV-2023-012',
  },
  {
    id: '3',
    date: '2023-12-15',
    description: 'Additional Storage - 100GB',
    amount: 49.0,
    status: 'paid',
    invoice: 'INV-2023-013',
  },
  {
    id: '4',
    date: '2023-11-01',
    description: 'Pro Plan - Monthly Subscription',
    amount: 299.0,
    status: 'paid',
    invoice: 'INV-2023-011',
  },
]

export default function BillingPage() {
  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Billing</h1>
          <p className="text-muted-foreground">Manage your subscription and payment methods</p>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Current Plan</CardTitle>
                  <CardDescription>Your active subscription</CardDescription>
                </div>
                <Badge variant="success">Active</Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <div className="text-3xl font-bold">Pro Plan</div>
                <div className="text-muted-foreground">$299/month</div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                  <span className="text-sm">Up to 10,000 users</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                  <span className="text-sm">500 GB storage</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                  <span className="text-sm">Unlimited bandwidth</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                  <span className="text-sm">50,000 AI credits/month</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                  <span className="text-sm">Priority support</span>
                </div>
              </div>

              <div className="pt-4 space-y-2">
                <Button className="w-full">Upgrade Plan</Button>
                <Button variant="outline" className="w-full">
                  Manage Subscription
                </Button>
              </div>
            </CardContent>
          </Card>

          <div className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Next Billing Date</CardTitle>
                    <CardDescription>Automatic renewal</CardDescription>
                  </div>
                  <Calendar className="h-8 w-8 text-muted-foreground" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">February 1, 2024</div>
                <p className="text-sm text-muted-foreground mt-1">
                  $299.00 will be charged to your payment method
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Payment Method</CardTitle>
                    <CardDescription>Default payment</CardDescription>
                  </div>
                  <CreditCard className="h-8 w-8 text-muted-foreground" />
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center gap-4">
                  <div className="flex h-12 w-16 items-center justify-center rounded border bg-muted">
                    <CreditCard className="h-6 w-6" />
                  </div>
                  <div className="flex-1">
                    <div className="font-medium">Visa ending in 4242</div>
                    <div className="text-sm text-muted-foreground">Expires 12/2025</div>
                  </div>
                  <Badge>Default</Badge>
                </div>
                <Button variant="outline" className="w-full">
                  Update Payment Method
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Billing History</CardTitle>
                <CardDescription>Your past invoices and payments</CardDescription>
              </div>
              <Button variant="outline" size="sm">
                <Download className="mr-2 h-4 w-4" />
                Export All
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Invoice</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {BILLING_HISTORY.map((transaction) => (
                  <TableRow key={transaction.id}>
                    <TableCell className="font-medium">
                      {new Date(transaction.date).toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                      })}
                    </TableCell>
                    <TableCell>{transaction.description}</TableCell>
                    <TableCell className="font-medium">${transaction.amount.toFixed(2)}</TableCell>
                    <TableCell>
                      <Badge variant="success">
                        <CheckCircle2 className="mr-1 h-3 w-3" />
                        {transaction.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {transaction.invoice}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm">
                        <Download className="h-4 w-4" />
                      </Button>
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
