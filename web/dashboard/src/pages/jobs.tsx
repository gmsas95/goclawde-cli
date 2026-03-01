import { useState } from 'react'
import { Play, Pause, RotateCcw, Trash2, Plus, Clock, CheckCircle2, AlertCircle, Calendar } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
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
  { 
    id: '5', 
    name: 'Generate Reports', 
    description: 'Generate usage and performance reports',
    status: 'paused', 
    schedule: 'Monthly on 1st', 
    lastRun: '3 weeks ago', 
    nextRun: 'Paused',
    runCount: 12,
    successRate: 100
  },
]

const getStatusColor = (status: Job['status']) => {
  const colors = {
    running: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
    scheduled: 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30',
    completed: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
    failed: 'bg-red-500/20 text-red-400 border-red-500/30',
    paused: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
  }
  return colors[status]
}

const getStatusIcon = (status: Job['status']) => {
  const icons = {
    running: RotateCcw,
    scheduled: Calendar,
    completed: CheckCircle2,
    failed: AlertCircle,
    paused: Pause,
  }
  return icons[status]
}

export function Jobs() {
  const [jobs, setJobs] = useState<Job[]>(mockJobs)
  const [filter, setFilter] = useState<string>('all')

  const filteredJobs = jobs.filter(job => {
    if (filter === 'all') return true
    if (filter === 'active') return job.status === 'running' || job.status === 'scheduled'
    return job.status === filter
  })

  const toggleJobStatus = (jobId: string) => {
    setJobs(jobs.map(job => {
      if (job.id === jobId) {
        const newStatus = job.status === 'paused' ? 'scheduled' : 'paused'
        return { ...job, status: newStatus }
      }
      return job
    }))
  }

  const runningJobs = jobs.filter(j => j.status === 'running').length
  const scheduledJobs = jobs.filter(j => j.status === 'scheduled').length
  const failedJobs = jobs.filter(j => j.status === 'failed').length

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="relative">
        <div className="absolute -inset-1 bg-gradient-to-r from-amber-500/20 to-orange-500/20 rounded-2xl blur-xl opacity-50" />
        <div className="relative">
          <h1 className="text-4xl font-bold gradient-text-amber mb-2">Job Scheduler</h1>
          <p className="text-lg text-white/60">Manage automated tasks and scheduled jobs</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-blue-500/20 to-indigo-500/20 border border-blue-500/30">
            <Calendar className="h-6 w-6 text-blue-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Total Jobs</p>
            <p className="text-2xl font-bold text-white">{jobs.length}</p>
          </div>
        </Card>

        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-emerald-500/20 to-teal-500/20 border border-emerald-500/30">
            <RotateCcw className="h-6 w-6 text-emerald-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Running</p>
            <p className="text-2xl font-bold text-white">{runningJobs}</p>
          </div>
        </Card>

        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-cyan-500/20 to-blue-500/20 border border-cyan-500/30">
            <Clock className="h-6 w-6 text-cyan-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Scheduled</p>
            <p className="text-2xl font-bold text-white">{scheduledJobs}</p>
          </div>
        </Card>

        <Card variant="glass" className="p-4 flex items-center gap-4">
          <div className="p-3 rounded-lg bg-gradient-to-br from-red-500/20 to-orange-500/20 border border-red-500/30">
            <AlertCircle className="h-6 w-6 text-red-400" />
          </div>
          <div>
            <p className="text-sm text-white/60">Failed</p>
            <p className="text-2xl font-bold text-white">{failedJobs}</p>
          </div>
        </Card>
      </div>

      {/* Controls */}
      <div className="flex flex-col sm:flex-row justify-between gap-4">
        <div className="flex gap-2">
          {(['all', 'active', 'running', 'scheduled', 'failed', 'paused'] as const).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                filter === f
                  ? 'bg-amber-500/30 text-amber-300 border border-amber-500/50'
                  : 'text-white/60 hover:text-white hover:bg-white/5'
              }`}
            >
              {f.charAt(0).toUpperCase() + f.slice(1)}
            </button>
          ))}
        </div>
        <Button variant="default" className="gap-2">
          <Plus className="h-4 w-4" />
          New Job
        </Button>
      </div>

      {/* Jobs List */}
      <div className="space-y-3">
        {filteredJobs.map((job) => {
          const StatusIcon = getStatusIcon(job.status)
          return (
            <Card key={job.id} variant="gradient" className="group hover:border-amber-500/30 transition-all">
              <CardContent className="p-5">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 mb-2">
                      <h3 className="text-lg font-semibold text-white truncate">{job.name}</h3>
                      <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${getStatusColor(job.status)}`}>
                        <span className="flex items-center gap-1">
                          <StatusIcon className="h-3 w-3" />
                          {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
                        </span>
                      </span>
                    </div>
                    <p className="text-sm text-white/50 mb-3">{job.description}</p>
                    
                    <div className="flex flex-wrap gap-4 text-sm">
                      <div className="flex items-center gap-2 text-white/40">
                        <Calendar className="h-4 w-4" />
                        <span>{job.schedule}</span>
                      </div>
                      {job.lastRun && (
                        <div className="flex items-center gap-2 text-white/40">
                          <RotateCcw className="h-4 w-4" />
                          <span>Last: {job.lastRun}</span>
                        </div>
                      )}
                      {job.nextRun && (
                        <div className="flex items-center gap-2 text-cyan-400/70">
                          <Clock className="h-4 w-4" />
                          <span>Next: {job.nextRun}</span>
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="flex flex-col items-end gap-3">
                    <div className="text-right">
                      <p className="text-2xl font-bold text-white">{job.runCount.toLocaleString()}</p>
                      <p className="text-xs text-white/40">Total runs</p>
                    </div>
                    <div className="text-right">
                      <p className={`text-lg font-semibold ${job.successRate >= 95 ? 'text-emerald-400' : job.successRate >= 90 ? 'text-amber-400' : 'text-red-400'}`}>
                        {job.successRate}%
                      </p>
                      <p className="text-xs text-white/40">Success rate</p>
                    </div>
                  </div>

                  <div className="flex flex-col gap-1">
                    <Button 
                      variant="ghost" 
                      size="sm" 
                      className="h-8 w-8 p-0"
                      onClick={() => toggleJobStatus(job.id)}
                    >
                      {job.status === 'paused' ? (
                        <Play className="h-4 w-4 text-emerald-400" />
                      ) : (
                        <Pause className="h-4 w-4 text-amber-400" />
                      )}
                    </Button>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                      <RotateCcw className="h-4 w-4 text-cyan-400" />
                    </Button>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0 text-red-400 hover:text-red-300">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {filteredJobs.length === 0 && (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <Calendar className="h-12 w-12 text-white/20 mx-auto mb-4" />
            <p className="text-white/40">No jobs found</p>
          </div>
        </div>
      )}
    </div>
  )
}
