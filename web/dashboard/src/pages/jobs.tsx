import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Play, Pause, RotateCcw, Trash2, Plus, CheckCircle2, AlertCircle, Calendar } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { jobsApi } from '@/lib/api'

interface Job {
  id: string
  name: string
  description: string
  status: 'running' | 'scheduled' | 'completed' | 'failed' | 'paused'
  schedule: string
  last_run: string | null
  next_run: string | null
  run_count: number
  is_active: boolean
}

export function Jobs() {
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState<string>('all')

  const { data: jobs = [], isLoading } = useQuery<Job[]>({
    queryKey: ['jobs'],
    queryFn: jobsApi.list,
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      jobsApi.toggle(id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: jobsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
  })

  const filteredJobs = jobs.filter(job => {
    if (filter === 'all') return true
    if (filter === 'active') return job.status === 'running' || job.status === 'scheduled'
    return job.status === filter
  })

  const runningJobs = jobs.filter(j => j.status === 'running').length
  const failedJobs = jobs.filter(j => j.status === 'failed').length

  if (isLoading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="flex items-center gap-3 text-muted-foreground">
          <RotateCcw className="h-5 w-5 animate-spin" />
          <span>Loading jobs...</span>
        </div>
      </div>
    )
  }

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
            <div className="text-2xl font-bold">{jobs.length}</div>
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
                    {job.last_run && (
                      <span>Last: {job.last_run}</span>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <p className="text-lg font-semibold">{job.run_count.toLocaleString()}</p>
                    <p className="text-xs text-muted-foreground">Runs</p>
                  </div>

                  <div className="flex gap-1">
                    <Button 
                      variant="ghost" 
                      size="icon" 
                      className="h-8 w-8"
                      onClick={() => toggleMutation.mutate({ id: job.id, enabled: !job.is_active })}
                    >
                      {job.is_active ? (
                        <Pause className="h-4 w-4" />
                      ) : (
                        <Play className="h-4 w-4" />
                      )}
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <RotateCcw className="h-4 w-4" />
                    </Button>
                    <Button 
                      variant="ghost" 
                      size="icon" 
                      className="h-8 w-8 text-destructive"
                      onClick={() => deleteMutation.mutate(job.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {filteredJobs.length === 0 && (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <Calendar className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground">No jobs found</p>
          </div>
        </div>
      )}
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
