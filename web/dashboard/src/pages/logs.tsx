import { useState } from 'react'
import { FileText, AlertTriangle, Info, CheckCircle2, Clock, Filter, Download } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface LogEntry {
  id: string
  timestamp: string
  level: 'info' | 'warning' | 'error' | 'success'
  source: string
  message: string
}

const mockLogs: LogEntry[] = [
  { id: '1', timestamp: '2024-01-15 14:32:05', level: 'info', source: 'System', message: 'Server started successfully' },
  { id: '2', timestamp: '2024-01-15 14:32:10', level: 'success', source: 'Telegram', message: 'Bot connected' },
  { id: '3', timestamp: '2024-01-15 14:35:22', level: 'info', source: 'LLM', message: 'Model loaded' },
  { id: '4', timestamp: '2024-01-15 14:40:15', level: 'warning', source: 'Memory', message: 'High memory usage detected' },
  { id: '5', timestamp: '2024-01-15 14:45:30', level: 'info', source: 'Scheduler', message: 'Job executed' },
  { id: '6', timestamp: '2024-01-15 14:50:00', level: 'error', source: 'API', message: 'Failed to connect to external API' },
  { id: '7', timestamp: '2024-01-15 14:55:12', level: 'info', source: 'User', message: 'New conversation started' },
  { id: '8', timestamp: '2024-01-15 15:00:00', level: 'success', source: 'Skills', message: 'Skill loaded successfully' },
]

export function Logs() {
  const [filter, setFilter] = useState<string>('all')
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null)

  const filteredLogs = mockLogs.filter(log => {
    if (filter === 'all') return true
    return log.level === filter
  })

  const logCounts = {
    all: mockLogs.length,
    info: mockLogs.filter(l => l.level === 'info').length,
    warning: mockLogs.filter(l => l.level === 'warning').length,
    error: mockLogs.filter(l => l.level === 'error').length,
    success: mockLogs.filter(l => l.level === 'success').length,
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Activity Logs</h1>
          <p className="text-sm text-muted-foreground mt-1">View system activity and events</p>
        </div>
        <Button variant="outline">
          <Download className="mr-2 h-4 w-4" />
          Export
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
        {(['all', 'info', 'warning', 'error', 'success'] as const).map((level) => (
          <button
            key={level}
            onClick={() => setFilter(level)}
            className={`p-3 rounded-lg border transition-colors text-left ${
              filter === level
                ? 'bg-primary/10 border-primary/30'
                : 'bg-card border-border hover:bg-muted'
            }`}
          >
            <p className="text-xs text-muted-foreground capitalize">{level}</p>
            <p className="text-xl font-bold">{logCounts[level]}</p>
          </button>
        ))}
      </div>

      {/* Logs List */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="flex items-center gap-2 text-base">
                <FileText className="h-4 w-4" />
                Log Entries
              </CardTitle>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Filter className="h-4 w-4" />
                <span>{filteredLogs.length} entries</span>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y divide-border max-h-[500px] overflow-y-auto">
                {filteredLogs.map((log) => {
                  const LevelIcon = getLevelIcon(log.level)
                  return (
                    <button
                      key={log.id}
                      onClick={() => setSelectedLog(log)}
                      className={`w-full p-4 text-left transition-colors hover:bg-muted ${
                        selectedLog?.id === log.id ? 'bg-muted' : ''
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <div className={`p-1.5 rounded ${getLevelColor(log.level)}`}>
                          <LevelIcon className="h-3.5 w-3.5" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">{log.message}</p>
                          <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {log.timestamp}
                            </span>
                            <span className="px-1.5 py-0.5 rounded bg-muted">
                              {log.source}
                            </span>
                          </div>
                        </div>
                      </div>
                    </button>
                  )
                })}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Log Details */}
        <div>
          <Card className="h-full">
            <CardHeader>
              <CardTitle className="text-base">Details</CardTitle>
            </CardHeader>
            <CardContent>
              {selectedLog ? (
                <div className="space-y-4">
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Level</p>
                    <span className={`inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium ${getLevelColor(selectedLog.level)}`}>
                      {(() => {
                        const Icon = getLevelIcon(selectedLog.level)
                        return <Icon className="h-3 w-3" />
                      })()}
                      {selectedLog.level.charAt(0).toUpperCase() + selectedLog.level.slice(1)}
                    </span>
                  </div>

                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Timestamp</p>
                    <p className="text-sm">{selectedLog.timestamp}</p>
                  </div>

                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Source</p>
                    <span className="px-2 py-1 rounded bg-muted text-sm">
                      {selectedLog.source}
                    </span>
                  </div>

                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Message</p>
                    <p className="text-sm">{selectedLog.message}</p>
                  </div>
                </div>
              ) : (
                <div className="flex h-48 items-center justify-center">
                  <div className="text-center">
                    <FileText className="h-10 w-10 text-muted-foreground mx-auto mb-3" />
                    <p className="text-sm text-muted-foreground">Select a log to view details</p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function getLevelColor(level: LogEntry['level']) {
  const colors = {
    info: 'bg-primary/10 text-primary',
    warning: 'bg-amber-500/10 text-amber-500',
    error: 'bg-destructive/10 text-destructive',
    success: 'bg-accent/10 text-accent',
  }
  return colors[level]
}

function getLevelIcon(level: LogEntry['level']) {
  const icons = {
    info: Info,
    warning: AlertTriangle,
    error: AlertTriangle,
    success: CheckCircle2,
  }
  return icons[level]
}
