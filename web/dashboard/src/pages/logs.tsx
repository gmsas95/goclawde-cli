import { useState } from 'react'
import { FileText, AlertTriangle, Info, CheckCircle2, Clock, Filter, Download, Trash2 } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface LogEntry {
  id: string
  timestamp: string
  level: 'info' | 'warning' | 'error' | 'success'
  source: string
  message: string
  details?: string
}

const mockLogs: LogEntry[] = [
  { id: '1', timestamp: '2024-01-15 14:32:05', level: 'info', source: 'System', message: 'Server started successfully', details: 'Port: 8080, Address: 0.0.0.0' },
  { id: '2', timestamp: '2024-01-15 14:32:10', level: 'success', source: 'Telegram', message: 'Bot connected', details: 'Bot ID: @myrai_bot' },
  { id: '3', timestamp: '2024-01-15 14:35:22', level: 'info', source: 'LLM', message: 'Model loaded', details: 'Provider: kimi, Model: kimi-k2.5' },
  { id: '4', timestamp: '2024-01-15 14:40:15', level: 'warning', source: 'Memory', message: 'High memory usage detected', details: 'Usage: 85%, Threshold: 80%' },
  { id: '5', timestamp: '2024-01-15 14:45:30', level: 'info', source: 'Scheduler', message: 'Job executed', details: 'Job: Health Check, Duration: 120ms' },
  { id: '6', timestamp: '2024-01-15 14:50:00', level: 'error', source: 'API', message: 'Failed to connect to external API', details: 'Error: Timeout after 30s, Endpoint: /v1/chat' },
  { id: '7', timestamp: '2024-01-15 14:55:12', level: 'info', source: 'User', message: 'New conversation started', details: 'Session ID: sess_123456' },
  { id: '8', timestamp: '2024-01-15 15:00:00', level: 'success', source: 'Skills', message: 'Skill loaded successfully', details: 'Skill: weather, Tools: 3' },
  { id: '9', timestamp: '2024-01-15 15:05:45', level: 'info', source: 'Database', message: 'Query executed', details: 'Duration: 45ms, Rows: 12' },
  { id: '10', timestamp: '2024-01-15 15:10:20', level: 'warning', source: 'Cache', message: 'Cache miss rate high', details: 'Miss rate: 35%, Recommended: <20%' },
  { id: '11', timestamp: '2024-01-15 15:15:00', level: 'info', source: 'Config', message: 'Configuration updated', details: 'Updated: telegram settings' },
  { id: '12', timestamp: '2024-01-15 15:20:30', level: 'success', source: 'Export', message: 'Data exported successfully', details: 'File: backup_20240115.zip, Size: 2.4MB' },
]

const getLevelColor = (level: LogEntry['level']) => {
  const colors = {
    info: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
    warning: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
    error: 'bg-red-500/20 text-red-400 border-red-500/30',
    success: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
  }
  return colors[level]
}

const getLevelIcon = (level: LogEntry['level']) => {
  const icons = {
    info: Info,
    warning: AlertTriangle,
    error: AlertTriangle,
    success: CheckCircle2,
  }
  return icons[level]
}

