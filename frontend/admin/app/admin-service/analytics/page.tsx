'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Users, HardDrive, Activity, Zap } from 'lucide-react'
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

const USER_DATA = [
  { name: 'Jan', active: 4000, total: 5000 },
  { name: 'Feb', active: 4200, total: 5200 },
  { name: 'Mar', active: 4500, total: 5500 },
  { name: 'Apr', active: 4800, total: 5800 },
  { name: 'May', active: 5200, total: 6200 },
  { name: 'Jun', active: 5500, total: 6500 },
]

const STORAGE_DATA = [
  { name: 'Week 1', used: 120, limit: 500 },
  { name: 'Week 2', used: 145, limit: 500 },
  { name: 'Week 3', used: 168, limit: 500 },
  { name: 'Week 4', used: 195, limit: 500 },
]

const BANDWIDTH_DATA = [
  { name: 'Mon', bandwidth: 1200 },
  { name: 'Tue', bandwidth: 1450 },
  { name: 'Wed', bandwidth: 1680 },
  { name: 'Thu', bandwidth: 1350 },
  { name: 'Fri', bandwidth: 1890 },
  { name: 'Sat', bandwidth: 980 },
  { name: 'Sun', bandwidth: 750 },
]

const AI_CREDITS_DATA = [
  { name: 'Jan', credits: 8500 },
  { name: 'Feb', credits: 9200 },
  { name: 'Mar', credits: 10100 },
  { name: 'Apr', credits: 11500 },
  { name: 'May', credits: 12800 },
  { name: 'Jun', credits: 14200 },
]

export default function AnalyticsPage() {
  return (
    <>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Usage Analytics</h1>
            <p className="text-muted-foreground">
              Real-time monitoring of resources and usage patterns
            </p>
          </div>
          <Select defaultValue="30d">
            <SelectTrigger className="w-32">
              <SelectValue placeholder="Time range" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="7d">Last 7 days</SelectItem>
              <SelectItem value="30d">Last 30 days</SelectItem>
              <SelectItem value="90d">Last 90 days</SelectItem>
              <SelectItem value="1y">Last year</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Tabs defaultValue="users" className="space-y-6">
          <TabsList>
            <TabsTrigger value="users">
              <Users className="mr-2 h-4 w-4" />
              Users
            </TabsTrigger>
            <TabsTrigger value="storage">
              <HardDrive className="mr-2 h-4 w-4" />
              Storage
            </TabsTrigger>
            <TabsTrigger value="bandwidth">
              <Activity className="mr-2 h-4 w-4" />
              Bandwidth
            </TabsTrigger>
            <TabsTrigger value="ai-credits">
              <Zap className="mr-2 h-4 w-4" />
              AI Credits
            </TabsTrigger>
          </TabsList>

          <TabsContent value="users" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Total Users</CardDescription>
                  <CardTitle className="text-3xl">6,500</CardTitle>
                  <p className="text-xs text-green-600">+8.2% from last month</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Active Users</CardDescription>
                  <CardTitle className="text-3xl">5,500</CardTitle>
                  <p className="text-xs text-green-600">+5.4% from last month</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Activation Rate</CardDescription>
                  <CardTitle className="text-3xl">84.6%</CardTitle>
                  <p className="text-xs text-muted-foreground">Steady growth</p>
                </CardHeader>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle>User Growth Trend</CardTitle>
                <CardDescription>Total vs Active users over time</CardDescription>
              </CardHeader>
              <CardContent className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={USER_DATA}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Area
                      type="monotone"
                      dataKey="total"
                      stackId="1"
                      stroke="#8884d8"
                      fill="#8884d8"
                    />
                    <Area
                      type="monotone"
                      dataKey="active"
                      stackId="2"
                      stroke="#82ca9d"
                      fill="#82ca9d"
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="storage" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Storage Used</CardDescription>
                  <CardTitle className="text-3xl">195 GB</CardTitle>
                  <p className="text-xs text-muted-foreground">of 500 GB</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Usage Rate</CardDescription>
                  <CardTitle className="text-3xl">39%</CardTitle>
                  <p className="text-xs text-yellow-600">Moderate usage</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Projected Full</CardDescription>
                  <CardTitle className="text-3xl">8 months</CardTitle>
                  <p className="text-xs text-muted-foreground">At current rate</p>
                </CardHeader>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle>Storage Usage Trend</CardTitle>
                <CardDescription>Weekly storage consumption</CardDescription>
              </CardHeader>
              <CardContent className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={STORAGE_DATA}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="used" fill="#3b82f6" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="bandwidth" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>This Week</CardDescription>
                  <CardTitle className="text-3xl">9.3 TB</CardTitle>
                  <p className="text-xs text-green-600">+12% from last week</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Daily Average</CardDescription>
                  <CardTitle className="text-3xl">1.33 TB</CardTitle>
                  <p className="text-xs text-muted-foreground">Typical usage</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Peak Day</CardDescription>
                  <CardTitle className="text-3xl">1.89 TB</CardTitle>
                  <p className="text-xs text-muted-foreground">Friday</p>
                </CardHeader>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle>Bandwidth Usage</CardTitle>
                <CardDescription>Daily bandwidth consumption</CardDescription>
              </CardHeader>
              <CardContent className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={BANDWIDTH_DATA}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Line type="monotone" dataKey="bandwidth" stroke="#8b5cf6" strokeWidth={2} />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="ai-credits" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Credits Used</CardDescription>
                  <CardTitle className="text-3xl">14,200</CardTitle>
                  <p className="text-xs text-muted-foreground">This month</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Credits Remaining</CardDescription>
                  <CardTitle className="text-3xl">35,800</CardTitle>
                  <p className="text-xs text-green-600">72% available</p>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardDescription>Growth Rate</CardDescription>
                  <CardTitle className="text-3xl">+24%</CardTitle>
                  <p className="text-xs text-yellow-600">Month over month</p>
                </CardHeader>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle>AI Credits Consumption</CardTitle>
                <CardDescription>Monthly usage trend</CardDescription>
              </CardHeader>
              <CardContent className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={AI_CREDITS_DATA}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Area type="monotone" dataKey="credits" stroke="#f59e0b" fill="#fbbf24" />
                  </AreaChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </>
  )
}
