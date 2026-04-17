'use client'

import { useState } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { useToast } from '@/hooks/use-toast'
import { Upload, FileSpreadsheet, Database, Cloud, Check, Loader2 } from 'lucide-react'

export default function BulkImportPage() {
  const [importMethod, setImportMethod] = useState<'csv' | 'api'>('csv')
  const [roleType, setRoleType] = useState('student')
  const [file, setFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [progress, setProgress] = useState(0)
  const { toast } = useToast()

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0])
    }
  }

  const handleImport = () => {
    if (!file && importMethod === 'csv') {
      toast({
        title: 'No file selected',
        description: 'Please select a CSV file to import',
        variant: 'destructive',
      })
      return
    }

    setImporting(true)
    setProgress(0)

    // Simulate import progress
    const interval = setInterval(() => {
      setProgress((prev) => {
        if (prev >= 100) {
          clearInterval(interval)
          setImporting(false)
          toast({
            title: 'Import Completed',
            description: 'Successfully imported 10,000 users in 1.8 minutes',
          })
          return 100
        }
        return prev + 5
      })
    }, 200)
  }

  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Bulk Import</h1>
          <p className="text-muted-foreground">
            Import users efficiently via CSV, API, or directory sync
          </p>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Import Configuration</CardTitle>
              <CardDescription>Upload up to 10,000+ users in under 2 minutes</CardDescription>
            </CardHeader>
            <CardContent>
              <Tabs value={importMethod} onValueChange={(v) => setImportMethod(v as any)}>
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="csv">
                    <FileSpreadsheet className="mr-2 h-4 w-4" />
                    CSV Upload
                  </TabsTrigger>
                  <TabsTrigger value="api">
                    <Database className="mr-2 h-4 w-4" />
                    API Import
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="csv" className="space-y-4 mt-6">
                  <div className="space-y-2">
                    <Label htmlFor="role-type">User Role</Label>
                    <Select value={roleType} onValueChange={setRoleType}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select role type" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="student">Student</SelectItem>
                        <SelectItem value="instructor">Instructor</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="csv-file">CSV File</Label>
                    <div className="flex items-center gap-4">
                      <Input
                        id="csv-file"
                        type="file"
                        accept=".csv"
                        onChange={handleFileChange}
                        disabled={importing}
                      />
                      {file && (
                        <Badge variant="success">
                          <Check className="mr-1 h-3 w-3" />
                          {file.name}
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Supports CSV files with columns: email, name, student_id
                    </p>
                  </div>

                  {importing && (
                    <div className="space-y-2">
                      <div className="flex items-center justify-between text-sm">
                        <span>Import Progress</span>
                        <span className="font-medium">{progress}%</span>
                      </div>
                      <Progress value={progress} />
                      <p className="text-xs text-muted-foreground">
                        Processing {Math.floor((progress / 100) * 10000)} / 10,000 users
                      </p>
                    </div>
                  )}

                  <Button onClick={handleImport} disabled={importing} className="w-full">
                    {importing ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Importing...
                      </>
                    ) : (
                      <>
                        <Upload className="mr-2 h-4 w-4" />
                        Start Import
                      </>
                    )}
                  </Button>
                </TabsContent>

                <TabsContent value="api" className="space-y-4 mt-6">
                  <div className="rounded-lg border p-4 bg-muted/50">
                    <h4 className="text-sm font-medium mb-2">API Endpoint</h4>
                    <code className="text-xs bg-background px-2 py-1 rounded">
                      POST /api/onboarding/bulk-import
                    </code>
                  </div>

                  <div className="space-y-2">
                    <Label>Sample Request Body</Label>
                    <pre className="text-xs bg-muted p-4 rounded-lg overflow-x-auto">
                      {`{
  "users": [
    {
      "email": "student@university.edu",
      "name": "John Doe",
      "student_id": "12345"
    }
  ],
  "role_type": "student",
  "auto_provision": true
}`}
                    </pre>
                  </div>

                  <Button variant="outline" className="w-full">
                    <Cloud className="mr-2 h-4 w-4" />
                    View API Documentation
                  </Button>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>

          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Import Statistics</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Last Import</span>
                  <span className="text-sm font-medium">2 hours ago</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Total Imported</span>
                  <span className="text-sm font-medium">45,230 users</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Success Rate</span>
                  <Badge variant="success">99.8%</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Avg. Speed</span>
                  <span className="text-sm font-medium">5,500/min</span>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Features</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex items-start gap-2">
                  <Check className="h-4 w-4 text-green-600 mt-0.5" />
                  <div className="text-sm">
                    <div className="font-medium">Lightning Fast</div>
                    <div className="text-muted-foreground">10,000+ users in under 2 minutes</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <Check className="h-4 w-4 text-green-600 mt-0.5" />
                  <div className="text-sm">
                    <div className="font-medium">Auto Provisioning</div>
                    <div className="text-muted-foreground">Automatic role assignment</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <Check className="h-4 w-4 text-green-600 mt-0.5" />
                  <div className="text-sm">
                    <div className="font-medium">Real-time Progress</div>
                    <div className="text-muted-foreground">WebSocket-based tracking</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <Check className="h-4 w-4 text-green-600 mt-0.5" />
                  <div className="text-sm">
                    <div className="font-medium">Error Handling</div>
                    <div className="text-muted-foreground">Detailed error reports</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </>
  )
}