export function Logs() {
  const [logs] = useState<LogEntry[]>(mockLogs)
  const [filter, setFilter] = useState<string>('all')
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null)

  const filteredLogs = logs.filter(log => {
    if (filter === 'all') return true
    return log.level === filter
  })

  const logCounts = {
    all: logs.length,
    info: logs.filter(l => l.level === 'info').length,
    warning: logs.filter(l => l.level === 'warning').length,
    error: logs.filter(l => l.level === 'error').length,
    success: logs.filter(l => l.level === 'success').length,
  }

  const clearLogs = () => {
    // TODO: Clear logs via API
  }

  const exportLogs = () => {
    const dataStr = JSON.stringify(logs, null, 2)
    const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
    const exportFileDefaultName = `logs_${new Date().toISOString().split('T')[0]}.json`
    const linkElement = document.createElement('a')
    linkElement.setAttribute('href', dataUri)
    linkElement.setAttribute('download', exportFileDefaultName)
    linkElement.click()
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="relative">
        <div className="absolute -inset-1 bg-gradient-to-r from-slate-500/20 to-slate-400/20 rounded-2xl blur-xl opacity-50" />
        <div className="relative flex items-center justify-between">
          <div>
            <h1 className="text-4xl font-bold gradient-text mb-2">Activity Logs</h1>
            <p className="text-lg text-white/60">View system activity and events</p>
          </div>
          <div className="flex gap-2">
            <Button variant="glass" className="gap-2" onClick={exportLogs}>
              <Download className="h-4 w-4" />
              Export
            </Button>
            <Button variant="glass" className="gap-2 text-red-400 hover:text-red-300" onClick={clearLogs}>
              <Trash2 className="h-4 w-4" />
              Clear
            </Button>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        {(['all', 'info', 'warning', 'error', 'success'] as const).map((level) => (
          <button
            key={level}
            onClick={() => setFilter(level)}
            className={`p-4 rounded-xl border transition-all text-left ${
              filter === level
                ? 'bg-white/10 border-white/30'
                : 'bg-white/5 border-white/10 hover:bg-white/10'
            }`}
          >
            <p className="text-sm text-white/60 capitalize">{level}</p>
            <p className="text-2xl font-bold text-white">{logCounts[level]}</p>
          </button>
        ))}
      </div>

      {/* Logs List */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <Card variant="gradient" className="overflow-hidden">
            <div className="p-4 border-b border-white/5 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-gradient-to-br from-slate-500/20 to-slate-400/20 border border-slate-500/30">
                  <FileText className="h-4 w-4 text-slate-400" />
                </div>
                <h3 className="font-semibold text-white">Log Entries</h3>
              </div>
              <div className="flex items-center gap-2 text-sm text-white/40">
                <Filter className="h-4 w-4" />
                <span>Showing {filteredLogs.length} entries</span>
              </div>
            </div>
            <CardContent className="p-0">
              <div className="divide-y divide-white/5 max-h-[600px] overflow-y-auto">
                {filteredLogs.map((log) => {
                  const LevelIcon = getLevelIcon(log.level)
                  return (
                    <button
                      key={log.id}
                      onClick={() => setSelectedLog(log)}
                      className={`w-full p-4 text-left transition-all hover:bg-white/5 ${
                        selectedLog?.id === log.id ? 'bg-white/10' : ''
                      }`}
                    >
                      <div className="flex items-start gap-4">
                        <div className={`p-2 rounded-lg border ${getLevelColor(log.level)}`}>
                          <LevelIcon className="h-4 w-4" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="text-sm font-medium text-white truncate">{log.message}</span>
                          </div>
                          <div className="flex items-center gap-3 text-xs text-white/40">
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {log.timestamp}
                            </span>
                            <span className="px-2 py-0.5 rounded-full bg-white/5">
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
          <Card variant="gradient" className="h-full">
            <div className="p-4 border-b border-white/5">
              <h3 className="font-semibold text-white">Log Details</h3>
            </div>
            <CardContent className="p-4">
              {selectedLog ? (
                <div className="space-y-4">
                  <div>
                    <p className="text-sm text-white/40 mb-1">Level</p>
                    <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-lg text-sm border ${getLevelColor(selectedLog.level)}`}>
                      {(() => {
                        const Icon = getLevelIcon(selectedLog.level)
                        return <Icon className="h-3 w-3" />
                      })()}
                      {selectedLog.level.charAt(0).toUpperCase() + selectedLog.level.slice(1)}
                    </span>
                  </div>

                  <div>
                    <p className="text-sm text-white/40 mb-1">Timestamp</p>
                    <p className="text-white">{selectedLog.timestamp}</p>
                  </div>

                  <div>
                    <p className="text-sm text-white/40 mb-1">Source</p>
                    <span className="px-2 py-1 rounded-lg bg-white/10 text-white text-sm">
                      {selectedLog.source}
                    </span>
                  </div>

                  <div>
                    <p className="text-sm text-white/40 mb-1">Message</p>
                    <p className="text-white">{selectedLog.message}</p>
                  </div>

                  {selectedLog.details && (
                    <div>
                      <p className="text-sm text-white/40 mb-1">Details</p>
                      <pre className="p-3 rounded-lg bg-slate-950/50 text-sm text-white/70 overflow-x-auto">
                        {selectedLog.details}
                      </pre>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex h-64 items-center justify-center">
                  <div className="text-center">
                    <FileText className="h-12 w-12 text-white/20 mx-auto mb-4" />
                    <p className="text-white/40">Select a log entry to view details</p>
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
