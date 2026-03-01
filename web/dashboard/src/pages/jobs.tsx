import { useState } from 'react'
import { Play, Pause, RotateCcw, Trash2, Plus, CheckCircle2, AlertCircle, Calendar } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface Job {
  id: string
  name: string
  description: string
  status: 'running' | 'scheduled' | 'completed' | 'failed' | 'paused'
  schedule: string
  lastRun: string | null
  nextRun: string | null
  runCount: number
  successRate: number
}

const mockJobs: Job[] = [
  { 
    id: '1', 
    name: 'Health Check', 
    description: 'Monitor system health and resources',
    status: 'running', 
    schedule: 'Every 5 minutes', 
    lastRun: '3 min ago', 
    nextRun: 'In 2 min',
    runCount: 1247,
    successRate: 99.9
  },
  { 
    id: '2', 
    name: 'Memory Cleanup', 
    description: 'Clean up expired and low-importance memories',
    status: 'scheduled', 
    schedule: 'Daily at 2 AM', 
    lastRun: '22 hours ago', 
    nextRun: 'In 2 hours',
    runCount: 365,
    successRate: 98.5
  },
  { 
    id: '3', 
    name: 'Backup Data', 
    description: 'Backup conversation and configuration data',
    status: 'completed', 
    schedule: 'Weekly on Sunday', 
    lastRun: '5 days ago', 
    nextRun: 'In 2 days',
    runCount: 52,
    successRate: 100
  },
  { 
    id: '4', 
    name: 'Sync External APIs', 
    description: 'Synchronize data with external services',
    status: 'failed', 
    schedule: 'Every hour', 
    lastRun: '1 hour ago', 
    nextRun: 'Now',
    runCount: 8760,
    successRate: 94.2
  },
]

export function Jobs() {
  const [filter, setFilter] = useState<string>('all')

  const filteredJobs = mockJobs.filter(job => {
    if (filter === 'all') return true
    if (filter === 'active') return job.status === 'running' || job.status === 'scheduled'
    return job.status === filter
  })

  const runningJobs = mockJobs.filter(j => j.status === 'running').length
  const failedJobs = mockJobs.filter(j => j.status === 'failed').length

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Job Scheduler</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage automated tasks</p>
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          New Job
        </Button>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Jobs</CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{mockJobs.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Running</CardTitle>
            <RotateCcw className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{runningJobs}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Failed</CardTitle>
            <AlertCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{failedJobs}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Success Rate</CardTitle>
            <CheckCircle2 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">98.2%</div>
          </CardContent>
        </Card>
      </div>

      {/* Filter */}
      <div className="flex gap-2">
        {(['all', 'active', 'running', 'failed'] as const).map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
              filter === f
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            }`}
          >
            {f.charAt(0).toUpperCase() + f.slice(1)}
          </button>
        ))}
      </div>

      {/* Jobs List */}
      <div className="space-y-3">
        {filteredJobs.map((job) => (
          <Card key={job.id}>
            <CardContent className="p-5">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <h3 className="font-medium">{job.name}</h3>
                    <StatusBadge status={job.status} />
                  </div>
                  <p className="text-sm text-muted-foreground mb-2">{job.description}</p>
                  
                  <div className="flex flex-wrap gap-3 text-sm text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Calendar className="h-3.5 w-3.5" />
                      {job.schedule}
                    </span>
                    {job.lastRun && (
                      <span>Last: {job.lastRun}</span>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <p className="text-lg font-semibold">{job.runCount.toLocaleString()}</p>
                    <p className="text-xs text-muted-foreground">Runs</p>
                  </div>
                  
                  <div className="text-right">
                    <p className="text-lg font-semibold text-accent">{job.successRate}%</p>
                    <p className="text-xs text-muted-foreground">Success</p>
                  </div>

                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      {job.status === 'paused' ? (
                        <Play className="h-4 w-4" />
                      ) : (
                        <Pause className="h-4 w-4" />
                      )}
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <RotateCcw className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: Job['status'] }) {
  const styles = {
    running: 'bg-accent/10 text-accent',
    scheduled: 'bg-primary/10 text-primary',
    completed: 'bg-muted text-muted-foreground',
    failed: 'bg-destructive/10 text-destructive',
    paused: 'bg-muted text-muted-foreground',
  }

  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${styles[status]}`}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}
